package sync

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
)

type FileMetadata struct {
	DriveID string `json:"drive_id"`
	ModTime int64  `json:"mod_time"`
	Hash    string `json:"hash"`
}

type GDriveProvider struct {
	srv         *drive.Service
	folderCache map[string]string       // path -> fileId
	fileCache   map[string]FileMetadata // localPath -> metadata
	stateFile   string
}

func NewGDriveProvider() (*GDriveProvider, error) {
	ctx := context.Background()

	// Assume credentials.json is in the config dir
	configDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home dir: %w", err)
	}
	credentialsPath := filepath.Join(configDir, ".config", "aerosync", "credentials.json")

	b, err := ioutil.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read client secret file: %v", err)
	}

	config, err := google.ConfigFromJSON(b, drive.DriveFileScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %v", err)
	}

	tokenPath := filepath.Join(configDir, ".config", "aerosync", "token.json")
	token, err := tokenFromFile(tokenPath)
	if err != nil {
		token = getTokenFromWeb(config)
		saveToken(tokenPath, token)
	}

	client := config.Client(ctx, token)
	srv, err := drive.New(client)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Drive client: %v", err)
	}

	// Load state file
	statePath := filepath.Join(configDir, ".config", "aerosync", "state.json")
	fileCache := make(map[string]FileMetadata)
	if data, err := ioutil.ReadFile(statePath); err == nil {
		json.Unmarshal(data, &fileCache)
	}

	return &GDriveProvider{
		srv:         srv,
		folderCache: make(map[string]string),
		fileCache:   fileCache,
		stateFile:   statePath,
	}, nil
}

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

// getOrCreateFolder gets or creates a folder in Google Drive and returns its ID
func (g *GDriveProvider) getOrCreateFolder(ctx context.Context, folderPath string) (string, error) {
	if id, exists := g.folderCache[folderPath]; exists {
		return id, nil
	}

	// Handle root
	if folderPath == "." {
		return "", nil
	}

	// Recursively create parent folders
	parentPath := filepath.Dir(folderPath)
	parentId := ""
	if parentPath != "." {
		var err error
		parentId, err = g.getOrCreateFolder(ctx, parentPath)
		if err != nil {
			return "", err
		}
	}

	folderName := filepath.Base(folderPath)

	// Check if folder already exists
	query := fmt.Sprintf("name='%s' and mimeType='application/vnd.google-apps.folder'", folderName)
	if parentId != "" {
		query += fmt.Sprintf(" and '%s' in parents", parentId)
	} else {
		query += " and 'root' in parents"
	}

	r, err := g.srv.Files.List().Q(query).Fields("files(id)").Do()
	if err != nil {
		return "", err
	}

	if len(r.Files) > 0 {
		id := r.Files[0].Id
		g.folderCache[folderPath] = id
		return id, nil
	}

	// Create new folder
	folder := &drive.File{
		Name:     folderName,
		MimeType: "application/vnd.google-apps.folder",
	}
	if parentId != "" {
		folder.Parents = []string{parentId}
	}

	created, err := g.srv.Files.Create(folder).Do()
	if err != nil {
		return "", err
	}

	g.folderCache[folderPath] = created.Id
	return created.Id, nil
}

// saveState saves the file cache to disk
func (g *GDriveProvider) saveState() error {
	data, err := json.MarshalIndent(g.fileCache, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(g.stateFile, data, 0644)
}

// computeHash computes MD5 hash of a file
func computeHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (g *GDriveProvider) Sync(ctx context.Context, path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	// Walk the path and upload files/directories
	return filepath.Walk(absPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path from sync root
		relPath, err := filepath.Rel(absPath, filePath)
		if err != nil {
			return err
		}
		if relPath == "." {
			// Skip the root directory itself
			return nil
		}

		if info.IsDir() {
			// Create folder in Drive
			_, err = g.getOrCreateFolder(ctx, relPath)
			return err
		}

		// Check if file needs to be uploaded/updated
		hash, err := computeHash(filePath)
		if err != nil {
			return err
		}

		modTime := info.ModTime().Unix()
		existing, exists := g.fileCache[filePath]

		if exists && existing.Hash == hash && existing.ModTime == modTime {
			// File unchanged, skip
			return nil
		}

		// File changed or new, upload/update
		file, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer file.Close()

		// Get parent folder ID
		parentId := ""
		if relDir := filepath.Dir(relPath); relDir != "." {
			parentId, err = g.getOrCreateFolder(ctx, relDir)
			if err != nil {
				return err
			}
		}

		var driveFileId string
		if exists {
			// Update existing file
			_, err = g.srv.Files.Update(existing.DriveID, &drive.File{}).Media(file).Do()
			if err != nil {
				return err
			}
			driveFileId = existing.DriveID
			log.Printf("Updated %s in Google Drive", filePath)
		} else {
			// Create new file
			driveFile := &drive.File{
				Name: filepath.Base(filePath),
			}
			if parentId != "" {
				driveFile.Parents = []string{parentId}
			}

			created, err := g.srv.Files.Create(driveFile).Media(file).Do()
			if err != nil {
				return err
			}
			driveFileId = created.Id
			log.Printf("Uploaded %s to Google Drive", filePath)
		}

		// Update cache
		g.fileCache[filePath] = FileMetadata{
			DriveID: driveFileId,
			ModTime: modTime,
			Hash:    hash,
		}
		return nil
	})

	// Save state
	return g.saveState()
}

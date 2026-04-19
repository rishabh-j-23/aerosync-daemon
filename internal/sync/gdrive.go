package sync

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type FileMetadata struct {
	DriveID string
	ModTime int64
	Hash    string
}

type GDriveProvider struct {
	srv         *drive.Service
	db          *sql.DB
	folderCache map[string]string // path -> driveId
	mu          sync.RWMutex
}

func NewGDriveProvider() (*GDriveProvider, error) {
	ctx := context.Background()

	configDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home dir: %w", err)
	}
	credentialsPath := filepath.Join(configDir, ".config", "aerosync", "credentials.json")

	b, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read client secret file: %v", err)
	}

	config, err := google.ConfigFromJSON(b, drive.DriveFileScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %v", err)
	}

	tokenPath := filepath.Join(configDir, ".config", "aerosync", "token.json")
	token, err := tokenFromFile(tokenPath)
	if err != nil || token.Expiry.Before(time.Now()) {
		token = getTokenFromWeb(config)
		saveToken(tokenPath, token)
	}

	client := config.Client(ctx, token)
	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Drive client: %v", err)
	}

	// Setup SQLite DB
	dbPath := filepath.Join(configDir, ".config", "aerosync", "metadata.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open DB: %v", err)
	}

	// Create tables if not exists
	if err := setupDatabase(db); err != nil {
		return nil, err
	}

	// Load folder cache from DB
	folderCache := make(map[string]string)
	rows, err := db.Query("SELECT path, drive_id FROM folders")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var path, driveId string
			rows.Scan(&path, &driveId)
			folderCache[path] = driveId
		}
	}

	return &GDriveProvider{
		srv:         srv,
		db:          db,
		folderCache: folderCache,
	}, nil
}

func setupDatabase(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS files (
			path TEXT PRIMARY KEY,
			drive_id TEXT,
			mod_time INTEGER,
			hash TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS folders (
			path TEXT PRIMARY KEY,
			drive_id TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS versions (
			file_path TEXT,
			version_num INTEGER,
			drive_id TEXT,
			timestamp INTEGER,
			PRIMARY KEY (file_path, version_num)
		)`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to create table: %v", err)
		}
	}
	return nil
}

func (g *GDriveProvider) Sync(ctx context.Context, path string, label string, versioning bool, maxVersions int, ignore []string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	baseName := filepath.Base(absPath)
	displayFolder := baseName
	if label != "" {
		displayFolder = label
	}
	log.Printf("Starting sync for path: %s (Label: %s), versioning: %v", absPath, displayFolder, versioning)

	return filepath.Walk(absPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(absPath, filePath)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		if g.shouldIgnore(relPath, ignore) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			folderPath := "aerosync/" + displayFolder + "/" + relPath
			_, err = g.getOrCreateFolder(ctx, folderPath)
			return err
		}

		hash, err := computeHash(filePath)
		if err != nil {
			return err
		}

		modTime := info.ModTime().Unix()
		existing, exists, err := g.getMetadata(filePath)
		if err != nil {
			return err
		}

		// Verify the file still exists in Drive
		if exists {
			_, err := g.srv.Files.Get(existing.DriveID).Fields("id").Do()
			if err != nil {
				log.Printf("File %s not found in Drive (ID: %s), treating as new: %v", filePath, existing.DriveID, err)
				// File not found in Drive, treat as not existing
				exists = false
				// Remove stale metadata
				g.db.Exec("DELETE FROM files WHERE path = ?", filePath)
			}
		}

		if exists && existing.Hash == hash && existing.ModTime == modTime {
			return nil
		}

		file, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer file.Close()

		parentId := ""
		if relDir := filepath.Dir(relPath); relDir != "." {
			folderPath := "aerosync/" + displayFolder + "/" + relDir
			parentId, err = g.getOrCreateFolder(ctx, folderPath)
			if err != nil {
				return err
			}
		} else {
			folderPath := "aerosync/" + displayFolder
			parentId, err = g.getOrCreateFolder(ctx, folderPath)
			if err != nil {
				return err
			}
		}

		var driveFileId string
		if exists && !versioning {
			// Update existing
			_, err = g.srv.Files.Update(existing.DriveID, &drive.File{}).Media(file).Do()
			if err != nil {
				return err
			}
			driveFileId = existing.DriveID
			log.Printf("Updated %s in Google Drive", filePath)
		} else {
			// Archive old version if versioning
			if exists && versioning {
				log.Printf("Versioning enabled for changed file: %s", filePath)
				versionsPath := "aerosync_versions/" + displayFolder + "/" + relPath
				if err := g.archiveFile(ctx, existing.DriveID, versionsPath, filePath); err != nil {
					log.Printf("Failed to archive %s: %v", filePath, err)
				}
				// Cleanup old versions
				if err := g.cleanupVersions(maxVersions, filePath); err != nil {
					log.Printf("Failed to cleanup versions for %s: %v", filePath, err)
				}
			}

			// Create new file
			driveFile := &drive.File{Name: filepath.Base(filePath)}
			if parentId != "" {
				driveFile.Parents = []string{parentId}
			}

			// Detect MimeType
			preview := make([]byte, 512)
			n, _ := file.ReadAt(preview, 0)
			mimeType := http.DetectContentType(preview[:n])
			if strings.HasSuffix(filePath, ".toml") {
				mimeType = "text/plain"
			}
			driveFile.MimeType = mimeType

			created, err := g.srv.Files.Create(driveFile).Media(file).Do()
			if err != nil {
				return err
			}
			driveFileId = created.Id
			if exists && versioning {
				log.Printf("Versioned %s in Google Drive (check aerosync_versions folder)", filePath)
			} else {
				folderPath := "aerosync/" + displayFolder
				if relDir := filepath.Dir(relPath); relDir != "." {
					folderPath += "/" + relDir
				}
				log.Printf("Uploaded %s to Google Drive folder: %s", filePath, folderPath)
			}
		}

		err = g.setMetadata(filePath, FileMetadata{
			DriveID: driveFileId,
			ModTime: modTime,
			Hash:    hash,
		})
		return err
	})
}

func (g *GDriveProvider) Restore(ctx context.Context, targetPath string, relPath string, label string, baseName string) error {
	fileName := filepath.Base(relPath)
	relDir := filepath.Dir(relPath)

	displayFolder := baseName
	if label != "" {
		displayFolder = label
	}

	// First, try to find current version in correct folder
	var parentId string
	if relDir != "." {
		folderPath := fmt.Sprintf("aerosync/%s/%s", displayFolder, relDir)
		var err error
		parentId, err = g.getOrCreateFolder(ctx, folderPath)
		if err != nil {
			return err
		}
	} else {
		folderPath := fmt.Sprintf("aerosync/%s", displayFolder)
		var err error
		parentId, err = g.getOrCreateFolder(ctx, folderPath)
		if err != nil {
			return err
		}
	}

	safeFileName := strings.ReplaceAll(fileName, "'", "\\'")
	query := fmt.Sprintf("name='%s' and '%s' in parents and trashed=false", safeFileName, parentId)
	if relPath == "." {
		// Special case: restoring the root of the label/backup
		return g.restoreFolder(ctx, parentId, targetPath)
	}

	files, err := g.srv.Files.List().Q(query).Fields("files(id,name,modifiedTime,mimeType)").OrderBy("modifiedTime desc").Do()
	if err != nil {
		return err
	}

	if len(files.Files) == 0 {
		return fmt.Errorf("file not found in Drive: %s", relPath)
	}

	driveFile := files.Files[0]

	if driveFile.MimeType == "application/vnd.google-apps.folder" {
		return g.restoreFolder(ctx, driveFile.Id, targetPath)
	}

	return g.downloadFile(ctx, driveFile.Id, targetPath)
}

func (g *GDriveProvider) restoreFolder(ctx context.Context, folderId string, targetPath string) error {
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		return err
	}

	query := fmt.Sprintf("'%s' in parents and trashed=false", folderId)
	r, err := g.srv.Files.List().Q(query).Fields("files(id,name,mimeType)").Do()
	if err != nil {
		return err
	}

	for _, f := range r.Files {
		dest := filepath.Join(targetPath, f.Name)
		if f.MimeType == "application/vnd.google-apps.folder" {
			if err := g.restoreFolder(ctx, f.Id, dest); err != nil {
				return err
			}
		} else {
			if err := g.downloadFile(ctx, f.Id, dest); err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *GDriveProvider) downloadFile(ctx context.Context, fileId, targetPath string) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}
	
	resp, err := g.srv.Files.Get(fileId).Download()
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Write to local file
	out, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func (g *GDriveProvider) getOrCreateFolder(ctx context.Context, folderPath string) (string, error) {
	// Normalize path to use forward slashes for cache keys
	folderPath = filepath.ToSlash(folderPath)

	g.mu.RLock()
	id, exists := g.folderCache[folderPath]
	g.mu.RUnlock()
	if exists {
		// Verify folder still exists in Drive
		_, err := g.srv.Files.Get(id).Fields("id").Do()
		if err == nil {
			return id, nil
		}
		// If 404, remove from cache and continue to recreate
		log.Printf("Cached folder ID %s for path %s not found in Drive, refreshing cache...", id, folderPath)
		g.mu.Lock()
		delete(g.folderCache, folderPath)
		g.mu.Unlock()
		g.db.Exec("DELETE FROM folders WHERE path = ?", folderPath)
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
	safeName := strings.ReplaceAll(folderName, "'", "\\'")
	query := fmt.Sprintf("name='%s' and mimeType='application/vnd.google-apps.folder' and trashed=false", safeName)
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
		g.setFolder(folderPath, id)
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

	g.setFolder(folderPath, created.Id)
	return created.Id, nil
}

func (g *GDriveProvider) getMetadata(path string) (FileMetadata, bool, error) {
	var meta FileMetadata
	err := g.db.QueryRow("SELECT drive_id, mod_time, hash FROM files WHERE path = ?", path).Scan(&meta.DriveID, &meta.ModTime, &meta.Hash)
	if err == sql.ErrNoRows {
		return meta, false, nil
	}
	if err != nil {
		return meta, false, err
	}
	return meta, true, nil
}

func (g *GDriveProvider) setMetadata(path string, meta FileMetadata) error {
	_, err := g.db.Exec("INSERT OR REPLACE INTO files (path, drive_id, mod_time, hash) VALUES (?, ?, ?, ?)",
		path, meta.DriveID, meta.ModTime, meta.Hash)
	return err
}

func (g *GDriveProvider) setFolder(path, driveId string) error {
	path = filepath.ToSlash(path)
	_, err := g.db.Exec("INSERT OR REPLACE INTO folders (path, drive_id) VALUES (?, ?)", path, driveId)
	if err == nil {
		g.mu.Lock()
		g.folderCache[path] = driveId
		g.mu.Unlock()
	}
	return err
}

func (g *GDriveProvider) archiveFile(ctx context.Context, fileId, versionsPath string, filePath string) error {
	log.Printf("Archiving file %s to versions path %s", filePath, versionsPath)

	versionsDir := filepath.Dir(versionsPath)
	versionsFolderId, err := g.getVersionsFolder(ctx, versionsDir)
	if err != nil {
		log.Printf("Failed to get versions folder for %s: %v", versionsDir, err)
		return err
	}
	log.Printf("Versions folder ID: %s for path: %s", versionsFolderId, versionsDir)

	// Get next version number
	var maxVersion int
	err = g.db.QueryRow("SELECT COALESCE(MAX(version_num), 0) FROM versions WHERE file_path = ?", filePath).Scan(&maxVersion)
	if err != nil {
		log.Printf("Failed to get max version for %s: %v", filePath, err)
		return err
	}
	versionNum := maxVersion + 1
	log.Printf("Creating version %d for %s", versionNum, filePath)

	timestamp := time.Now().Unix()
	newName := fmt.Sprintf("%s_v%d_%d", filepath.Base(versionsPath), versionNum, timestamp)

	// Move file to versions folder
	// In Drive API v3, moving requires addParents/removeParents
	file, err := g.srv.Files.Get(fileId).Fields("parents").Do()
	if err != nil {
		log.Printf("Failed to get current parents for %s: %v", fileId, err)
		return err
	}
	oldParents := strings.Join(file.Parents, ",")

	_, err = g.srv.Files.Update(fileId, &drive.File{Name: newName}).
		AddParents(versionsFolderId).
		RemoveParents(oldParents).
		Do()
	if err != nil {
		log.Printf("Failed to move file %s to versions: %v", fileId, err)
		return err
	}

	log.Printf("Archived file %s as %s in folder %s", filePath, newName, versionsFolderId)

	// Record in DB
	_, err = g.db.Exec("INSERT INTO versions (file_path, version_num, drive_id, timestamp) VALUES (?, ?, ?, ?)",
		filePath, versionNum, fileId, timestamp)
	if err != nil {
		log.Printf("Failed to record version in DB: %v", err)
	}
	return err
}

func (g *GDriveProvider) cleanupVersions(maxVersions int, filePath string) error {
	// Get versions from DB
	rows, err := g.db.Query("SELECT version_num, drive_id FROM versions WHERE file_path = ? ORDER BY version_num DESC", filePath)
	if err != nil {
		return err
	}
	defer rows.Close()

	var versions []struct {
		num     int
		driveId string
	}
	for rows.Next() {
		var v struct {
			num     int
			driveId string
		}
		rows.Scan(&v.num, &v.driveId)
		versions = append(versions, v)
	}

	// Keep only maxVersions, delete older
	if len(versions) > maxVersions {
		for i := maxVersions; i < len(versions); i++ {
			// Delete from Drive
			g.srv.Files.Delete(versions[i].driveId).Do()
			// Delete from DB
			g.db.Exec("DELETE FROM versions WHERE file_path = ? AND version_num = ?", filePath, versions[i].num)
		}
	}
	return nil
}

func (g *GDriveProvider) getVersionsFolder(ctx context.Context, versionsPath string) (string, error) {
	return g.getOrCreateFolder(ctx, versionsPath)
}

func (g *GDriveProvider) shouldIgnore(relPath string, ignore []string) bool {
	relPath = filepath.ToSlash(relPath)
	for _, pattern := range ignore {
		// Match the full relative path
		if matched, _ := filepath.Match(pattern, relPath); matched {
			return true
		}
		// Match the filename only (e.g. *.log)
		if matched, _ := filepath.Match(pattern, filepath.Base(relPath)); matched {
			return true
		}
		// Match parent directories (e.g. .git)
		parts := strings.Split(relPath, "/")
		for i := range parts {
			if matched, _ := filepath.Match(pattern, parts[i]); matched {
				return true
			}
		}
	}
	return false
}

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

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then authorize the app: \n%v\n", authURL)

	// Check if redirect URI supports localhost (automatic handling)
	if config.RedirectURL != "" && strings.Contains(config.RedirectURL, "localhost") {
		// Handle localhost redirect automatically
		codeCh := make(chan string, 1)
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			code := r.URL.Query().Get("code")
			if code != "" {
				fmt.Fprintf(w, "<h1>Authorization successful!</h1><p>You can close this window.</p>")
				codeCh <- code
			}
		})
		go func() {
			log.Println("Starting local server on :8080 for OAuth redirect...")
			if err := http.ListenAndServe(":8080", nil); err != nil {
				log.Printf("Local server error: %v", err)
			}
		}()
		log.Println("Waiting for authorization...")
		authCode := <-codeCh
		log.Printf("Received authorization code")
		tok, err := config.Exchange(context.TODO(), authCode)
		if err != nil {
			log.Fatalf("Unable to retrieve token from web %v", err)
		}
		return tok
	} else {
		// Manual code entry (OOB)
		fmt.Println("After authorization, copy the authorization code from the browser and paste it here:")
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
	// Simple JSON encode for token
	data, _ := json.Marshal(token)
	f.Write(data)
}

func (g *GDriveProvider) Cleanup(ctx context.Context) error {
	log.Println("Cleaning up local sync data...")
	// Clear local DB tables
	_, err := g.db.Exec("DELETE FROM files")
	if err != nil {
		return fmt.Errorf("failed to clear files table: %w", err)
	}
	_, err = g.db.Exec("DELETE FROM folders")
	if err != nil {
		return fmt.Errorf("failed to clear folders table: %w", err)
	}
	_, err = g.db.Exec("DELETE FROM versions")
	if err != nil {
		return fmt.Errorf("failed to clear versions table: %w", err)
	}

	// Clear cache
	g.mu.Lock()
	g.folderCache = make(map[string]string)
	g.mu.Unlock()

	log.Println("Cleaning up remote sync data from Google Drive...")
	// Delete remote folders
	roots := []string{"aerosync", "aerosync_versions"}
	for _, root := range roots {
		query := fmt.Sprintf("name='%s' and mimeType='application/vnd.google-apps.folder' and 'root' in parents and trashed=false", root)
		r, err := g.srv.Files.List().Q(query).Fields("files(id)").Do()
		if err == nil {
			for _, f := range r.Files {
				log.Printf("Deleting remote folder: %s (ID: %s)", root, f.Id)
				if err := g.srv.Files.Delete(f.Id).Do(); err != nil {
					log.Printf("Warning: Failed to delete remote folder %s: %v", root, err)
				}
			}
		}
	}

	return nil
}

func (g *GDriveProvider) ListRemote(ctx context.Context, label string) ([]RemoteFile, error) {
	// Find the top-level folder for the label
	folderPath := "aerosync/" + label
	folderId, err := g.getOrCreateFolder(ctx, folderPath)
	if err != nil {
		return nil, err
	}

	var results []RemoteFile
	err = g.listRecursive(ctx, folderId, "", &results)
	return results, err
}

func (g *GDriveProvider) listRecursive(ctx context.Context, folderId, relPrefix string, results *[]RemoteFile) error {
	query := fmt.Sprintf("'%s' in parents and trashed=false", folderId)
	r, err := g.srv.Files.List().Q(query).Fields("files(id, name, mimeType)").Do()
	if err != nil {
		return err
	}

	for _, f := range r.Files {
		path := f.Name
		if relPrefix != "" {
			path = relPrefix + "/" + f.Name
		}

		if f.MimeType == "application/vnd.google-apps.folder" {
			if err := g.listRecursive(ctx, f.Id, path, results); err != nil {
				return err
			}
		} else {
			*results = append(*results, RemoteFile{
				Path:    path,
				DriveID: f.Id,
			})
		}
	}
	return nil
}

func (g *GDriveProvider) RenameLabel(ctx context.Context, oldLabel, newLabel string) error {
	oldPath := "aerosync/" + oldLabel
	newPath := "aerosync/" + newLabel

	// 1. Find the aerosync root folder using the existing logic
	aerosyncRootId, err := g.getOrCreateFolder(ctx, "aerosync")
	if err != nil {
		return fmt.Errorf("failed to find aerosync root: %w", err)
	}

	// 2. Find the old label folder within aerosync
	safeOldLabel := strings.ReplaceAll(oldLabel, "'", "\\'")
	query := fmt.Sprintf("name='%s' and '%s' in parents and mimeType='application/vnd.google-apps.folder' and trashed=false", safeOldLabel, aerosyncRootId)
	r, err := g.srv.Files.List().Q(query).Fields("files(id)").Do()
	if err != nil || len(r.Files) == 0 {
		return fmt.Errorf("old label folder '%s' not found on Drive inside aerosync root", oldLabel)
	}
	oldFolderId := r.Files[0].Id

	// 3. Rename on Drive
	_, err = g.srv.Files.Update(oldFolderId, &drive.File{Name: newLabel}).Do()
	if err != nil {
		return fmt.Errorf("failed to rename folder on Drive from '%s' to '%s': %w", oldLabel, newLabel, err)
	}

	// 4. Update local cache and DB
	g.mu.Lock()
	defer g.mu.Unlock()

	// Use a temporary map to avoid modifying while iterating
	toAdd := make(map[string]string)
	toDelete := []string{}
	
	prefix := oldPath + "/"
	newPrefix := newPath + "/"

	for path, id := range g.folderCache {
		if path == oldPath {
			toDelete = append(toDelete, path)
			toAdd[newPath] = id
		} else if strings.HasPrefix(path, prefix) {
			newP := newPrefix + strings.TrimPrefix(path, prefix)
			toDelete = append(toDelete, path)
			toAdd[newP] = id
		}
	}

	for _, k := range toDelete {
		delete(g.folderCache, k)
	}
	for k, v := range toAdd {
		g.folderCache[k] = v
	}

	// Update DB
	_, err = g.db.Exec("UPDATE folders SET path = REPLACE(path, ?, ?) WHERE path LIKE ?", oldPath, newPath, oldPath+"%")
	return err
}

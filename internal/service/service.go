package service

import (
	"aerosync-service/internal/config"
	"aerosync-service/internal/sync"
	"context"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Service struct {
	config  *config.Config
	syncMgr *sync.SyncManager
	ctx     context.Context
	cancel  context.CancelFunc
	pidFile string
}

func NewService(cfg *config.Config, provider sync.CloudProvider) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	pidFile := filepath.Join(config.GetConfigDir(), "aerosync.pid")
	return &Service{
		config:  cfg,
		syncMgr: sync.NewSyncManager(provider),
		ctx:     ctx,
		cancel:  cancel,
		pidFile: pidFile,
	}
}

func (s *Service) Start() {
	// Check if already running
	if s.IsRunning() {
		log.Fatal("Service is already running")
	}

	// Write PID file
	if err := s.writePidFile(); err != nil {
		log.Fatalf("Failed to write PID file: %v", err)
	}

	// Run initial sync
	log.Println("Running initial sync...")
	if err := s.syncMgr.SyncPaths(s.ctx, s.config.SyncPaths, s.config.Versioning, s.config.MaxVersions, s.config.Ignore); err != nil {
		log.Printf("Initial sync error: %v", err)
	} else {
		log.Println("Initial sync completed")
	}

	go s.run()
	log.Println("Aerosync service started")
}

func (s *Service) Stop() {
	s.cancel()
	os.Remove(s.pidFile)
	log.Println("Aerosync service stopped")
}

func (s *Service) Restore(path string, targetOverride string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	for _, syncPath := range s.config.SyncPaths {
		absSyncPath, _ := filepath.Abs(syncPath.Path)
		if strings.HasPrefix(absPath, absSyncPath) {
			relPath, err := filepath.Rel(absSyncPath, absPath)
			if err != nil {
				continue
			}

			destPath := absPath
			if targetOverride != "" {
				destPath = filepath.Join(targetOverride, relPath)
			}

			baseName := filepath.Base(absSyncPath)
			return s.syncMgr.Restore(s.ctx, destPath, relPath, syncPath.Label, baseName)
		}
	}

	// If not found in sync paths, assume it's the file name relative to the first sync path
	syncPath := s.config.SyncPaths[0]
	absSyncPath, _ := filepath.Abs(syncPath.Path)
	
	destPath := filepath.Join(absSyncPath, path)
	if targetOverride != "" {
		destPath = filepath.Join(targetOverride, path)
	}

	baseName := filepath.Base(absSyncPath)
	return s.syncMgr.Restore(s.ctx, destPath, path, syncPath.Label, baseName)
}

func (s *Service) IsRunning() bool {
	data, err := os.ReadFile(s.pidFile)
	if err != nil {
		return false
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Send signal 0 to check if process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

func (s *Service) writePidFile() error {
	pid := os.Getpid()
	return os.WriteFile(s.pidFile, []byte(strconv.Itoa(pid)), 0644)
}

func (s *Service) run() {
	interval := 1 * time.Hour // default
	if s.config.SyncInterval != "" {
		if d, err := time.ParseDuration(s.config.SyncInterval); err == nil {
			interval = d
		} else {
			log.Printf("Invalid sync_interval '%s', using default 1h: %v", s.config.SyncInterval, err)
		}
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	log.Printf("Sync interval set to %v", interval)

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			if err := s.syncMgr.SyncPaths(s.ctx, s.config.SyncPaths, s.config.Versioning, s.config.MaxVersions, s.config.Ignore); err != nil {
				log.Printf("Sync error: %v", err)
			}
		}
	}
}

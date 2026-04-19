package service

import (
	"aerosync-service/internal/config"
	"aerosync-service/internal/sync"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
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

func (s *Service) SetupLogging() error {
	logsDir := config.GetLogsDir()
	logFile := filepath.Join(logsDir, "service.log")

	// 1. Rotation Check (50MB Limit)
	if info, err := os.Stat(logFile); err == nil {
		if info.Size() > 50*1024*1024 {
			rotatedName := filepath.Join(logsDir, fmt.Sprintf("service_%s.log", time.Now().Format("20060102_150405")))
			os.Rename(logFile, rotatedName)
		}
	}

	// 2. Automated Cleanup (Files > 7 Days)
	files, _ := os.ReadDir(logsDir)
	cutoff := time.Now().Add(-7 * 24 * time.Hour)
	for _, f := range files {
		path := filepath.Join(logsDir, f.Name())
		if info, err := f.Info(); err == nil {
			if info.ModTime().Before(cutoff) && !f.IsDir() {
				os.Remove(path)
			}
		}
	}

	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	log.SetOutput(f)
	return nil
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

func (s *Service) SyncAll() error {
	return s.syncMgr.SyncPaths(s.ctx, s.config.SyncPaths, s.config.Versioning, s.config.MaxVersions, s.config.Ignore)
}

func (s *Service) SyncLabel(label string) error {
	for _, p := range s.config.SyncPaths {
		if p.Label == label {
			return s.syncMgr.Provider.Sync(s.ctx, p.Path, p.Label, s.config.Versioning, s.config.MaxVersions, s.config.Ignore)
		}
	}
	return nil
}

func (s *Service) IsRunning() bool {
	data, err := os.ReadFile(s.pidFile)
	if err != nil {
		return false
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return false
	}

	// First try: standard Go process check
	process, err := os.FindProcess(pid)
	if err == nil {
		if err := process.Signal(syscall.Signal(0)); err == nil {
			return true
		}
	}

	// Second try: Windows specific fallback
	// This handles cases where Signal(0) might return 'Access Denied' 
	// but the process is clearly visible to the system.
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/NH")
	out, _ := cmd.Output()
	return strings.Contains(strings.ToLower(string(out)), "aerosync.exe")
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

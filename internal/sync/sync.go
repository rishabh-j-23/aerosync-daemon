package sync

import (
	"aerosync-service/internal/config"
	"context"
)

type CloudProvider interface {
	Sync(ctx context.Context, path string, label string, versioning bool, maxVersions int, ignore []string) error
	Restore(ctx context.Context, targetPath string, relPath string, label string, baseName string) error
	RestoreSpecific(ctx context.Context, driveID string, targetPath string) error
	Cleanup(ctx context.Context) error
	ListRemote(ctx context.Context, label string) ([]RemoteFile, error)
	GetFileVersions(ctx context.Context, label string, relPath string) ([]FileVersion, error)
	RenameLabel(ctx context.Context, oldLabel, newLabel string) error
	GetStatus(key string) (string, error)
	SetStatus(key, value string) error
}

type RemoteFile struct {
	Path    string
	DriveID string
}

type FileVersion struct {
	Number    int
	DriveID   string
	Timestamp int64
}

type SyncManager struct {
	Provider CloudProvider
}

func NewSyncManager(provider CloudProvider) *SyncManager {
	return &SyncManager{Provider: provider}
}

func (sm *SyncManager) SyncPaths(ctx context.Context, paths []config.SyncPath, versioning bool, maxVersions int, ignore []string) error {
	for _, p := range paths {
		if err := sm.Provider.Sync(ctx, p.Path, p.Label, versioning, maxVersions, ignore); err != nil {
			return err
		}
	}
	return nil
}

func (sm *SyncManager) SyncSpecific(ctx context.Context, p config.SyncPath, versioning bool, maxVersions int, ignore []string) error {
	return sm.Provider.Sync(ctx, p.Path, p.Label, versioning, maxVersions, ignore)
}

func (sm *SyncManager) Restore(ctx context.Context, targetPath string, relPath string, label string, baseName string) error {
	return sm.Provider.Restore(ctx, targetPath, relPath, label, baseName)
}

func (sm *SyncManager) Cleanup(ctx context.Context) error {
	return sm.Provider.Cleanup(ctx)
}

func (sm *SyncManager) ListRemote(ctx context.Context, label string) ([]RemoteFile, error) {
	return sm.Provider.ListRemote(ctx, label)
}

func (sm *SyncManager) RenameLabel(ctx context.Context, oldLabel, newLabel string) error {
	return sm.Provider.RenameLabel(ctx, oldLabel, newLabel)
}

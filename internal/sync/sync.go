package sync

import (
	"aerosync-service/internal/config"
	"context"
)

type CloudProvider interface {
	Sync(ctx context.Context, path string, label string, versioning bool, maxVersions int, ignore []string) error
	Restore(ctx context.Context, targetPath string, relPath string, label string, baseName string) error
	Cleanup(ctx context.Context) error
	ListRemote(ctx context.Context, label string) ([]RemoteFile, error)
}

type RemoteFile struct {
	Path    string
	DriveID string
}

type SyncManager struct {
	provider CloudProvider
}

func NewSyncManager(provider CloudProvider) *SyncManager {
	return &SyncManager{provider: provider}
}

func (sm *SyncManager) SyncPaths(ctx context.Context, paths []config.SyncPath, versioning bool, maxVersions int, ignore []string) error {
	for _, p := range paths {
		if err := sm.provider.Sync(ctx, p.Path, p.Label, versioning, maxVersions, ignore); err != nil {
			return err
		}
	}
	return nil
}

func (sm *SyncManager) Restore(ctx context.Context, targetPath string, relPath string, label string, baseName string) error {
	return sm.provider.Restore(ctx, targetPath, relPath, label, baseName)
}

func (sm *SyncManager) Cleanup(ctx context.Context) error {
	return sm.provider.Cleanup(ctx)
}

func (sm *SyncManager) ListRemote(ctx context.Context, label string) ([]RemoteFile, error) {
	return sm.provider.ListRemote(ctx, label)
}

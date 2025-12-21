package sync

import (
	"context"
)

type CloudProvider interface {
	Sync(ctx context.Context, path string, versioning bool, maxVersions int, ignore []string) error
	Restore(ctx context.Context, path string, baseName string) error
}

type SyncManager struct {
	provider CloudProvider
}

func NewSyncManager(provider CloudProvider) *SyncManager {
	return &SyncManager{provider: provider}
}

func (sm *SyncManager) SyncPaths(ctx context.Context, paths []string, versioning bool, maxVersions int, ignore []string) error {
	for _, path := range paths {
		if err := sm.provider.Sync(ctx, path, versioning, maxVersions, ignore); err != nil {
			return err
		}
	}
	return nil
}

func (sm *SyncManager) Restore(ctx context.Context, path string, baseName string) error {
	return sm.provider.Restore(ctx, path, baseName)
}

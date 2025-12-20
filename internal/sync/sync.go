package sync

import (
	"context"
)

type CloudProvider interface {
	Sync(ctx context.Context, path string) error
}

type SyncManager struct {
	provider CloudProvider
}

func NewSyncManager(provider CloudProvider) *SyncManager {
	return &SyncManager{provider: provider}
}

func (sm *SyncManager) SyncPaths(ctx context.Context, paths []string) error {
	for _, path := range paths {
		if err := sm.provider.Sync(ctx, path); err != nil {
			return err
		}
	}
	return nil
}

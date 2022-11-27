// Package syncmanager provides a manager for syncing btrfs snapshots
// with a local or remote host.
package syncmanager

import (
	"context"
	"fmt"

	"github.com/tinyzimmer/btrsync/pkg/cmd/config"
	"github.com/tinyzimmer/btrsync/pkg/cmd/snaputil"
)

var OffsetDirectory = ".btrsync"

type Manager interface {
	Sync(ctx context.Context) error
	Prune(ctx context.Context) error
	Close() error
}

func New(cfg *Config) (Manager, error) {
	subvolInfo, err := snaputil.ResolveSubvolumeDetails(
		cfg.Logger,
		cfg.Verbosity,
		cfg.FullSubvolumePath,
		cfg.SnapshotDirectory,
		cfg.SnapshotName,
	)
	if err != nil {
		return nil, fmt.Errorf("error resolving subvolume details: %w", err)
	}
	mirrorURL, err := cfg.MirrorURL()
	if err != nil {
		return nil, err
	}
	var manager Manager
	switch mirrorURL.Scheme {
	case "file":
		if cfg.MirrorFormat.IsCompressed() {
			manager, err = NewLocalCompressedManager(cfg, subvolInfo)
		} else {
			switch cfg.MirrorFormat {
			case config.MirrorFormatSubvolume, "":
				manager, err = NewLocalSubvolumeManager(cfg, subvolInfo)
			case config.MirrorFormatDirectory:
				manager, err = NewLocalDirectoryManager(cfg, subvolInfo)
			default:
				return nil, fmt.Errorf("unsupported local mirror format: %s", cfg.MirrorFormat)
			}
		}
	case "ssh":
		if cfg.MirrorFormat.IsCompressed() {
			manager, err = NewSSHCompressedManager(cfg, subvolInfo)
		} else {
			switch cfg.MirrorFormat {
			case config.MirrorFormatSubvolume, "":
				manager, err = NewSSHSubvolumeManager(cfg, subvolInfo)
			case config.MirrorFormatDirectory:
				manager, err = NewSSHDirectoryManager(cfg, subvolInfo)
			default:
				return nil, fmt.Errorf("unsupported ssh mirror format: %s", cfg.MirrorFormat)
			}
		}
	default:
		err = fmt.Errorf("unsupported mirror scheme: %s", mirrorURL.Scheme)
	}
	if err != nil {
		return nil, err
	}
	return manager, nil
}

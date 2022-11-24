/*
This file is part of btrsync.

Btrsync is free software: you can redistribute it and/or modify it under the terms of the
GNU Lesser General Public License as published by the Free Software Foundation, either
version 3 of the License, or (at your option) any later version.

Btrsync is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY;
without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
See the GNU Lesser General Public License for more details.

You should have received a copy of the GNU Lesser General Public License along with btrsync.
If not, see <https://www.gnu.org/licenses/>.
*/

// Package snapmanager provides a simple snapshot manager for btrfs subvolumes.
package snapmanager

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/tinyzimmer/btrsync/cmd/btrsync/cmd/snaputil"
	"github.com/tinyzimmer/btrsync/pkg/btrfs"
)

// Config is the config for a snapshot manager.
type Config struct {
	FullSubvolumePath         string
	SnapshotDirectory         string
	SnapshotName              string
	SnapshotInterval          time.Duration
	SnapshotMinimumRetention  time.Duration
	SnapshotRetention         time.Duration
	SnapshotRetentionInterval time.Duration
	TimeFormat                string
	Logger                    *log.Logger
	Verbosity                 int
}

// SnapManager is a manager for snapshots of a given subvolume.
type SnapManager struct {
	config   *Config
	rootInfo *btrfs.RootInfo
}

// New prepares a new snapshot manager for the given subvolume path and config.
func New(cfg *Config) (*SnapManager, error) {
	info, err := snaputil.ResolveSubvolumeDetails(
		cfg.Logger,
		cfg.Verbosity,
		cfg.FullSubvolumePath,
		cfg.SnapshotDirectory,
		cfg.SnapshotName,
	)
	if err != nil {
		return nil, err
	}
	return &SnapManager{cfg, info}, nil
}

// EnsureMostRecentSnapshot ensures that a snapshot exists for the subvolume within
// the configured snapshot interval. If a snapshot does not exist, it will be created
// with the name and timestamp format provided in the configuration.
func (sm *SnapManager) EnsureMostRecentSnapshot() error {
	if err := sm.ensureSnapshotSubvol(); err != nil {
		return err
	}
	mostRecent, err := sm.GetMostRecentSnapshot()
	if err != nil {
		return err
	}
	if mostRecent != nil {
		if sm.config.Verbosity >= 2 {
			sm.config.Logger.Printf("Most recent snapshot found at %q: %s\n", mostRecent.FullPath, mostRecent.CreationTime)
		}
		if time.Since(mostRecent.CreationTime) < sm.config.SnapshotInterval {
			if sm.config.Verbosity >= 2 {
				sm.config.Logger.Printf("Most recent snapshot is within interval, skipping\n")
			}
			return nil
		}
	}
	snapshotPath := filepath.Join(
		sm.config.SnapshotDirectory,
		fmt.Sprintf("%s.%s", sm.config.SnapshotName, time.Now().Format(sm.config.TimeFormat)),
	)
	sm.config.Logger.Printf("Creating read-only snapshot %q from %q\n", snapshotPath, sm.config.FullSubvolumePath)
	if err := btrfs.CreateSnapshot(
		sm.config.FullSubvolumePath,
		btrfs.WithSnapshotPath(snapshotPath),
		btrfs.WithReadOnlySnapshot(),
	); err != nil {
		return err
	}
	if sm.config.Verbosity >= 2 {
		sm.config.Logger.Printf("Snapshot created successfully, syncing filesystem to disk\n")
	}
	return btrfs.SyncFilesystem(snapshotPath)
}

// GetMostRecentSnapshot returns the most recent snapshot of the subvolume.
func (sm *SnapManager) GetMostRecentSnapshot() (*btrfs.RootInfo, error) {
	var latest *btrfs.RootInfo
	for _, snap := range sm.rootInfo.Snapshots {
		if latest == nil || snap.CreationTime.After(latest.CreationTime) {
			latest = snap
		}
	}
	return latest, nil
}

// PruneSnapshots prunes snapshots that are older than the configured retention period and that are
// within the minimum retention period according to the configured intervals.
func (sm *SnapManager) PruneSnapshots() error {
	if sm.config.SnapshotRetention == 0 {
		return nil
	}
	// Delete snapshots older than the retention period
	if sm.config.Verbosity >= 1 {
		sm.config.Logger.Printf("Pruning snapshots older than %s\n", sm.config.SnapshotRetention)
	}

	remaining := make([]*btrfs.RootInfo, 0)
	for _, snap := range sm.rootInfo.Snapshots {
		fullPath := filepath.Join(sm.config.SnapshotDirectory, snap.Name)
		if sm.config.Verbosity >= 3 {
			sm.config.Logger.Printf("Considering snapshot %q created at %s for max retention deletion\n", fullPath, snap.CreationTime)
		}
		if time.Since(snap.CreationTime) > sm.config.SnapshotRetention {
			sm.config.Logger.Printf("Deleting snapshot %q\n", fullPath)
			if err := btrfs.DeleteSubvolume(fullPath); err != nil {
				return err
			}
		} else {
			remaining = append(remaining, snap)
		}
	}
	sm.rootInfo.Snapshots = remaining

	// Prune snapshots within the retention period according to the retention interval
	if sm.config.SnapshotRetentionInterval == 0 {
		return nil
	}
	if sm.config.Verbosity >= 1 {
		sm.config.Logger.Printf("Pruning snapshots within retention period %s according to interval %s\n", sm.config.SnapshotRetention, sm.config.SnapshotRetentionInterval)
	}
	snapshots := make([]*btrfs.RootInfo, 0)
	for _, snap := range sm.rootInfo.Snapshots {
		if sm.config.Verbosity >= 3 {
			sm.config.Logger.Printf("Considering snapshot %q created at %s for minimum retention deletion\n", snap.Path, snap.CreationTime)
		}
		if time.Since(snap.CreationTime) < sm.config.SnapshotRetention {
			if time.Since(snap.CreationTime) > sm.config.SnapshotMinimumRetention {
				if sm.config.Verbosity >= 3 {
					sm.config.Logger.Printf("Snapshot %q is within the maximum and minimum retention periods\n", snap.Path)
				}
				snapshots = append(snapshots, snap)
			}
		}
	}
	if len(snapshots) == 0 {
		if sm.config.Verbosity >= 1 {
			sm.config.Logger.Printf("No snapshots to prune\n")
		}
	}
	snaputil.SortSnapshots(snapshots, snaputil.SortAscending)
	for i, snap := range snapshots {
		fullPath := filepath.Join(sm.config.SnapshotDirectory, snap.Name)
		if sm.config.Verbosity >= 3 {
			sm.config.Logger.Printf("Considering snapshot %q created at %s for interval retention deletion\n", fullPath, snap.CreationTime)
		}
		if i%int(sm.config.SnapshotRetention/sm.config.SnapshotRetentionInterval) == 0 {
			sm.config.Logger.Printf("Deleting snapshot %q\n", fullPath)
			if err := btrfs.DeleteSubvolume(fullPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func (sm *SnapManager) ensureSnapshotSubvol() error {
	snapDir := sm.config.SnapshotDirectory
	isSubvol, err := btrfs.IsSubvolume(snapDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		sm.config.Logger.Printf("Creating snapshot subvolume %s\n", snapDir)
		if err := btrfs.CreateSubvolume(snapDir); err != nil {
			return err
		}
		return nil
	}
	if !isSubvol {
		return fmt.Errorf("%s is not a btrfs subvolume", snapDir)
	}
	if sm.config.Verbosity >= 2 {
		sm.config.Logger.Printf("Snapshot subvolume %s already exists\n", snapDir)
	}
	return nil
}

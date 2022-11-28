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

package syncmanager

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
	"github.com/tinyzimmer/btrsync/pkg/btrfs"
	"github.com/tinyzimmer/btrsync/pkg/cmd/snaputil"
	"github.com/tinyzimmer/btrsync/pkg/receive"
	"github.com/tinyzimmer/btrsync/pkg/receive/receivers/directory"
)

type localDirectoryManager struct {
	config     *Config
	sourceInfo *btrfs.RootInfo
	mirrorPath string
}

func NewLocalDirectoryManager(cfg *Config, subvolInfo *btrfs.RootInfo) (Manager, error) {
	mirrorUrl, err := cfg.MirrorURL()
	if err != nil {
		return nil, err
	}
	mirrorPath := mirrorUrl.Path
	cfg.LogVerbose(0, "Initiating local directory sync manager for %q with mirror URL: %s\n", cfg.FullSubvolumePath, mirrorPath)
	return &localDirectoryManager{
		config:     cfg,
		sourceInfo: subvolInfo,
		mirrorPath: mirrorPath,
	}, nil
}

func (sm *localDirectoryManager) Sync(ctx context.Context) error {
	path := filepath.Join(sm.mirrorPath, sm.config.SubvolumeIdentifier)
	sm.config.LogVerbose(0, "Syncing directory mirror: %s\n", path)
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create mirror directory: %s", err)
	}
	snapshots := snaputil.MapParents(sm.sourceInfo.Snapshots)
	for _, snap := range snapshots {
		if err := sm.syncSnapshot(ctx, path, snap.Parent, snap.Snapshot); err != nil {
			return err
		}
	}
	return nil
}

func (sm *localDirectoryManager) syncSnapshot(ctx context.Context, destination string, parent, snap *btrfs.RootInfo) error {
	// Check if the snapshot is already synced by verifying it's UUID file
	uuidFile := filepath.Join(destination, OffsetDirectory, snap.UUID.String())
	sm.config.LogVerbose(1, "Checking for snapshot progress file at %q\n", uuidFile)
	f, err := os.Open(uuidFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to open uuid file: %s", err)
		}
		sm.config.LogVerbose(2, "Progress file %q does not exist\n", uuidFile)
	} else {
		var offset uint64
		if _, err := fmt.Fscanf(f, "%d", &offset); err != nil {
			return err
		}
		f.Close()
		sm.config.LogVerbose(2, "Progress file %q found with offset %d\n", uuidFile, offset)
		if offset == math.MaxUint64-1 {
			sm.config.LogVerbose(1, "Snapshot %s is already synced, skipping", snap.Name)
			return nil
		}
	}

	sm.config.LogVerbose(0, "Syncing snapshot %q contents to %q\n", snap.Path, destination)
	var wg sync.WaitGroup

	pipeOpt, pipe, err := btrfs.SendToPipe()
	if err != nil {
		return fmt.Errorf("error creating send pipe: %w", err)
	}
	defer pipe.Close()
	errors := make(chan error, 1)

	wg.Add(1)
	go func() {
		defer wg.Done()
		sendOpts := []btrfs.SendOption{
			pipeOpt,
			btrfs.SendWithLogger(sm.config.Logger, sm.config.Verbosity),
			btrfs.SendCompressedData(),
		}
		if parent != nil {
			parentPath := filepath.Join(sm.config.SnapshotDirectory, parent.Name)
			sendOpts = append(sendOpts, btrfs.SendWithParentRoot(parentPath))
		}
		snapshotPath := filepath.Join(sm.config.SnapshotDirectory, snap.Name)
		if err := btrfs.Send(snapshotPath, sendOpts...); err != nil {
			err = fmt.Errorf("error sending snapshot: %w", err)
			errors <- err
			return
		}
		errors <- nil
	}()

	receiveOpts := []receive.Option{
		receive.WithLogger(sm.config.Logger, sm.config.Verbosity),
		receive.WithContext(ctx),
		receive.HonorEndCommand(),
		receive.To(directory.New(destination, OffsetDirectory)),
	}
	err = receive.ProcessSendStream(pipe, receiveOpts...)
	if err != nil {
		return err
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			return err
		}
	}
	return nil
}

func (sm *localDirectoryManager) Prune(ctx context.Context) error {
	sm.config.LogVerbose(0, "Pruning expired offset files")
	path := filepath.Join(sm.mirrorPath, sm.config.SubvolumeIdentifier)
	files, err := os.ReadDir(filepath.Join(path, OffsetDirectory))
	if err != nil {
		return fmt.Errorf("failed to read offset directory: %s", err)
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		uuid, err := uuid.Parse(file.Name())
		if err != nil {
			sm.config.LogVerbose(1, "Failed to parse uuid from file %q: %s", file.Name(), err)
			continue
		}
		if !snaputil.SnapshotUUIDExists(sm.sourceInfo.Snapshots, uuid) {
			sm.config.LogVerbose(1, "Removing expired offset file %q", file.Name())
			if err := os.Remove(filepath.Join(path, OffsetDirectory, file.Name())); err != nil {
				return fmt.Errorf("failed to remove expired offset file: %s", err)
			}
		} else {
			sm.config.LogVerbose(2, "Keeping offset file %q", file.Name())
		}
	}
	return nil
}

func (sm *localDirectoryManager) Close() error {
	return nil
}

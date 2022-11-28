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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/tinyzimmer/btrsync/pkg/btrfs"
	"github.com/tinyzimmer/btrsync/pkg/cmd/snaputil"
	"github.com/tinyzimmer/btrsync/pkg/receive"
	"github.com/tinyzimmer/btrsync/pkg/receive/receivers/local"
)

type localSubvolumeManager struct {
	config     *Config
	sourceInfo *btrfs.RootInfo
	mirrorPath string
}

func NewLocalSubvolumeManager(cfg *Config, subvolInfo *btrfs.RootInfo) (Manager, error) {
	mirrorUrl, err := cfg.MirrorURL()
	if err != nil {
		return nil, err
	}
	mirrorPath := mirrorUrl.Path
	cfg.LogVerbose(0, "Initiating local subvolume sync manager for %q with mirror URL: %s\n", cfg.FullSubvolumePath, mirrorPath)
	return &localSubvolumeManager{
		config:     cfg,
		sourceInfo: subvolInfo,
		mirrorPath: mirrorPath,
	}, nil
}

func (sm *localSubvolumeManager) Sync(ctx context.Context) error {
	sm.config.LogVerbose(1, "Ensuring mirror path is ready and accessible")
	if err := sm.ensureLocalMirrorPath(ctx); err != nil {
		return err
	}
	snapshots := snaputil.MapParents(sm.sourceInfo.Snapshots)
	for _, snap := range snapshots {
		if err := sm.syncSnapshot(ctx, snap.Parent, snap.Snapshot); err != nil {
			return err
		}
	}
	return nil
}

func (sm *localSubvolumeManager) Prune(ctx context.Context) error {
	sm.config.LogVerbose(0, "Pruning expired snapshots from mirror: %s\n", sm.config.MirrorPath)
	return sm.pruneLocalMirror(ctx)
}

func (sm *localSubvolumeManager) syncSnapshot(ctx context.Context, parent, snap *btrfs.RootInfo) error {
	var wg sync.WaitGroup

	destination := filepath.Join(sm.mirrorPath, sm.config.SubvolumeIdentifier)
	snapshotPath := filepath.Join(sm.config.SnapshotDirectory, snap.Name)
	destinationPath := filepath.Join(destination, snap.Path)

	receiveOpts := []receive.Option{
		receive.WithLogger(sm.config.Logger, sm.config.Verbosity),
		receive.WithContext(ctx),
		receive.HonorEndCommand(),
		receive.To(local.New(destination)),
	}

	// Check if the destination exists
	found, synced, err := sm.checkDestinationSnapshotLocal(ctx, snap)
	if err != nil {
		return err
	}
	if synced {
		sm.config.LogVerbose(1, "Snapshot %q already synced to %q\n", snap.Path, destination)
		return nil
	} else if found {
		sm.config.LogVerbose(0, "Snapshot %q already exists at %q, but is not synced. Will try incremental send.\n", snap.Path, destination)
		sm.config.LogVerbose(0, "Searching for command offset to resume from")
		var parentPath string
		var destinationParentPath string
		if parent != nil {
			parentPath = filepath.Join(sm.config.SnapshotDirectory, parent.Name)
			destinationParentPath = filepath.Join(destination, parent.Path)
		}
		offset, err := receive.FindPathDiffOffset(snapshotPath, destinationPath, parentPath, destinationParentPath)
		if err != nil {
			return fmt.Errorf("error finding path diff offset: %w", err)
		}
		sm.config.LogVerbose(0, "Found stream diff offset at %d", offset)
		receiveOpts = append(receiveOpts, receive.FromOffset(offset))
	}

	sm.config.LogVerbose(0, "Syncing snapshot %q to %q\n", snap.Path, destination)

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
		if err := btrfs.Send(snapshotPath, sendOpts...); err != nil {
			err = fmt.Errorf("error sending snapshot: %w", err)
			errors <- err
		}
	}()

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

func (sm *localSubvolumeManager) checkDestinationSnapshotLocal(ctx context.Context, snap *btrfs.RootInfo) (found, synced bool, err error) {
	destination := filepath.Join(sm.mirrorPath, sm.config.SubvolumeIdentifier, snap.Path)
	if _, err := os.Stat(destination); err != nil {
		if os.IsNotExist(err) {
			return false, false, nil
		}
		return false, false, err
	}
	subvol, err := btrfs.SubvolumeSearch(btrfs.SearchWithPath(destination))
	if err != nil {
		if errors.Is(err, btrfs.ErrNotFound) {
			return false, false, nil
		}
		return false, false, fmt.Errorf("error searching for subvolume: %w", err)
	}
	return true, subvol.ReceivedUUID == snap.UUID, nil
}

func (sm *localSubvolumeManager) ensureLocalMirrorPath(ctx context.Context) error {
	path := sm.mirrorPath
	// Make sure the base mirror path exists
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Check if the destination is on a btrfs filesystem, we'll use a subvolume
			// if so, otherwise we'll use a directory.
			sm.config.LogVerbose(1, "Mirror path %q does not exist, creating\n", path)
			ok, err := btrfs.IsBtrfs(path)
			if err != nil {
				return fmt.Errorf("error checking if destination is btrfs: %w", err)
			}
			if !ok {
				return fmt.Errorf("local destination %s is not a btrfs filesystem (may be supported in the future)", path)
			}
			// Make a subvolume
			sm.config.LogVerbose(0, "Creating btrfs subvolume at %q\n", path)
			if err := btrfs.CreateSubvolume(path); err != nil {
				return fmt.Errorf("error creating subvolume at %q: %w", path, err)
			}
			if err := btrfs.SyncFilesystem(path); err != nil {
				return fmt.Errorf("error syncing filesystem: %w", err)
			}
		} else {
			return fmt.Errorf("error accessing mirror path: %w", err)
		}
	}
	// Make sure a subvolume for this subvolume exists
	path = filepath.Join(path, sm.config.SubvolumeIdentifier)
	_, err = os.Stat(path)
	if err == nil {
		return nil
	}
	if os.IsNotExist(err) {
		// Check if the destination is on a btrfs filesystem, we'll use a subvolume
		// if so, otherwise we'll use a directory.
		sm.config.LogVerbose(1, "Mirror subvolume path %q does not exist, creating\n", path)
		ok, err := btrfs.IsBtrfs(path)
		if err != nil {
			return fmt.Errorf("error checking if destination is btrfs: %w", err)
		}
		if !ok {
			return fmt.Errorf("local destination %s is not a btrfs filesystem (may be supported in the future)", path)
		}
		// Make a subvolume
		sm.config.LogVerbose(0, "Creating btrfs subvolume at %q\n", path)
		if err := btrfs.CreateSubvolume(path); err != nil {
			return fmt.Errorf("error creating subvolume at %q: %w", path, err)
		}
		return btrfs.SyncFilesystem(path)
	}
	return fmt.Errorf("error accessing mirror subvolume path: %w", err)
}

func (sm *localSubvolumeManager) pruneLocalMirror(ctx context.Context) error {
	destination := filepath.Join(sm.mirrorPath, sm.config.SubvolumeIdentifier)
	sm.config.LogVerbose(2, "Listing snapshots in tree at %q\n", sm.mirrorPath)

	mirrorInfo, err := btrfs.SubvolumeSearch(btrfs.SearchWithPath(sm.mirrorPath))
	if err != nil {
		return fmt.Errorf("error looking up information on mirror path: %w", err)
	}

	var tree *btrfs.RBRoot
	var retries int
	for tree == nil && retries <= 3 {
		if retries > 0 {
			sm.config.LogVerbose(1, "Error while building tree at %q, retrying: %s\n", sm.mirrorPath, err)
		}
		tree, err = btrfs.BuildRBTree(sm.mirrorPath)
		if err != nil {
			time.Sleep(time.Millisecond * 100)
		}
	}
	if err != nil {
		return fmt.Errorf("error building tree at %q: %w", sm.mirrorPath, err)
	}
	tree = tree.FilterFromRoot(mirrorInfo.RootID)

	var expired []string
	tree.PreOrderIterate(func(info *btrfs.RootInfo, _ error) error {
		if info.Deleted || info.FullPath == "" || !strings.HasPrefix(info.FullPath, sm.config.SubvolumeIdentifier) {
			return nil
		}
		if info.Name == sm.config.SubvolumeIdentifier {
			return nil
		}
		fullpath := filepath.Join(destination, info.Name)
		sm.config.LogVerbose(3, "Checking if mirrored snapshot %q is expired\n", fullpath)
		if !snaputil.SnapshotSliceContains(sm.sourceInfo.Snapshots, info.Name) {
			sm.config.LogVerbose(1, "Marking snapshot %q for expiry\n", fullpath)
			expired = append(expired, fullpath)
		} else {
			sm.config.LogVerbose(3, "Mirrored snapshot %q has not expired\n", fullpath)
		}
		return nil
	})

	for _, path := range expired {
		sm.config.LogVerbose(0, "Expiring mirrored snapshot %q\n", path)
		if err := btrfs.DeleteSubvolume(path, true); err != nil {
			return fmt.Errorf("error deleting subvolume %q: %w", path, err)
		}
	}

	return nil
}

func (sm *localSubvolumeManager) Close() error {
	return nil
}

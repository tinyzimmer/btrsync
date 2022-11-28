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
	"bytes"
	"context"
	"fmt"
	"math"
	"net/url"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
	"github.com/tinyzimmer/btrsync/pkg/btrfs"
	"github.com/tinyzimmer/btrsync/pkg/cmd/snaputil"
	"github.com/tinyzimmer/btrsync/pkg/cmd/sshutil"
	"github.com/tinyzimmer/btrsync/pkg/receive"
	"github.com/tinyzimmer/btrsync/pkg/receive/receivers/sshdir"
	"golang.org/x/crypto/ssh"
)

type sshDirectoryManager struct {
	config     *Config
	sourceInfo *btrfs.RootInfo
	mirrorURL  *url.URL
	sshClient  *ssh.Client
}

func NewSSHDirectoryManager(cfg *Config, subvolInfo *btrfs.RootInfo) (Manager, error) {
	mirrorURL, err := cfg.MirrorURL()
	if err != nil {
		return nil, err
	}
	cfg.LogVerbose(0, "Initiating SSH directory sync manager for %q with mirror URL: %s\n",
		cfg.FullSubvolumePath, mirrorURL.String())

	sshcfg, err := cfg.SSHConfig()
	if err != nil {
		return nil, err
	}
	cfg.LogVerbose(1, "Connecting to remote host using tcp: %s\n", mirrorURL.String())
	sshClient, err := sshutil.Dial(context.Background(), mirrorURL, sshcfg)
	if err != nil {
		return nil, fmt.Errorf("failed to dial ssh server: %s", err)
	}
	return &sshDirectoryManager{
		config:     cfg,
		sourceInfo: subvolInfo,
		mirrorURL:  mirrorURL,
		sshClient:  sshClient,
	}, nil
}

func (sm *sshDirectoryManager) Sync(ctx context.Context) error {
	path := filepath.Join(sm.mirrorURL.Path, sm.config.SubvolumeIdentifier)
	sm.config.LogVerbose(0, "Syncing ssh directory mirror: %s\n", path)
	if err := sshutil.MkdirAll(ctx, sm.sshClient, path); err != nil {
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

func (sm *sshDirectoryManager) syncSnapshot(ctx context.Context, destination string, parent, snap *btrfs.RootInfo) error {
	// Check if the snapshot is already synced by verifying it's UUID file
	uuidFile := filepath.Join(destination, OffsetDirectory, snap.UUID.String())
	sm.config.LogVerbose(1, "Checking for snapshot progress file on remote at %q\n", uuidFile)
	data, err := sshutil.ReadFile(ctx, sm.sshClient, uuidFile)
	if err != nil {
		if !sshutil.IsFileNotExist(err) {
			return fmt.Errorf("failed to read snapshot progress file: %s", err)
		}
		sm.config.LogVerbose(2, "Progress file %q does not exist\n", uuidFile)
	} else {
		var offset uint64
		if _, err := fmt.Fscanf(bytes.NewReader(data), "%d", &offset); err != nil {
			return err
		}
		sm.config.LogVerbose(2, "Progress file %q found with offset %d\n", uuidFile, offset)
		if offset == math.MaxUint64-1 {
			sm.config.LogVerbose(1, "Snapshot %s is already synced, skipping", snap.Name)
			return nil
		}
	}

	sm.config.LogVerbose(0, "Syncing snapshot %q contents to remote %q\n", snap.Path, destination)
	var wg sync.WaitGroup

	pipeOpt, pipe, err := btrfs.SendToPipe()
	if err != nil {
		return fmt.Errorf("error creating send pipe: %w", err)
	}
	defer pipe.Close()

	wg.Add(1)
	errors := make(chan error, 1)
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
		}
	}()

	receiveOpts := []receive.Option{
		receive.WithLogger(sm.config.Logger, sm.config.Verbosity),
		receive.WithContext(ctx),
		receive.HonorEndCommand(),
		receive.To(sshdir.New(sm.sshClient, destination, OffsetDirectory)),
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

func (sm *sshDirectoryManager) Prune(ctx context.Context) error {
	sm.config.LogVerbose(0, "Pruning expired offset files on the remote host")
	path := filepath.Join(sm.mirrorURL.Path, sm.config.SubvolumeIdentifier)
	files, err := sshutil.ReadDir(ctx, sm.sshClient, filepath.Join(path, OffsetDirectory))
	if err != nil {
		return fmt.Errorf("failed to list offset files: %s", err)
	}
	for _, file := range files {
		uuid, err := uuid.Parse(file)
		if err != nil {
			sm.config.LogVerbose(1, "Failed to parse uuid from file %q: %s", file, err)
			continue
		}
		if !snaputil.SnapshotUUIDExists(sm.sourceInfo.Snapshots, uuid) {
			sm.config.LogVerbose(1, "Removing expired offset file %q", file)
			if err := sshutil.RemoveFile(ctx, sm.sshClient, filepath.Join(path, OffsetDirectory, file)); err != nil {
				return fmt.Errorf("failed to remove offset file %q: %s", file, err)
			}
		} else {
			sm.config.LogVerbose(2, "Keeping offset file %q", file)
		}
	}
	return nil
}

func (sm *sshDirectoryManager) Close() error {
	return sm.sshClient.Close()
}

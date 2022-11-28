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
	"compress/gzip"
	"compress/lzw"
	"context"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"
	"github.com/tinyzimmer/btrsync/pkg/btrfs"
	"github.com/tinyzimmer/btrsync/pkg/cmd/config"
	"github.com/tinyzimmer/btrsync/pkg/cmd/snaputil"
	"github.com/tinyzimmer/btrsync/pkg/cmd/sshutil"
	"golang.org/x/crypto/ssh"
)

type sshCompressedManager struct {
	config     *Config
	sourceInfo *btrfs.RootInfo
	mirrorURL  *url.URL
	sshClient  *ssh.Client
}

func NewSSHCompressedManager(cfg *Config, subvolInfo *btrfs.RootInfo) (Manager, error) {
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
	var addr string
	if mirrorURL.Port() != "" {
		addr = fmt.Sprintf("%s:%s", mirrorURL.Hostname(), mirrorURL.Port())
	} else {
		addr = fmt.Sprintf("%s:22", mirrorURL.Hostname())
	}
	cfg.LogVerbose(1, "Connecting to remote host using tcp: %s\n", addr)
	sshClient, err := ssh.Dial("tcp", addr, sshcfg)
	if err != nil {
		return nil, fmt.Errorf("failed to dial ssh server: %s", err)
	}
	return &sshCompressedManager{
		config:     cfg,
		sourceInfo: subvolInfo,
		mirrorURL:  mirrorURL,
		sshClient:  sshClient,
	}, nil
}

func (sm *sshCompressedManager) Sync(ctx context.Context) error {
	path := filepath.Join(sm.mirrorURL.Path, sm.config.SubvolumeIdentifier)
	sm.config.LogVerbose(0, "Syncing %s compressed mirror: %q\n", sm.config.MirrorFormat, path)
	if err := sshutil.MkdirAll(ctx, sm.sshClient, path); err != nil {
		return fmt.Errorf("failed to create mirror directory: %s", err)
	}
	snaputil.SortSnapshots(sm.sourceInfo.Snapshots, snaputil.SortAscending)
	for _, snap := range sm.sourceInfo.Snapshots {
		if err := sm.syncSnapshot(ctx, path, nil, snap); err != nil {
			return err
		}
	}
	return nil
}

func (sm *sshCompressedManager) syncSnapshot(ctx context.Context, destination string, _, snap *btrfs.RootInfo) error {
	snapshotPath := filepath.Join(sm.config.SnapshotDirectory, snap.Name)
	uuidfile := filepath.Join(destination, OffsetDirectory, snap.UUID.String())
	destination = filepath.Join(destination, snap.Name+"."+string(sm.config.MirrorFormat))

	sm.config.LogVerbose(1, "Checking for snapshot completion file at %q\n", uuidfile)
	_, err := sshutil.ReadFile(ctx, sm.sshClient, uuidfile)
	if err == nil {
		sm.config.LogVerbose(1, "Snapshot %q already synced, skipping\n", snap.Name)
		return nil
	}
	if !sshutil.IsFileNotExist(err) {
		return fmt.Errorf("failed to check for snapshot completion file: %s", err)
	}

	sm.config.LogVerbose(0, "Syncing %s compressed snapshot %q to %q on remote %s\n",
		sm.config.MirrorFormat, snap.Path, destination, sm.mirrorURL.Hostname())

	r, w := io.Pipe()
	var enc io.WriteCloser
	switch sm.config.MirrorFormat {
	case config.MirrorFormatGzip:
		enc = gzip.NewWriter(w)
	case config.MirrorFormatLzw:
		enc = lzw.NewWriter(w, lzw.LSB, 8)
	case config.MirrorFormatZlib:
		enc = gzip.NewWriter(w)
	case config.MirrorFormatZstd:
		enc, err = zstd.NewWriter(w)
		if err != nil {
			return fmt.Errorf("failed to create zstd writer: %s", err)
		}
	}

	// Set up the pipe to the encoder
	var wg sync.WaitGroup
	pipeOpt, pipe, err := btrfs.SendToPipe()
	if err != nil {
		return fmt.Errorf("error creating send pipe: %w", err)
	}
	defer pipe.Close()
	errors := make(chan error, 3)

	// Start the sendstream to the pipe
	wg.Add(1)
	go func() {
		defer wg.Done()
		sendOpts := []btrfs.SendOption{
			pipeOpt,
			btrfs.SendWithLogger(sm.config.Logger, sm.config.Verbosity),
			btrfs.SendCompressedData(),
		}
		if err := btrfs.Send(snapshotPath, sendOpts...); err != nil {
			err = fmt.Errorf("error sending snapshot: %w", err)
			errors <- err
		}
	}()

	// Start the encoder on the read end of the pipe to the writer
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer w.Close()
		defer enc.Close()
		_, err := io.Copy(enc, pipe)
		if err != nil {
			errors <- fmt.Errorf("error copying to encoder: %w", err)
		}
	}()

	// Start the write to the remote destination
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := sshutil.WriteFile(ctx, sm.sshClient, destination, r); err != nil {
			errors <- fmt.Errorf("error writing to remote destination: %w", err)
		}
	}()

	wg.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			return err
		}
	}

	// Create the completion file
	sm.config.LogVerbose(1, "Creating snapshot completion file at %q\n", uuidfile)
	if err := sshutil.MkdirAll(ctx, sm.sshClient, filepath.Dir(uuidfile)); err != nil {
		return fmt.Errorf("failed to create completion file directory: %s", err)
	}
	if err := sshutil.WriteFile(ctx, sm.sshClient, uuidfile, bytes.NewReader(nil)); err != nil {
		return fmt.Errorf("failed to create completion file: %s", err)
	}
	return nil
}

func (sm *sshCompressedManager) Prune(ctx context.Context) error {
	destination := filepath.Join(sm.mirrorURL.Path, sm.config.SubvolumeIdentifier)
	uuiddir := filepath.Join(destination, OffsetDirectory)
	sm.config.LogVerbose(2, "Listing compressed snapshots on remote at %q\n", destination)

	files, err := sshutil.ReadDir(ctx, sm.sshClient, destination)
	if err != nil {
		return fmt.Errorf("failed to read destination directory: %w", err)
	}

	var expired []string
	for _, file := range files {
		snapshotName := strings.TrimSuffix(file, "."+string(sm.config.MirrorFormat))
		if !snaputil.SnapshotSliceContains(sm.sourceInfo.Snapshots, snapshotName) {
			sm.config.LogVerbose(1, "Marking snapshot %q for expiry\n", file)
			expired = append(expired, file)
		} else {
			sm.config.LogVerbose(3, "Mirrored snapshot %q has not expired\n", file)
		}
	}

	for _, path := range expired {
		path = filepath.Join(destination, path)
		sm.config.LogVerbose(0, "Expiring mirrored snapshot %q\n", path)
		if err := sshutil.RemoveFile(ctx, sm.sshClient, path); err != nil {
			return fmt.Errorf("error deleting snapshot file %q: %w", path, err)
		}
	}

	uuids, err := sshutil.ReadDir(ctx, sm.sshClient, uuiddir)
	if err != nil {
		return fmt.Errorf("failed to read uuid directory: %w", err)
	}
	var completedUUIDs []uuid.UUID
	for _, uuStr := range uuids {
		uu, err := uuid.Parse(uuStr)
		if err != nil {
			return fmt.Errorf("failed to parse uuid %q: %w", uuStr, err)
		}
		completedUUIDs = append(completedUUIDs, uu)
	}

	for _, uuid := range completedUUIDs {
		if !snaputil.SnapshotUUIDExists(sm.sourceInfo.Snapshots, uuid) {
			path := filepath.Join(uuiddir, uuid.String())
			sm.config.LogVerbose(1, "Deleting expired completion file %q", path)
			if err := sshutil.RemoveFile(ctx, sm.sshClient, path); err != nil {
				return fmt.Errorf("failed to delete completion file %q: %w", path, err)
			}
		} else {
			sm.config.LogVerbose(3, "Mirrored uuid file %q has not expired\n", uuid)
		}
	}

	return nil
}

func (sm *sshCompressedManager) Close() error {
	return sm.sshClient.Close()
}

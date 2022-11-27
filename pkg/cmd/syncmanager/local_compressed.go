package syncmanager

import (
	"compress/gzip"
	"compress/lzw"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"

	"github.com/tinyzimmer/btrsync/pkg/btrfs"
	"github.com/tinyzimmer/btrsync/pkg/cmd/config"
	"github.com/tinyzimmer/btrsync/pkg/cmd/snaputil"
)

type localCompressedManager struct {
	config     *Config
	sourceInfo *btrfs.RootInfo
	mirrorPath string
}

func NewLocalCompressedManager(cfg *Config, subvolInfo *btrfs.RootInfo) (Manager, error) {
	mirrorUrl, err := cfg.MirrorURL()
	if err != nil {
		return nil, err
	}
	mirrorPath := mirrorUrl.Path
	cfg.LogVerbose(0, "Initiating local compressed sync manager for %q with mirror URL: %s\n", cfg.FullSubvolumePath, mirrorPath)
	return &localCompressedManager{
		config:     cfg,
		sourceInfo: subvolInfo,
		mirrorPath: mirrorPath,
	}, nil
}

func (sm *localCompressedManager) Sync(ctx context.Context) error {
	path := filepath.Join(sm.mirrorPath, sm.config.SubvolumeIdentifier)
	sm.config.LogVerbose(0, "Syncing %s compressed mirror: %q\n", sm.config.MirrorFormat, path)
	if err := os.MkdirAll(path, 0755); err != nil {
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

func (sm *localCompressedManager) syncSnapshot(ctx context.Context, destination string, _, snap *btrfs.RootInfo) error {
	snapshotPath := filepath.Join(sm.config.SnapshotDirectory, snap.Name)
	uuidfile := filepath.Join(destination, OffsetDirectory, snap.UUID.String())
	destination = filepath.Join(destination, snap.Name+"."+string(sm.config.MirrorFormat))
	sm.config.LogVerbose(1, "Checking for snapshot completion file at %q\n", uuidfile)
	_, err := os.Stat(uuidfile)
	if err == nil {
		sm.config.LogVerbose(1, "Snapshot %q already synced, skipping\n", snap.Name)
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check for snapshot completion file: %s", err)
	}

	sm.config.LogVerbose(0, "Syncing snapshot %q to %q\n", snap.Path, destination)

	// Set up the destination
	f, err := os.Create(destination)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %s", err)
	}
	defer f.Close()

	// Set up the encoder
	var enc io.WriteCloser
	switch sm.config.MirrorFormat {
	case config.MirrorFormatGzip:
		enc = gzip.NewWriter(f)
	case config.MirrorFormatLzw:
		enc = lzw.NewWriter(f, lzw.LSB, 8)
	case config.MirrorFormatZlib:
		enc = gzip.NewWriter(f)
	case config.MirrorFormatZstd:
		enc, err = zstd.NewWriter(f)
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
	errors := make(chan error, 2)

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

	// Start the encoder to the destination
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer enc.Close()
		_, err := io.Copy(enc, pipe)
		if err != nil {
			errors <- fmt.Errorf("error copying to encoder: %w", err)
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
	if err := os.MkdirAll(filepath.Dir(uuidfile), 0755); err != nil {
		return fmt.Errorf("failed to create completion file directory: %s", err)
	}
	if err := os.WriteFile(uuidfile, []byte{}, 0644); err != nil {
		return fmt.Errorf("failed to create completion file: %s", err)
	}
	return nil
}

func (sm *localCompressedManager) Prune(ctx context.Context) error {
	destination := filepath.Join(sm.mirrorPath, sm.config.SubvolumeIdentifier)
	uuiddir := filepath.Join(destination, OffsetDirectory)
	sm.config.LogVerbose(2, "Listing compressed snapshots at %q\n", destination)

	files, err := os.ReadDir(destination)
	if err != nil {
		return fmt.Errorf("failed to read destination directory: %w", err)
	}

	var expired []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		snapshotName := strings.TrimSuffix(file.Name(), "."+string(sm.config.MirrorFormat))
		if !snaputil.SnapshotSliceContains(sm.sourceInfo.Snapshots, snapshotName) {
			sm.config.LogVerbose(1, "Marking snapshot %q for expiry\n", file.Name())
			expired = append(expired, file.Name())
		} else {
			sm.config.LogVerbose(3, "Mirrored snapshot %q has not expired\n", file.Name())
		}
	}

	for _, path := range expired {
		path = filepath.Join(destination, path)
		sm.config.LogVerbose(0, "Expiring mirrored snapshot %q\n", path)
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("error deleting snapshot file %q: %w", path, err)
		}
	}

	uuids, err := os.ReadDir(uuiddir)
	if err != nil {
		return fmt.Errorf("failed to read uuid directory: %w", err)
	}
	var completedUUIDs []uuid.UUID
	for _, uuStr := range uuids {
		if uuStr.IsDir() {
			continue
		}
		uu, err := uuid.Parse(uuStr.Name())
		if err != nil {
			return fmt.Errorf("failed to parse uuid %q: %w", uuStr.Name(), err)
		}
		completedUUIDs = append(completedUUIDs, uu)
	}

	for _, uuid := range completedUUIDs {
		if !snaputil.SnapshotUUIDExists(sm.sourceInfo.Snapshots, uuid) {
			path := filepath.Join(uuiddir, uuid.String())
			sm.config.LogVerbose(1, "Deleting expired completion file %q", path)
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to delete completion file %q: %w", path, err)
			}
		} else {
			sm.config.LogVerbose(3, "Mirrored snapshot uuid %q has not expired\n", uuid)
		}
	}

	return nil
}

func (sm *localCompressedManager) Close() error {
	return nil
}

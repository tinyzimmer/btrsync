// Package syncmanager provides a manager for syncing btrfs snapshots
// with a local or remote host.
package syncmanager

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/tinyzimmer/btrsync/cmd/btrsync/cmd/snaputil"
	"github.com/tinyzimmer/btrsync/pkg/btrfs"
	"github.com/tinyzimmer/btrsync/pkg/receive"
	"github.com/tinyzimmer/btrsync/pkg/receive/receivers/local"
)

type Config struct {
	Logger              *log.Logger
	Verbosity           int
	SubvolumeIdentifier string
	FullSubvolumePath   string
	SnapshotDirectory   string
	SnapshotName        string
	MirrorPath          string
}

func (c *Config) logLevel(level int, format string, args ...interface{}) {
	if c.Verbosity >= level {
		c.Logger.Printf(format, args...)
	}
}

func (c *Config) MirrorURL() (*url.URL, error) {
	u, err := url.Parse(c.MirrorPath)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "" {
		u.Scheme = "file"
	}
	return u, checkMirror(u)
}

func checkMirror(u *url.URL) error {
	switch u.Scheme {
	case "file":
		if u.Path == "" {
			return fmt.Errorf("mirror path cannot be empty")
		}
		if _, err := os.Stat(u.Path); err != nil {
			if os.IsNotExist(err) {
				// If the path does not exist, we can create it later
				return nil
			}
			return fmt.Errorf("error accessing mirror path: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported mirror scheme: %s", u.Scheme)
	}
}

type SyncManager struct {
	config    *Config
	rootInfo  *btrfs.RootInfo
	mirrorURL *url.URL
}

func New(cfg *Config) (*SyncManager, error) {
	mirrorURL, err := cfg.MirrorURL()
	if err != nil {
		return nil, err
	}
	cfg.logLevel(0, "Initiating sync manager for %q with mirror URL: %s\n", cfg.FullSubvolumePath, mirrorURL)
	info, err := snaputil.ResolveSubvolumeDetails(
		cfg.Logger,
		cfg.Verbosity,
		cfg.FullSubvolumePath,
		cfg.SnapshotDirectory,
		cfg.SnapshotName,
	)
	if err != nil {
		return nil, fmt.Errorf("error resolving subvolume details: %w", err)
	}
	return &SyncManager{cfg, info, mirrorURL}, nil
}

func (sm *SyncManager) Sync(ctx context.Context) error {
	sm.config.logLevel(1, "Ensuring mirror path is ready and accessible")
	mirror, err := sm.config.MirrorURL()
	if err != nil {
		return err
	}
	if err := sm.ensureLocalMirrorPath(ctx, mirror.Path); err != nil {
		return err
	}
	snaputil.SortSnapshots(sm.rootInfo.Snapshots, snaputil.SortAscending)
	for idx, snap := range sm.rootInfo.Snapshots {
		var parent *btrfs.RootInfo
		if idx == 0 {
			parent = nil
		} else {
			parent = sm.rootInfo.Snapshots[idx-1]
		}
		if err := sm.syncSnapshot(ctx, parent, snap); err != nil {
			return err
		}
	}
	return nil
}

func (sm *SyncManager) Prune(ctx context.Context) error {
	sm.config.logLevel(0, "Pruning expired snapshots from mirror: %s\n", sm.config.MirrorPath)
	return sm.pruneLocalMirror(ctx)
}

func (sm *SyncManager) syncSnapshot(ctx context.Context, parent, snap *btrfs.RootInfo) error {
	switch sm.mirrorURL.Scheme {
	case "file":
		return sm.syncSnapshotLocal(ctx, parent, snap)
	default:
		// This should never happen, but just in case
		return fmt.Errorf("unsupported mirror scheme: %s", sm.mirrorURL.Scheme)
	}
}

func (sm *SyncManager) syncSnapshotLocal(ctx context.Context, parent, snap *btrfs.RootInfo) error {
	var wg sync.WaitGroup

	mirror, err := sm.config.MirrorURL()
	if err != nil {
		return err
	}

	destination := filepath.Join(mirror.Path, sm.config.SubvolumeIdentifier)
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
		sm.config.logLevel(1, "Snapshot %q already synced to %q\n", snap.Path, destination)
		return nil
	} else if found {
		sm.config.logLevel(0, "Snapshot %q already exists at %q, but is not synced. Will try incremental send.\n", snap.Path, destination)
		sm.config.logLevel(0, "Searching for command offset to resume from")
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
		sm.config.logLevel(0, "Found stream diff offset at %d", offset)
		receiveOpts = append(receiveOpts, receive.FromOffset(offset))
	}

	sm.config.logLevel(0, "Syncing snapshot %q to %q\n", snap.Path, destination)

	pipeOpt, pipe, err := btrfs.SendToPipe()
	if err != nil {
		return fmt.Errorf("error creating send pipe: %w", err)
	}
	defer pipe.Close()
	errors := make(chan error, 2)

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
		errors <- nil
	}()

	err = receive.ProcessSendStream(pipe, receiveOpts...)
	if err != nil {
		return err
	}
	wg.Wait()
	return <-errors
}

func (sm *SyncManager) checkDestinationSnapshotLocal(ctx context.Context, snap *btrfs.RootInfo) (found, synced bool, err error) {
	mirror, err := sm.config.MirrorURL()
	if err != nil {
		return false, false, err
	}
	destination := filepath.Join(mirror.Path, sm.config.SubvolumeIdentifier, snap.Path)
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

func (sm *SyncManager) ensureLocalMirrorPath(ctx context.Context, path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return nil
	}
	if os.IsNotExist(err) {
		// Check if the destination is on a btrfs filesystem, we'll use a subvolume
		// if so, otherwise we'll use a directory.
		sm.config.logLevel(1, "Mirror path %q does not exist, creating\n", path)
		ok, err := btrfs.IsBtrfs(path)
		if err != nil {
			return fmt.Errorf("error checking if destination is btrfs: %w", err)
		}
		if !ok {
			return fmt.Errorf("local destination %s is not a btrfs filesystem (may be supported in the future)", path)
		}
		// Make a subvolume
		sm.config.logLevel(0, "Creating btrfs subvolume at %q\n", path)
		if err := btrfs.CreateSubvolume(path); err != nil {
			return fmt.Errorf("error creating subvolume at %q: %w", path, err)
		}
		return btrfs.SyncFilesystem(path)
	}
	return fmt.Errorf("error accessing mirror path: %w", err)
}

func (sm *SyncManager) pruneLocalMirror(ctx context.Context) error {
	mirror, err := sm.config.MirrorURL()
	if err != nil {
		return err
	}
	destination := filepath.Join(mirror.Path, sm.config.SubvolumeIdentifier)
	sm.config.logLevel(2, "Listing snapshots in tree at %q\n", mirror.Path)

	mirrorInfo, err := btrfs.SubvolumeSearch(btrfs.SearchWithPath(mirror.Path))
	if err != nil {
		return fmt.Errorf("error looking up information on mirror path: %w", err)
	}

	var tree *btrfs.RBRoot
	var retries int
	for tree == nil && retries <= 3 {
		if retries > 0 {
			sm.config.logLevel(1, "Error while building tree at %q, retrying: %s\n", mirror.Path, err)
		}
		tree, err = btrfs.BuildRBTree(mirror.Path)
		if err != nil {
			time.Sleep(time.Millisecond * 100)
		}
	}
	if err != nil {
		return fmt.Errorf("error building tree at %q: %w", mirror.Path, err)
	}
	tree = tree.FilterFromRoot(mirrorInfo.RootID)

	var expired []string
	tree.PreOrderIterate(func(info *btrfs.RootInfo, _ error) error {
		if info.Deleted || info.FullPath == "" || !strings.HasPrefix(info.FullPath, sm.config.SubvolumeIdentifier) {
			return nil
		}
		fullpath := filepath.Join(destination, info.Name)
		sm.config.logLevel(3, "Checking if mirrored snapshot %q is expired\n", fullpath)
		if !mirroredSnapshotExists(sm.rootInfo.Snapshots, info) {
			sm.config.logLevel(1, "Marking snapshot %q for expiry\n", fullpath)
			expired = append(expired, fullpath)
		} else {
			sm.config.logLevel(3, "Mirrored snapshot %q has not expired\n", fullpath)
		}
		return nil
	})

	for _, path := range expired {
		sm.config.logLevel(0, "Expiring mirrored snapshot %q\n", path)
		if err := btrfs.DeleteSubvolume(path); err != nil {
			return fmt.Errorf("error deleting subvolume %q: %w", path, err)
		}
	}

	return nil
}

func mirroredSnapshotExists(ss []*btrfs.RootInfo, s *btrfs.RootInfo) bool {
	for _, snap := range ss {
		if s.Name == snap.Name {
			return true
		}
	}
	return false
}

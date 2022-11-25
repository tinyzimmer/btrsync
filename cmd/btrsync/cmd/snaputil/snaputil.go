// Snaputil provides utility functions for working with snapshots.
package snaputil

import (
	"log"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tinyzimmer/btrsync/pkg/btrfs"
)

type SortOrder int

const (
	SortAscending SortOrder = iota
	SortDescending
)

// SortSnapshots will sort the given snapshots by their creation time in the given order.
func SortSnapshots(snapshots []*btrfs.RootInfo, order SortOrder) {
	sort.Slice(snapshots, func(i, j int) bool {
		if order == SortAscending {
			return snapshots[i].CreationTime.Before(snapshots[j].CreationTime)
		}
		return snapshots[i].CreationTime.After(snapshots[j].CreationTime)
	})
}

// ResolveSubvolumeDetails will lookup the information for a subvolume and all its corresponding snapshots.
// Snapshots are filtered by the given snapshot directory and name.
func ResolveSubvolumeDetails(logger *log.Logger, verbosity int, subvolumePath, snapshotDirectory, snapshotName string) (*btrfs.RootInfo, error) {
	// Find the root mount of the subvolume
	mount, err := btrfs.FindRootMount(subvolumePath)
	if err != nil {
		return nil, err
	}
	// Lookup informatin and all snapshots associated with the volume
	var info *btrfs.RootInfo
	var retries int
	for info == nil && retries <= 3 {
		if retries > 0 && verbosity >= 1 {
			logger.Printf("Retrying subvolume lookup after error: %v", err)
		}
		info, err = btrfs.SubvolumeSearch(
			btrfs.SearchWithRootMount(mount),
			btrfs.SearchWithSnapshots(),
			btrfs.SearchWithPath(subvolumePath),
		)
		if err != nil {
			time.Sleep(time.Millisecond * 100)
		}
	}
	if err != nil {
		return nil, err
	}
	managedSnaps := make([]*btrfs.RootInfo, 0)
	for _, snap := range info.Snapshots {
		if snap.Deleted {
			continue
		}
		if !strings.HasPrefix(snap.Name, snapshotName) {
			if verbosity >= 3 {
				logger.Printf("Skipping snapshot %q, does not match configured snapshot name %q\n", snap.Name, snapshotName)
			}
			continue
		}
		if !strings.HasSuffix(snapshotDirectory, filepath.Dir(snap.FullPath)) {
			if verbosity >= 3 {
				logger.Printf("Snapshot %q is not in the managed snapshot directory, skipping\n", snap.FullPath)
			}
			continue
		}
		if verbosity >= 3 {
			logger.Printf("Found managed snapshot %q\n", snap.FullPath)
		}
		managedSnaps = append(managedSnaps, snap)
	}
	info.Snapshots = managedSnaps
	return info, nil
}

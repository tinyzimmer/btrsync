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

package btrfs

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

type SearchOption func(*searchContext)

type searchContext struct {
	// The root mount point of the filesystem to search
	rootMount string
	// The root ID of the subvolume to search from.
	// If not provided, the search will start from the root of the filesystem.
	rootID uint64
	// The path to the subvolume to search.
	path string
	// The UUID of the subvolume to search.
	uuid uuid.UUID
	// The received UUID of the subvolume to search.
	receivedUUID uuid.UUID
	// Whether to search for snapshots.
	searchSnapshots bool
}

// SearchWithRootID searches for a subvolume starting from the given root ID.
func SearchWithRootID(id uint64) SearchOption {
	return func(opts *searchContext) {
		opts.rootID = id
	}
}

// SearchWithRootMount searches for a subvolume starting from the given root mount point.
// If not provided, the search will start from the root of the filesystem. You can use the
// FindRootMount function to find the root mount point of a given path.
func SearchWithRootMount(path string) SearchOption {
	return func(opts *searchContext) {
		opts.rootMount = path
	}
}

// SearchWithPath searches for a subvolume starting from the given path.
func SearchWithPath(path string) SearchOption {
	return func(opts *searchContext) {
		opts.path = path
	}
}

// SearchWithUUID searches for a subvolume with the given UUID.
func SearchWithUUID(uuid uuid.UUID) SearchOption {
	return func(opts *searchContext) {
		opts.uuid = uuid
	}
}

// SearchWithReceivedUUID searches for a subvolume with the given received UUID.
func SearchWithReceivedUUID(uuid uuid.UUID) SearchOption {
	return func(opts *searchContext) {
		opts.receivedUUID = uuid
	}
}

// SearchWithSnapshots searches for snapshots of the given subvolume and populates
// the results with them.
func SearchWithSnapshots() SearchOption {
	return func(opts *searchContext) {
		opts.searchSnapshots = true
	}
}

// SubvolumeSearch searches for a subvolume using the given options.
func SubvolumeSearch(opts ...SearchOption) (*RootInfo, error) {
	var ctx searchContext
	ctx.rootMount = "/"
	for _, opt := range opts {
		opt(&ctx)
	}

	// Find the root id
	if ctx.rootID == 0 {
		var err error
		if ctx.path != "" {
			var f *os.File
			f, err = os.OpenFile(ctx.path, os.O_RDONLY, os.ModeDir)
			if err != nil {
				return nil, err
			}
			defer f.Close()
			ctx.rootID, err = lookupRootIDFromFd(f.Fd())
		}
		if ctx.uuid != uuid.Nil {
			ctx.rootID, err = UUIDTreeLookupID(ctx.rootMount, ctx.uuid, LookupUUIDKeySubvol)
		}
		if ctx.receivedUUID != uuid.Nil {
			ctx.rootID, err = UUIDTreeLookupID(ctx.rootMount, ctx.receivedUUID, LookupUUIDKeyReceivedSubvol)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to find root id: %w", err)
		}
	}

	// Read the root item
	rootItem, err := lookupRootItem(ctx.rootMount, ctx.rootID)
	if err != nil {
		return nil, fmt.Errorf("failed to read root item: %w", err)
	}
	info := &RootInfo{
		RootID:             ObjectID(ctx.rootID),
		Flags:              rootItem.Flags,
		Generation:         rootItem.Generation,
		OriginalGeneration: rootItem.Otransid,
		CreationTime:       time.Unix(int64(rootItem.Ctime.Sec), int64(rootItem.Ctime.Nsec)),
		UUID:               rootItem.Uuid,
		ParentUUID:         rootItem.Parent_uuid,
		ReceivedUUID:       rootItem.Received_uuid,
	}
	if err != nil {
		return nil, fmt.Errorf("failed to convert root item to subvolume info: %w", err)
	}
	if ctx.path != "" {
		info.Path = ctx.path
	} else {
		info.Path, err = lookupPathFromSubvolumeID(ctx.rootMount, ctx.rootID)
		if err != nil {
			return nil, fmt.Errorf("failed to find path: %w", err)
		}
	}
	info.Name = filepath.Base(info.Path)
	if ctx.searchSnapshots {
		info.Snapshots = make([]*RootInfo, 0)
		tree, err := BuildRBTree(ctx.rootMount)
		if err != nil {
			return nil, fmt.Errorf("failed to build root tree: %w", err)
		}
		return info, tree.InOrderIterate(func(item *RootInfo, lastErr error) error {
			if item.ParentUUID == info.UUID {
				info.Snapshots = append(info.Snapshots, item)
			}
			return nil
		})
	}
	return info, nil
}

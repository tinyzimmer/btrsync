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
	"math"
	"os"
	"time"
)

// BuildRBTree builds a red-black tree from the subvolume root tree. Colors are
// currently not assigned as they are not needed for the current implementation.
func BuildRBTree(path string) (*RBRoot, error) {
	f, err := os.OpenFile(path, os.O_RDONLY, os.ModeDir)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	rootID, err := lookupRootIDFromFd(f.Fd())
	if err != nil {
		return nil, fmt.Errorf("failed to find root id: %w", err)
	}
	searchKey := SearchParams{
		Tree_id:      uint64(RootTreeObjectID),
		Min_objectid: rootID,
		Max_objectid: uint64(LastFreeObjectID),
		Min_offset:   0,
		Max_offset:   math.MaxUint64,
		Min_transid:  0,
		Max_transid:  math.MaxUint64,
		Min_type:     uint32(RootItemKey),
		Max_type:     uint32(RootBackrefKey),
		Nr_items:     4096,
	}
	tree := newRBRoot()
	err = walkBtrfsTreeFd(f.Fd(), searchKey, func(hdr SearchHeader, item TreeItem, lastErr error) error {
		if lastErr != nil {
			return lastErr
		}
		switch hdr.ItemType() {
		case RootItemKey:
			rootItem, err := item.RootItem()
			if err != nil {
				return fmt.Errorf("failed to decode root item: %w", err)
			}
			node := &RBNode{
				Info: &RootInfo{
					RootID:             ObjectID(hdr.Objectid),
					RootOffset:         hdr.Offset,
					Flags:              rootItem.Flags,
					Generation:         rootItem.Generation,
					OriginalGeneration: rootItem.Otransid,
					CreationTime:       time.Unix(int64(rootItem.Otime.Sec), int64(rootItem.Otime.Nsec)),
					SendTime:           time.Unix(int64(rootItem.Stime.Sec), int64(rootItem.Stime.Nsec)),
					ReceiveTime:        time.Unix(int64(rootItem.Rtime.Sec), int64(rootItem.Rtime.Nsec)),
					UUID:               rootItem.Uuid,
					ParentUUID:         rootItem.Parent_uuid,
					ReceivedUUID:       rootItem.Received_uuid,
					Item:               &rootItem,
				},
			}
			node.Info.RBNode = node
			if !tree.UpdateRoot(node.Info) {
				tree.InsertRoot(node.Info)
			}
		case RootBackrefKey:
			ref, name, err := item.RootRef()
			if err != nil {
				return fmt.Errorf("failed to decode root ref: %w", err)
			}
			node := &RBNode{
				Info: &RootInfo{
					RootID:  ObjectID(hdr.Objectid),
					DirID:   ref.Dirid,
					RefTree: ObjectID(hdr.Offset),
					Name:    name,
					Ref:     &ref,
				},
			}
			node.Info.RBNode = node
			if !tree.UpdateRoot(node.Info) {
				tree.InsertRoot(node.Info)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk root tree: %w", err)
	}
	if err := tree.resolveFullPaths(f.Fd(), rootID); err != nil {
		return nil, fmt.Errorf("failed to resolve full paths: %w", err)
	}
	return tree, nil
}

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
	"strings"
)

func lookupInoPath(fd uintptr, info *RootInfo) (path string, err error) {
	var args inoLookupArgs
	args.Treeid = uint64(info.RefTree)
	args.Objectid = info.DirID
	err = callWriteIoctl(fd, BTRFS_IOC_INO_LOOKUP, &args)
	if err != nil {
		err = fmt.Errorf("failed to lookup inode path: %w", err)
		return
	}
	name := stringFromLookupArgsName(args.Name)
	if name == "" {
		path = info.Name
	} else {
		path = name + info.Name
	}
	return
}

func lookupRootItem(path string, rootID uint64) (*BtrfsRootItem, error) {
	params := SearchParams{
		Tree_id:      uint64(RootTreeObjectID),
		Min_objectid: rootID,
		Max_objectid: rootID,
		Max_offset:   math.MaxUint64,
		Max_transid:  math.MaxUint64,
		Min_type:     uint32(RootItemKey),
		Max_type:     uint32(RootItemKey),
	}
	var found *BtrfsRootItem
	err := WalkBtrfsTree(path, params, func(hdr SearchHeader, item TreeItem, lastErr error) error {
		if lastErr != nil {
			return lastErr
		}
		if hdr.Objectid > rootID {
			return ErrNotFound
		}
		if hdr.Objectid == rootID && hdr.Type == uint32(RootItemKey) {
			rootItem, err := item.RootItem()
			if err != nil {
				return err
			}
			found = &rootItem
			return ErrStopWalk
		}
		return nil
	})
	if found == nil {
		err = ErrNotFound
	}
	return found, err
}

func lookupPathFromSubvolumeID(root string, id uint64) (string, error) {
	if id == uint64(FSTreeObjectID) {
		return root, nil
	}
	f, err := os.OpenFile(root, os.O_RDONLY, os.ModeDir)
	if err != nil {
		return "", err
	}
	defer f.Close()
	params := SearchParams{
		Tree_id:      uint64(RootTreeObjectID),
		Min_objectid: id,
		Max_objectid: id,
		Min_offset:   0,
		Max_offset:   math.MaxUint64,
		Min_transid:  0,
		Max_transid:  math.MaxUint64,
		Min_type:     uint32(RootBackrefKey),
		Max_type:     uint32(RootBackrefKey),
		Nr_items:     1,
	}
	var path string
	err = walkBtrfsTreeFd(f.Fd(), params, func(hdr SearchHeader, item TreeItem, lastErr error) error {
		if lastErr != nil {
			return lastErr
		}
		if hdr.ItemType() == RootBackrefKey {
			ref, name, err := item.RootRef()
			if err != nil {
				return err
			}
			if ref.Dirid != uint64(FirstFreeObjectID) {
				var lookupArgs inoLookupArgs
				lookupArgs.Treeid = hdr.Offset
				lookupArgs.Objectid = ref.Dirid
				err = callWriteIoctl(f.Fd(), BTRFS_IOC_INO_LOOKUP, &lookupArgs)
				if err != nil {
					return fmt.Errorf("failed to lookup inode: %w", err)
				}
				path = stringFromLookupArgsName(lookupArgs.Name)
				return ErrStopWalk
			}
			path = name
			return ErrStopWalk
		}
		return nil
	})
	return path, err
}

func lookupRootIDFromFd(fd uintptr) (uint64, error) {
	args := &inoLookupArgs{
		Treeid:   0,
		Objectid: uint64(FirstFreeObjectID),
	}
	err := callWriteIoctl(fd, BTRFS_IOC_INO_LOOKUP, args)
	if err != nil {
		return 0, err
	}
	return args.Treeid, nil
}

func stringFromLookupArgsName(bb [4080]int8) string {
	var sb strings.Builder
	for _, b := range bb {
		if b == 0 {
			break
		}
		sb.WriteByte(byte(b))
	}
	return sb.String()
}

func convertSearchArgsBuf(args *SearchArgs) []byte {
	buf := make([]byte, len(args.Buf))
	for i, v := range args.Buf {
		buf[i] = byte(v)
	}
	return buf
}

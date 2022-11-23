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
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

// TreeIterFunc is a function that is called for each item found in the tree. If the function returns
// any error it is passed as lastErr to the next call of the function. If the error wraps the
// ErrStopIteration error, the iteration is stopped.
type TreeIterFunc func(hdr SearchHeader, item TreeItem, lastErr error) error

var ErrStopWalk = fmt.Errorf("stop btrfs walk")

// WalkBtrfsTree walks the Btrfs tree at the given path with the given search arguments.
// The TreeIterFunc is called for each item found in the tree.
func WalkBtrfsTree(path string, params SearchParams, fn TreeIterFunc) error {
	f, err := os.OpenFile(path, os.O_RDONLY, os.ModeDir)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()
	return walkBtrfsTreeFd(f.Fd(), params, fn)
}

func walkBtrfsTreeFd(fd uintptr, params SearchParams, fn TreeIterFunc) error {
	var lastErr error
	minType := params.Min_type
	maxType := params.Max_type
	args := SearchArgs{Key: params}
	for {
		args.Key.Nr_items = 4096
		err := callWriteIoctl(fd, BTRFS_IOC_TREE_SEARCH, &args)
		if err != nil {
			return fmt.Errorf("failed to call ioctl: %w", err)
		}
		if args.Key.Nr_items == 0 {
			return lastErr
		}
		r := bytes.NewReader(convertSearchArgsBuf(&args))
		for i := 0; i < int(args.Key.Nr_items); i++ {
			var hdr SearchHeader
			if err = binary.Read(r, binary.LittleEndian, &hdr); err != nil {
				return fmt.Errorf("failed to read search header: %w", err)
			}
			databuf := make([]byte, hdr.Len)
			if _, err = io.ReadFull(r, databuf); err != nil {
				return fmt.Errorf("failed to read item data: %w", err)
			}
			item := TreeItem{Data: databuf}
			if hdr.Type == uint32(RootBackrefKey) || hdr.Type == uint32(RootRefKey) {
				ref, _, err := item.RootRef()
				if err != nil {
					return fmt.Errorf("failed to decode root backref: %w", err)
				}
				namebuf := databuf[len(databuf)-int(ref.Len):]
				item.Name = string(namebuf)
				item.Data = bytes.TrimSuffix(item.Data, namebuf)
			}
			lastErr = fn(hdr, item, lastErr)
			if lastErr != nil && errors.Is(lastErr, ErrStopWalk) {
				return nil
			}
			args.Key.Min_objectid = hdr.Objectid
			args.Key.Min_type = hdr.Type
			args.Key.Min_offset = hdr.Offset
		}
		args.Key.Min_offset++
		if args.Key.Min_offset == 0 {
			args.Key.Min_type++
			if args.Key.Min_type > maxType {
				args.Key.Min_type = minType
				args.Key.Min_objectid++
				if args.Key.Min_objectid > args.Key.Max_objectid {
					break
				}
			}
		}
	}
	return lastErr
}

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
	"os"
	"path/filepath"
	"syscall"
)

type SnapshotOption func(*volumeArgsV2) error

// WithSnapshotName sets the name of the snapshot to be created.
func WithSnapshotName(name string) SnapshotOption {
	return func(args *volumeArgsV2) error {
		args.Name = toSnapInt8Array(name)
		return nil
	}
}

// WithSnapshotPath sets an absolute path for the snapshot to be created.
func WithSnapshotPath(path string) SnapshotOption {
	return func(args *volumeArgsV2) error {
		var err error
		path, err = filepath.Abs(path)
		if err != nil {
			return err
		}
		topdir := filepath.Dir(path)
		name := filepath.Base(path)
		if err := os.MkdirAll(topdir, 0755); err != nil {
			return err
		}
		dest, err := os.OpenFile(topdir, os.O_RDONLY, os.ModeDir)
		if err != nil {
			return err
		}
		args.Fd = int64(dest.Fd())
		args.Name = toSnapInt8Array(name)
		return nil
	}
}

// WithReadOnlySnapshot sets the snapshot to be read-only.
func WithReadOnlySnapshot() SnapshotOption {
	return func(args *volumeArgsV2) error {
		args.Flags |= SubvolReadOnly
		return nil
	}
}

// CreateSnapshot creates a snapshot of the given subvolume with the given
// options.
func CreateSnapshot(source string, opts ...SnapshotOption) error {
	var err error
	source, err = filepath.Abs(source)
	if err != nil {
		return err
	}
	src, err := os.OpenFile(source, os.O_RDONLY, os.ModeDir)
	if err != nil {
		return err
	}
	defer src.Close()
	fddst := int64(src.Fd())
	args := &volumeArgsV2{
		Fd: int64(src.Fd()),
	}
	for _, opt := range opts {
		if err := opt(args); err != nil {
			return err
		}
	}
	if args.Fd != int64(src.Fd()) {
		// This is a bit of a hack...
		// If the user specified a full path destination, the ioctl needs to be called
		// at the parent directory of the destination while the fd in the arguments should
		// remain the fd of the source.
		fddst = args.Fd
		args.Fd = int64(src.Fd())
		defer syscall.Close(int(fddst))
	}
	if err := callWriteIoctl(uintptr(fddst), BTRFS_IOC_SNAP_CREATE_V2, args); err != nil {
		return err
	}
	return SyncFilesystem(source)
}

// DeleteSnapshot deletes the given snapshot.
func DeleteSnapshot(path string) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	topdir := filepath.Dir(path)
	name := filepath.Base(path)
	f, err := os.OpenFile(topdir, os.O_RDONLY, os.ModeDir)
	if err != nil {
		return err
	}
	defer f.Close()
	args := &volumeArgsV2{
		Fd:      int64(f.Fd()),
		Transid: 0,
		Flags:   0,
		Name:    toSnapInt8Array(name),
	}
	return callWriteIoctl(uintptr(f.Fd()), BTRFS_IOC_SNAP_DESTROY_V2, args)
}

func toSnapInt8Array(s string) [4040]int8 {
	var a [4040]int8
	for i := range s {
		a[i] = int8(s[i])
	}
	return a
}

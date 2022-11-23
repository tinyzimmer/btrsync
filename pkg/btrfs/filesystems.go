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
	"bufio"
	"io"
	"os"
	"strings"
)

// Filesystem represents a mounted BTRFS filesystem.
type Filesystem struct {
	Path         string
	Device       string
	MountOptions []string
}

// ListFilesystems returns a list of mounted BTRFS filesystems.
func ListFilesystems() ([]*Filesystem, error) {
	return listMounts()
}

// GetInfo returns metadata about the filesystem.
func (f *Filesystem) GetInfo() (*FilesystemInfo, error) {
	return GetFilesystemInfo(f.Path)
}

// GetDevice returns the device information for the filesystem.
func (f *Filesystem) GetDevice() (*DeviceInfo, error) {
	return GetDeviceInfo(f.Device)
}

// Sync runs an I/O sync on the filesystem.
func (f *Filesystem) Sync() error { return SyncFilesystem(f.Path) }

// Snapshot creates a snapshot of this filesystem at the given path (assumed to be based on the
// subvolume of this filesystem). If readonly is true, the snapshot will be read-only.
func (f *Filesystem) Snapshot(opts ...SnapshotOption) error {
	return CreateSnapshot(f.Path, opts...)
}

func listMounts() ([]*Filesystem, error) {
	f, err := os.Open("/proc/mounts")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return listMountsFromReader(f)
}

func listMountsFromReader(f io.Reader) ([]*Filesystem, error) {
	s := bufio.NewScanner(f)
	var out []*Filesystem
	for s.Scan() {
		fields := strings.Fields(s.Text())
		if len(fields) < 3 {
			// This is not a line we can parse for a filesystem
			continue
		}
		if fields[2] != "btrfs" {
			// This is not a btrfs filesystem
			continue
		}
		out = append(out, &Filesystem{
			Path:         fields[1],
			Device:       fields[0],
			MountOptions: strings.Split(fields[3], ","),
		})
	}
	return out, nil
}

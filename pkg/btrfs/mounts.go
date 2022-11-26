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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// BtrfsMount represents a mounted Btrfs filesystem.
type BtrfsMount struct {
	Path    string
	Device  string
	Options []string
}

func (b *BtrfsMount) String() string {
	return fmt.Sprintf("%s on %s type btrfs (%s)", b.Device, b.Path, strings.Join(b.Options, ","))
}

func (b *BtrfsMount) DeviceInfo() (*DeviceInfo, error) {
	return getDeviceInfoFromRoot(b.Path)
}

func (b *BtrfsMount) DeviceStats() (*DeviceStats, error) {
	return getDeviceStatsFromRoot(b.Path)
}

var (
	// ErrRootMountNotFound is returned when a root mount cannot be found for a given path.
	ErrRootMountNotFound = fmt.Errorf("could not find root mount for path")
)

// IsBtrfs returns true if the given path is a btrfs mount.
func IsBtrfs(path string) (bool, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return false, err
	}
	_, err = FindRootMount(path)
	if err != nil {
		if errors.Is(err, ErrRootMountNotFound) {
			return false, nil
		}
	}
	return true, nil
}

// FindRootMount returns the root btrfs mount for the given path.
func FindRootMount(path string) (*BtrfsMount, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	mounts, err := ListBtrfsMounts()
	if err != nil {
		return nil, err
	}
	var rootMount *BtrfsMount
	for _, mount := range mounts {
		if strings.HasPrefix(path, mount.Path) {
			// If we find a mount that is a prefix of the path, we need to make sure
			// it is the longest prefix. If we find a longer prefix, we need to use
			// that instead.
			if rootMount == nil || len(mount.Path) > len(rootMount.Path) {
				rootMount = mount
			}
		}
	}
	if rootMount == nil {
		return nil, fmt.Errorf("%w %s", ErrRootMountNotFound, path)
	}
	return rootMount, nil
}

// FindMountForDevice returns the mount for the given device.
func FindMountForDevice(device string) (*BtrfsMount, error) {
	mounts, err := ListBtrfsMounts()
	if err != nil {
		return nil, err
	}
	for _, mount := range mounts {
		if mount.Device == device {
			return mount, nil
		}
	}
	return nil, fmt.Errorf("could not find mount for device %s", device)
}

// FindDeviceForMount returns the device for the given mount. In most circumstances
// this is analogous (but more expensive than) to calling FindRootMount.
func FindDeviceForMount(mount string) (string, error) {
	mounts, err := ListBtrfsMounts()
	if err != nil {
		return "", err
	}
	for _, m := range mounts {
		if m.Path == mount {
			return m.Device, nil
		}
	}
	return "", fmt.Errorf("could not find device for mount %s", mount)
}

// ListBtrfsMounts returns a list of all btrfs mounts on the system.
func ListBtrfsMounts() ([]*BtrfsMount, error) {
	f, err := os.Open("/proc/mounts")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var mounts []*BtrfsMount
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue
		}
		if fields[2] != "btrfs" {
			continue
		}
		mount := BtrfsMount{
			Path:   fields[1],
			Device: fields[0],
		}
		if len(fields) >= 4 {
			mount.Options = strings.Split(fields[3], ",")
		}
		mounts = append(mounts, &mount)

	}
	return mounts, nil
}

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
func FindRootMount(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	mounts, err := ListBtrfsMounts()
	if err != nil {
		return "", err
	}
	var rootMount string
	for _, mount := range mounts {
		if strings.HasPrefix(path, mount) {
			// If we find a mount that is a prefix of the path, we need to make sure
			// it is the longest prefix. If we find a longer prefix, we need to use
			// that instead.
			if len(mount) > len(rootMount) {
				rootMount = mount
			}
		}
	}
	if rootMount == "" {
		return "", fmt.Errorf("%w %s", ErrRootMountNotFound, path)
	}
	return rootMount, nil
}

// ListBtrfsMounts returns a list of all btrfs mounts on the system.
func ListBtrfsMounts() ([]string, error) {
	f, err := os.Open("/proc/mounts")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var mounts []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue
		}
		if fields[2] == "btrfs" {
			mounts = append(mounts, fields[1])
		}
	}
	return mounts, nil
}

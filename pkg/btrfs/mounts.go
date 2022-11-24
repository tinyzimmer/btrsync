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

var (
	ErrRootMountNotFound = fmt.Errorf("could not find root mount for path")
)

// FindRootMount returns the root btrfs mount for the given path.
func FindRootMount(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	f, err := os.Open("/proc/mounts")
	if err != nil {
		return "", err
	}
	defer f.Close()
	var rootMount string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue
		}
		if fields[2] != "btrfs" {
			continue
		}
		if strings.HasPrefix(path, fields[1]) {
			// If we find a mount that is a prefix of the path, we need to make sure
			// it is the longest prefix. If we find a longer prefix, we need to use
			// that instead.
			if len(fields[1]) > len(rootMount) {
				rootMount = fields[1]
			}
		}
	}
	if rootMount == "" {
		return "", fmt.Errorf("%w %s", ErrRootMountNotFound, path)
	}
	return rootMount, nil
}

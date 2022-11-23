package btrfs

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
		return "", fmt.Errorf("could not find root mount for path %s", path)
	}
	return rootMount, nil
}

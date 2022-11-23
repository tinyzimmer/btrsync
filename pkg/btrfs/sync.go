package btrfs

import "os"

// SyncFilesystem runs an I/O sync on the filesystem at the given path.
// If the path is not a BTRFS filesystem, an error will be returned.
func SyncFilesystem(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return ioctlUnsafe(f.Fd(), BTRFS_IOC_SYNC, nil)
}

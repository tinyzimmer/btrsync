package btrfs

import (
	"os"
	"path/filepath"
	"syscall"

	"github.com/google/uuid"
)

// IsSubvolume returns true if the given path is a subvolume.
func IsSubvolume(path string) (bool, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return false, err
	}
	var statfs syscall.Statfs_t
	err = syscall.Statfs(path, &statfs)
	if err != nil {
		return false, err
	}
	return statfs.Type == BTRFS_SUPER_MAGIC, nil
}

// CreateSubvolume creates a subvolume at the given path.
func CreateSubvolume(path string) error {
	path, err := filepath.Abs(path)
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
	args := &volumeArgsV2{
		Fd:   int64(dest.Fd()),
		Name: toSnapInt8Array(name),
	}
	return callWriteIoctl(dest.Fd(), BTRFS_IOC_SUBVOL_CREATE_V2, args)
}

// SetReceivedSubvolume sets the received UUID and ctransid for a subvolume. This
// method is intended for use by receive operations.
func SetReceivedSubvolume(path string, uuid uuid.UUID, ctransid uint64) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_RDONLY, os.ModeDir)
	if err != nil {
		return err
	}
	defer f.Close()
	args := &receivedSubvolArgs{
		Uuid:     uuidToInt8Array(uuid),
		Stransid: ctransid,
	}
	return callWriteIoctl(f.Fd(), BTRFS_IOC_SET_RECEIVED_SUBVOL, args)
}

// SetSubvolumeReadOnly sets the read-only status of the subvolume at the given path to
// readonly.
func SetSubvolumeReadOnly(path string, readonly bool) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_RDONLY, os.ModeDir)
	if err != nil {
		return err
	}
	defer f.Close()
	var flags uint64
	err = ioctlUint64(f.Fd(), BTRFS_IOC_SUBVOL_GETFLAGS, &flags)
	if err != nil {
		return err
	}
	if readonly {
		flags |= SubvolReadOnly
	} else {
		flags = flags &^ SubvolReadOnly
	}
	return ioctlUint64(f.Fd(), BTRFS_IOC_SUBVOL_SETFLAGS, &flags)
}

// DeleteSubvolume deletes the subvolume at the given path. If the subvolume
// is read-only then it will be made read-write before deletion.
func DeleteSubvolume(path string) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_RDONLY, os.ModeDir)
	if err != nil {
		return err
	}
	// Check if readonly flag is set - if so, remove it
	var flags uint64
	err = ioctlUint64(f.Fd(), BTRFS_IOC_SUBVOL_GETFLAGS, &flags)
	if err != nil {
		return err
	}
	if flags&SubvolReadOnly != 0 {
		flags = flags &^ SubvolReadOnly
		if err := ioctlUint64(f.Fd(), BTRFS_IOC_SUBVOL_SETFLAGS, &flags); err != nil {
			return err
		}
	}
	return os.RemoveAll(path)
}

// IsSubvolumeReadOnly returns true if the subvolume at the given path is read-only.
func IsSubvolumeReadOnly(path string) (bool, error) {
	f, err := os.OpenFile(path, os.O_RDONLY, os.ModeDir)
	if err != nil {
		return false, err
	}
	defer f.Close()
	return isSubvolumeReadOnlyFd(f.Fd())
}

func isSubvolumeReadOnlyFd(fd uintptr) (bool, error) {
	var flags uint64
	err := ioctlUint64(fd, BTRFS_IOC_SUBVOL_GETFLAGS, &flags)
	if err != nil {
		return false, err
	}
	return flags&SubvolReadOnly != 0, nil
}

func uuidToInt8Array(uuid uuid.UUID) [16]int8 {
	var arr [16]int8
	for i, b := range uuid {
		arr[i] = int8(b)
	}
	return arr
}

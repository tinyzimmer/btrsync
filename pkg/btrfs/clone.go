package btrfs

import "os"

func Clone(src string, dest string, srcOffset uint64, destOffset uint64, size uint64) error {
	return clone(src, dest, srcOffset, destOffset, size)
}

func clone(src string, dest string, srcOffset uint64, destOffset uint64, size uint64) error {
	srcFile, err := os.OpenFile(src, os.O_RDONLY, 0600)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	destFile, err := os.OpenFile(dest, os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	var args cloneRangeArgs
	args.Src_fd = int64(srcFile.Fd())
	args.Src_offset = srcOffset
	args.Src_length = size
	args.Dest_offset = destOffset
	return callWriteIoctl(destFile.Fd(), BTRFS_IOC_CLONE_RANGE, &args)
}

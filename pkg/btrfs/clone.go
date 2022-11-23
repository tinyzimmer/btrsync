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

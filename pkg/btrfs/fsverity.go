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
	"unsafe"
)

// EnableVerity enables fs-verity on a path.
func EnableVerity(path string, algorithm uint32, blockSize uint32, salt []byte, sig []byte) error {
	return enableVerity(path, algorithm, blockSize, salt, sig)
}

func enableVerity(path string, algorithm uint32, blockSize uint32, salt []byte, sig []byte) error {
	args := fsVerityEnableArg{
		Version:        1,
		Hash_algorithm: algorithm,
		Block_size:     blockSize,
		Salt_size:      uint32(len(salt)),
		Salt_ptr:       uint64(uintptr(unsafe.Pointer(&salt[0]))),
		Sig_size:       uint32(len(sig)),
		Sig_ptr:        uint64(uintptr(unsafe.Pointer(&sig[0]))),
	}
	f, err := os.OpenFile(path, os.O_RDONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	return callWriteIoctl(f.Fd(), FS_IOC_ENABLE_VERITY, &args)
}

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
	"bytes"
	"encoding/binary"
	"fmt"
	"syscall"
	"unsafe"
)

func callReadIoctl[T any](fd uintptr, c IoctlCmd, out *T) error {
	buf := make([]byte, c.Size())
	if err := ioctlBytes(fd, c, buf); err != nil {
		return err
	}
	return decodeStructure(buf, out)
}

func callWriteIoctl[T any](fd uintptr, c IoctlCmd, data *T) error {
	buf, err := encodeStructure(data)
	if err != nil {
		return err
	}
	err = ioctlBytes(fd, c, buf)
	if err != nil {
		return err
	}
	return decodeStructure(buf, data)
}

// decodeStructure decodes a structure from a byte slice.
func decodeStructure[T any](data []byte, out *T) error {
	return binary.Read(bytes.NewReader(data), binary.LittleEndian, out)
}

// encodeStructure encodes a structure into a byte slice.
func encodeStructure[T any](data *T) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ioctlUint64 sends an ioctl command with a uint64.
func ioctlUint64(fd uintptr, name IoctlCmd, data *uint64) error {
	return ioctlUnsafe(fd, name, unsafe.Pointer(data))
}

// ioctlBytes sends an ioctl command with a byte slice.
func ioctlBytes(fd uintptr, name IoctlCmd, data []byte) error {
	return ioctlUnsafe(fd, name, unsafe.Pointer(&data[0]))
}

// ioctlUnsafe sends an ioctl command with an unsafe.Pointer.
func ioctlUnsafe(fd uintptr, name IoctlCmd, data unsafe.Pointer) error {
	return ioctl(fd, name, uintptr(data))
}

// ioctl sends an ioctl command.
func ioctl(fd uintptr, name IoctlCmd, data uintptr) error {
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(name), data)
	if err != 0 {
		return fmt.Errorf("ioctl %s failed: %w", name.String(), syscall.Errno(err))
	}
	return nil
}

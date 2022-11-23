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
	"errors"
	"fmt"
	"os"
	"unsafe"
)

var (
	ErrEncryptionNotSupported = errors.New("encryption not supported")
)

// EncodedWriteOp is an operation to write encoded data to a file.
type EncodedWriteOp struct {
	Offset              uint64
	Data                []byte
	UnencodedFileLength uint64
	UnencodedLength     uint64
	UnencodedOffset     uint64
	Compression         CompressionType
	Encryption          uint32 // Not supported yet
}

// Decompress decompresses the data in the EncodedWriteOp.
func (e *EncodedWriteOp) Decompress() ([]byte, error) {
	switch e.Compression {
	case CompressionNone:
		return e.Data, nil
	case CompressionZLib:
		return decompressZlip(e.Data)
	case CompressionLZO4k, CompressionLZO8k, CompressionLZO16k, CompressionLZO32k, CompressionLZO64k:
		return decompressLzo(e.Data, len(e.Data), int(e.UnencodedLength))
	case CompressionZSTD:
		return decompressZstd(e.Data)
	default:
		return nil, fmt.Errorf("Decompress: unknown compression type %d", e.Compression)
	}
}

// EncodedWrite writes encoded data to a file via ioctl.
func EncodedWrite(path string, op *EncodedWriteOp) error {
	if op.Encryption != 0 {
		return fmt.Errorf("EncodedWrite: %w", ErrEncryptionNotSupported)
	}
	args := encodedIOArgs{
		Iov: &ioVec{
			IovBase: uintptr(unsafe.Pointer(&op.Data[0])),
			IovLen:  uint64(len(op.Data)),
		},
		Iovcnt:           1,
		Offset:           int64(op.Offset),
		Len:              op.UnencodedFileLength,
		Unencoded_len:    op.UnencodedLength,
		Unencoded_offset: op.UnencodedOffset,
		Compression:      uint32(op.Compression),
		Encryption:       op.Encryption,
	}
	f, err := os.OpenFile(path, os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	return callWriteIoctl(f.Fd(), BTRFS_IOC_ENCODED_WRITE, &args)
}

type encodedIOArgs struct {
	Iov              *ioVec
	Iovcnt           uint64
	Offset           int64
	Flags            uint64
	Len              uint64
	Unencoded_len    uint64
	Unencoded_offset uint64
	Compression      uint32
	Encryption       uint32
	Reserved         [64]uint8
}

type ioVec struct {
	IovBase uintptr
	IovLen  uint64
}

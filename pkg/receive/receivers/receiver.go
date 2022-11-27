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

// Package receivers exposes the interface for receiving data from a btrfs send stream.
// Subpackages contain implementations for different types of receivers.
package receivers

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tinyzimmer/btrsync/pkg/btrfs"
	"github.com/tinyzimmer/btrsync/pkg/sendstream"
)

// ErrUnsupported should be returned by a receiver if it does not support the given
// operation. For operations that support a fallback, the receiver's fallback function
// will be called (e.g. EncodedWrite -> Write).
var ErrUnsupported = fmt.Errorf("unsupported operation for receiver")

// ErrSkipCommand is returned by a receiver if it does not want to handle a command.
// This is useful for receivers that want to handle a subset of commands, but not all.
// It can also be returned by a PreOp function to skip the respective method call.
var ErrSkipCommand = fmt.Errorf("skip command")

// Receiver is the interface for receiving data from a btrfs send stream.
type Receiver interface {
	Subvol(ctx ReceiveContext, path string, uuid uuid.UUID, ctransid uint64) error
	Snapshot(ctx ReceiveContext, path string, uuid uuid.UUID, ctransid uint64, cloneUUID uuid.UUID, cloneCtransid uint64) error
	Mkfile(ctx ReceiveContext, path string, ino uint64) error
	Mkdir(ctx ReceiveContext, path string, ino uint64) error
	Mknod(ctx ReceiveContext, path string, ino uint64, mode uint32, rdev uint64) error
	Mkfifo(ctx ReceiveContext, path string, ino uint64) error
	Mksock(ctx ReceiveContext, path string, ino uint64) error
	Symlink(ctx ReceiveContext, path string, ino uint64, linkTo string) error
	Rename(ctx ReceiveContext, oldPath string, newPath string) error
	Link(ctx ReceiveContext, path string, linkTo string) error
	Unlink(ctx ReceiveContext, path string) error
	Rmdir(pctx ReceiveContext, ath string) error
	Write(ctx ReceiveContext, path string, offset uint64, data []byte) error
	EncodedWrite(ctx ReceiveContext, path string, op *btrfs.EncodedWriteOp) error
	Clone(ctx ReceiveContext, path string, offset uint64, len uint64, cloneUUID uuid.UUID, cloneCtransid uint64, clonePath string, cloneOffset uint64) error
	SetXattr(ctx ReceiveContext, path string, name string, data []byte) error
	RemoveXattr(ctx ReceiveContext, path string, name string) error
	Truncate(ctx ReceiveContext, path string, size uint64) error
	Chmod(ctx ReceiveContext, path string, mode uint64) error
	Chown(pctx ReceiveContext, path string, uid uint64, gid uint64) error
	Utimes(ctx ReceiveContext, path string, atime, mtime, ctime time.Time) error
	UpdateExtent(ctx ReceiveContext, path string, fileOffset uint64, tmpSize uint64) error
	EnableVerity(ctx ReceiveContext, path string, algorithm uint8, blockSize uint32, salt []byte, sig []byte) error
	Fallocate(ctx ReceiveContext, path string, mode uint32, offset uint64, len uint64) error
	Fileattr(ctx ReceiveContext, path string, attr uint32) error
	FinishSubvolume(ctx ReceiveContext) error
}

// PreOpReceiver can be implemented by receivers that need to perform some action before
// a btrfs send operation is performed.
type PreOpReceiver interface {
	Receiver

	PreOp(ctx ReceiveContext, hdr sendstream.CmdHeader, attrs sendstream.CmdAttrs) error
}

// PostOpReceiver can be implemented by receivers that need to perform some action after
// a btrfs send operation is performed.
type PostOpReceiver interface {
	Receiver

	PostOp(ctx ReceiveContext, hdr sendstream.CmdHeader, attrs sendstream.CmdAttrs) error
}

// ReceiveContext is the context passed to a receiver for each operation.
type ReceiveContext interface {
	context.Context

	// CurrentOffset returns the current offset in the stream.
	CurrentOffset() uint64
	// CurrentSubvolume returns the current subvolume being received.
	CurrentSubvolume() *sendstream.ReceivingSubvolume
	// ResolvePath returns the absolute path for the given path in the current subvolume.
	ResolvePath(path string) string
	// LogVerbose will emit a log message at the given verbosity level.
	LogVerbose(level int, format string, args ...interface{})
}

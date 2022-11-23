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

package receivers

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/tinyzimmer/btrsync/pkg/btrfs"
)

var ErrUnsupported = fmt.Errorf("unsupported operation for receiver")

type Receiver interface {
	Subvol(ctx ReceiveContext, path string, uuid uuid.UUID, ctransid uint64) error
	Snapshot(ctx ReceiveContext, path string, uuid uuid.UUID, ctransid uint64, cloneUUID uuid.UUID, cloneCtransid uint64) error
	Mkfile(ctx ReceiveContext, path string, ino uint64) error
	Mkdir(ctx ReceiveContext, path string, ino uint64) error
	Mknod(ctx ReceiveContext, path string, ino uint64, mode fs.FileMode, rdev uint64) error
	Mkfifo(ctx ReceiveContext, path string, ino uint64) error
	Mksock(ctx ReceiveContext, path string, ino uint64) error
	Symlink(ctx ReceiveContext, path string, ino uint64, linkTo string) error
	Rename(ctx ReceiveContext, oldPath string, newPath string) error
	Link(ctx ReceiveContext, path string, linkTo string) error
	Unlink(ctx ReceiveContext, path string) error
	Rmdir(pctx ReceiveContext, ath string) error
	Write(ctx ReceiveContext, path string, offset uint64, data []byte) error
	EncodedWrite(ctx ReceiveContext, path string, op *btrfs.EncodedWriteOp, forceDecompress bool) error
	Clone(ctx ReceiveContext, path string, offset uint64, len uint64, cloneUUID uuid.UUID, cloneCtransid uint64, clonePath string, cloneOffset uint64) error
	SetXattr(ctx ReceiveContext, path string, name string, data []byte) error
	RemoveXattr(ctx ReceiveContext, path string, name string) error
	Truncate(ctx ReceiveContext, path string, size uint64) error
	Chmod(ctx ReceiveContext, path string, mode fs.FileMode) error
	Chown(pctx ReceiveContext, ath string, uid uint64, gid uint64) error
	Utimes(ctx ReceiveContext, path string, atime, mtime, ctime time.Time) error
	UpdateExtent(ctx ReceiveContext, path string, fileOffset uint64, tmpSize uint64) error
	EnableVerity(ctx ReceiveContext, path string, algorithm uint8, blockSize uint32, salt []byte, sig []byte) error
	Fallocate(ctx ReceiveContext, path string, mode fs.FileMode, offset uint64, len uint64) error
	Fileattr(ctx ReceiveContext, path string, attr uint32) error

	FinishSubvolume(ctx ReceiveContext) error
}

type ReceiveContext interface {
	context.Context

	CurrentSubvolume() *ReceivingSubvolume
	ResolvePath(path string) string
	Log() *log.Logger
	Verbosity() int
}

type ReceivingSubvolume struct {
	// The path of the subvolume
	Path string
	// The UUID of the subvolume
	UUID uuid.UUID
	// The ctransid of the subvolume
	Ctransid uint64
}

func (r *ReceivingSubvolume) ResolvePath(path string) string {
	return filepath.Join(r.Path, path)
}

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

// Package directory implements a receiver that receives snapshots into a directory,
// typically on a non-btrfs filesystem. It tries to track progress via a file and
// resume from the last known offset. For each subvolume, a file named after the
// UUID is used to track the offset. Upon completion of a subvolume math.MaxUint64 is
// written to the file. This can be used to determine if a transfer was completed.
package directory

import (
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/google/uuid"

	"github.com/tinyzimmer/btrsync/pkg/btrfs"
	"github.com/tinyzimmer/btrsync/pkg/receive/receivers"
	"github.com/tinyzimmer/btrsync/pkg/sendstream"
)

type directoryReceiver struct {
	destPath        string
	currentOffset   uint64
	offsetDirectory string
}

func New(path string, offsetDirectory string) receivers.Receiver {
	return &directoryReceiver{path, 0, offsetDirectory}
}

func (n *directoryReceiver) resolvePath(ctx receivers.ReceiveContext, path string) string {
	return filepath.Join(n.destPath, path)
}

func (n *directoryReceiver) currentOffsetPath(ctx receivers.ReceiveContext) string {
	return filepath.Join(n.destPath, n.offsetDirectory, ctx.CurrentSubvolume().UUID.String())
}

func (n *directoryReceiver) PreOp(ctx receivers.ReceiveContext, hdr sendstream.CmdHeader, attrs sendstream.CmdAttrs) error {
	if n.currentOffset > ctx.CurrentOffset() {
		ctx.LogVerbose(4, "skipping preop for %q, already at offset %d", attrs.GetPath(), n.currentOffset)
		return receivers.ErrSkipCommand
	}
	return nil
}

func (n *directoryReceiver) PostOp(ctx receivers.ReceiveContext, hdr sendstream.CmdHeader, attrs sendstream.CmdAttrs) error {
	if ctx.CurrentOffset() < n.currentOffset {
		return nil
	}
	f, err := os.OpenFile(n.currentOffsetPath(ctx), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	ctx.LogVerbose(4, "writing offset %d to %q\n", ctx.CurrentOffset(), f.Name())
	_, err = f.Write([]byte(fmt.Sprintf("%d", ctx.CurrentOffset())))
	return err
}

func (n *directoryReceiver) Subvol(ctx receivers.ReceiveContext, path string, uuid uuid.UUID, ctransid uint64) error {
	ctx.LogVerbose(2, "creating directory at %q\n", n.destPath)
	if err := os.MkdirAll(filepath.Dir(n.currentOffsetPath(ctx)), 0755); err != nil {
		return err
	}
	f, err := os.Open(n.currentOffsetPath(ctx))
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	defer f.Close()
	var offset uint64
	if _, err := fmt.Fscanf(f, "%d", &offset); err != nil {
		return err
	}
	n.currentOffset = offset
	return nil
}

func (n *directoryReceiver) Snapshot(ctx receivers.ReceiveContext, path string, uuid uuid.UUID, ctransid uint64, cloneUUID uuid.UUID, cloneCtransid uint64) error {
	f, err := os.Open(n.currentOffsetPath(ctx))
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	defer f.Close()
	var offset uint64
	if _, err := fmt.Fscanf(f, "%d", &offset); err != nil {
		return err
	}
	n.currentOffset = offset
	return nil
}

func (n *directoryReceiver) Mkfile(ctx receivers.ReceiveContext, path string, ino uint64) error {
	path = n.resolvePath(ctx, path)
	ctx.LogVerbose(2, "creating file at %q\n", path)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	return f.Close()
}

func (n *directoryReceiver) Mkdir(ctx receivers.ReceiveContext, path string, ino uint64) error {
	path = n.resolvePath(ctx, path)
	ctx.LogVerbose(2, "creating directory at %q\n", path)
	return os.MkdirAll(path, 0755)
}

func (n *directoryReceiver) Mknod(ctx receivers.ReceiveContext, path string, ino uint64, mode uint32, rdev uint64) error {
	path = n.resolvePath(ctx, path)
	ctx.LogVerbose(3, "creating device %q with mode %d and rdev %d\n", path, mode, rdev)
	return syscall.Mknod(path, mode, int(rdev))
}

func (n *directoryReceiver) Mkfifo(ctx receivers.ReceiveContext, path string, ino uint64) error {
	path = n.resolvePath(ctx, path)
	ctx.LogVerbose(3, "creating fifo %q with mode 0600\n", path)
	return syscall.Mkfifo(path, 0600)
}

func (n *directoryReceiver) Mksock(ctx receivers.ReceiveContext, path string, ino uint64) error {
	path = n.resolvePath(ctx, path)
	ctx.LogVerbose(3, "creating unix domain socket at %q\n", path)
	sock, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		return err
	}
	defer syscall.Close(sock)
	return syscall.Bind(sock, &syscall.SockaddrUnix{Name: path})
}

func (n *directoryReceiver) Symlink(ctx receivers.ReceiveContext, path string, ino uint64, linkTo string) error {
	path = n.resolvePath(ctx, path)
	linkTo = n.resolvePath(ctx, linkTo)
	ctx.LogVerbose(3, "creating symlink %q -> %q\n", path, linkTo)
	return os.Symlink(linkTo, path)
}

func (n *directoryReceiver) Rename(ctx receivers.ReceiveContext, oldPath string, newPath string) error {
	oldPath = n.resolvePath(ctx, oldPath)
	newPath = n.resolvePath(ctx, newPath)
	ctx.LogVerbose(3, "rename %q to %q\n", oldPath, newPath)
	return os.Rename(oldPath, newPath)
}

func (n *directoryReceiver) Link(ctx receivers.ReceiveContext, path string, linkTo string) error {
	path = n.resolvePath(ctx, path)
	linkTo = n.resolvePath(ctx, linkTo)
	ctx.LogVerbose(3, "link %q -> %q\n", path, linkTo)
	return os.Link(linkTo, path)
}

func (n *directoryReceiver) Unlink(ctx receivers.ReceiveContext, path string) error {
	path = n.resolvePath(ctx, path)
	ctx.LogVerbose(3, "unlinking %q\n", path)
	return os.Remove(path)
}

func (n *directoryReceiver) Rmdir(ctx receivers.ReceiveContext, path string) error {
	path = n.resolvePath(ctx, path)
	ctx.LogVerbose(3, "removing directory %q\n", path)
	return os.RemoveAll(path)
}

func (n *directoryReceiver) Write(ctx receivers.ReceiveContext, path string, offset uint64, data []byte) error {
	path = n.resolvePath(ctx, path)
	ctx.LogVerbose(3, "write %d bytes to %q at offset %d\n", len(data), path, offset)
	f, err := os.OpenFile(path, os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteAt(data, int64(offset)); err != nil {
		return err
	}
	return nil
}

func (n *directoryReceiver) EncodedWrite(ctx receivers.ReceiveContext, path string, op *btrfs.EncodedWriteOp) error {
	data, err := op.Decompress()
	if err != nil {
		return err
	}
	return n.Write(ctx, path, op.Offset, data)
}

func (n *directoryReceiver) Clone(ctx receivers.ReceiveContext, path string, offset uint64, len uint64, cloneUUID uuid.UUID, cloneCtransid uint64, clonePath string, cloneOffset uint64) error {
	return receivers.ErrNotSupported
}

func (n *directoryReceiver) SetXattr(ctx receivers.ReceiveContext, path string, name string, data []byte) error {
	path = n.resolvePath(ctx, path)
	ctx.LogVerbose(3, "setting xattr %q on %q\n", name, path)
	return syscall.Setxattr(path, name, data, 0)
}

func (n *directoryReceiver) RemoveXattr(ctx receivers.ReceiveContext, path string, name string) error {
	path = n.resolvePath(ctx, path)
	ctx.LogVerbose(3, "removing xattr %q on %q\n", name, path)
	return syscall.Removexattr(path, name)
}

func (n *directoryReceiver) Truncate(ctx receivers.ReceiveContext, path string, size uint64) error {
	path = n.resolvePath(ctx, path)
	ctx.LogVerbose(3, "truncating %q to %d bytes\n", path, size)
	return os.Truncate(path, int64(size))
}

func (n *directoryReceiver) Chmod(ctx receivers.ReceiveContext, path string, mode uint64) error {
	path = n.resolvePath(ctx, path)
	ctx.LogVerbose(3, "chmod %q to %o\n", path, mode)
	return os.Chmod(path, fs.FileMode(mode))
}

func (n *directoryReceiver) Chown(ctx receivers.ReceiveContext, path string, uid uint64, gid uint64) error {
	path = n.resolvePath(ctx, path)
	ctx.LogVerbose(3, "chown %q to %d:%d\n", path, uid, gid)
	return os.Chown(path, int(uid), int(gid))
}

func (n *directoryReceiver) Utimes(ctx receivers.ReceiveContext, path string, atime, mtime, ctime time.Time) error {
	path = n.resolvePath(ctx, path)
	ctx.LogVerbose(3, "utimes %q to %v:%v", path, atime, mtime)
	return os.Chtimes(path, atime, mtime)
}

func (n *directoryReceiver) UpdateExtent(ctx receivers.ReceiveContext, path string, fileOffset uint64, tmpSize uint64) error {
	/*
	 * Sent with BTRFS_SEND_FLAG_NO_FILE_DATA, nothing to do.
	 */
	ctx.LogVerbose(3, "update extent %q at offset %d with %d bytes\n", path, fileOffset, tmpSize)
	return nil
}

func (n *directoryReceiver) EnableVerity(ctx receivers.ReceiveContext, path string, algorithm uint8, blockSize uint32, salt []byte, sig []byte) error {
	return receivers.ErrNotSupported
}

func (n *directoryReceiver) Fallocate(ctx receivers.ReceiveContext, path string, mode uint32, offset uint64, len uint64) error {
	path = n.resolvePath(ctx, path)
	ctx.LogVerbose(3, "fallocate %q to %d bytes at offset %d\n", path, len, offset)
	f, err := os.OpenFile(path, os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	return syscall.Fallocate(int(f.Fd()), mode, int64(offset), int64(len))
}

func (n *directoryReceiver) Fileattr(ctx receivers.ReceiveContext, path string, attr uint32) error {
	// From source it looks like this just makes sure it can open the file for writing
	path = n.resolvePath(ctx, path)
	ctx.LogVerbose(3, "fileattr %q to %d\n", path, attr)
	f, err := os.OpenFile(path, os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	return f.Close()
}

func (n *directoryReceiver) FinishSubvolume(ctx receivers.ReceiveContext) error {
	f, err := os.OpenFile(n.currentOffsetPath(ctx), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	ctx.LogVerbose(4, "writing max uint64 to %q\n", f.Name())
	_, err = f.Write([]byte(fmt.Sprintf("%d", uint64(math.MaxUint64-1))))
	return err
}

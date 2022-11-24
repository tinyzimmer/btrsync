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

package local

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/google/uuid"

	"github.com/tinyzimmer/btrsync/pkg/btrfs"
	"github.com/tinyzimmer/btrsync/pkg/receive/receivers"
)

type localReceiver struct {
	destPath string
}

func New(destPath string) receivers.Receiver {
	return &localReceiver{destPath: destPath}
}

func (n *localReceiver) resolvePath(ctx receivers.ReceiveContext, path string) string {
	return filepath.Join(n.destPath, ctx.CurrentSubvolume().ResolvePath(path))
}

func (n *localReceiver) Subvol(ctx receivers.ReceiveContext, path string, uuid uuid.UUID, ctransid uint64) error {
	fullpath := filepath.Join(n.destPath, path)
	if ctx.Verbosity() >= 2 {
		ctx.Log().Printf("creating subvolume %q at %q\n", path, fullpath)
	}
	if err := os.MkdirAll(n.destPath, 0755); err != nil {
		return err
	}
	return btrfs.CreateSubvolume(fullpath)
}

func (n *localReceiver) Snapshot(ctx receivers.ReceiveContext, path string, uuid uuid.UUID, ctransid uint64, cloneUUID uuid.UUID, cloneCtransid uint64) error {
	if ctx.Verbosity() >= 3 {
		ctx.Log().Printf("searching for parent subvolume of snapshot %q\n", path)
	}
	root, err := btrfs.FindRootMount(n.destPath)
	if err != nil {
		return fmt.Errorf("failed to find root mount for %s: %w", n.destPath, err)
	}
	rbtree, err := btrfs.BuildRBTree(root)
	if err != nil {
		return fmt.Errorf("failed to build rbtree for %s: %w", root, err)
	}
	var parent *btrfs.RootInfo
	rbtree.PostOrderIterate(func(node *btrfs.RootInfo, lastErr error) error {
		if node.Deleted {
			return nil
		}
		if ctx.Verbosity() >= 3 {
			ctx.Log().Printf("checking if %s (%d) matches with subvolume %s (%d)\n", cloneUUID, ctransid, node.ReceivedUUID, node.Item.Stransid)
		}
		if node.ReceivedUUID == cloneUUID && node.Item.Stransid == cloneCtransid {
			if ctx.Verbosity() >= 3 {
				ctx.Log().Printf("found parent subvolume %s (%s) for snapshot %s\n", node.FullPath, node.ReceivedUUID, path)
			}
			parent = node
			return btrfs.ErrStopTreeIteration
		}
		return nil
	})
	if parent == nil {
		return fmt.Errorf("could not find parent subvolume for snapshot %q", path)
	}
	dest := filepath.Join(n.destPath, path)
	if ctx.Verbosity() >= 2 {
		ctx.Log().Printf("creating snapshot of %q at %q\n", parent.FullPath, dest)
	}
	if err := btrfs.CreateSnapshot(parent.FullPath, btrfs.WithSnapshotPath(dest)); err != nil {
		return err
	}
	return btrfs.SyncFilesystem(dest)
}

func (n *localReceiver) Mkfile(ctx receivers.ReceiveContext, path string, ino uint64) error {
	path = n.resolvePath(ctx, path)
	if ctx.Verbosity() >= 3 {
		ctx.Log().Printf("creating file %q with mode 0600\n", path)
	}
	f, err := os.OpenFile(path, os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	return f.Close()
}

func (n *localReceiver) Mkdir(ctx receivers.ReceiveContext, path string, ino uint64) error {
	path = n.resolvePath(ctx, path)
	if ctx.Verbosity() >= 3 {
		ctx.Log().Printf("making new directory at %q with mode 0755\n", path)
	}
	return os.Mkdir(path, 0755)
}

func (n *localReceiver) Mknod(ctx receivers.ReceiveContext, path string, ino uint64, mode fs.FileMode, rdev uint64) error {
	path = n.resolvePath(ctx, path)
	if ctx.Verbosity() >= 3 {
		ctx.Log().Printf("creating device %q with mode %d and rdev %d\n", path, mode, rdev)
	}
	return syscall.Mknod(path, uint32(mode), int(rdev))
}

func (n *localReceiver) Mkfifo(ctx receivers.ReceiveContext, path string, ino uint64) error {
	path = n.resolvePath(ctx, path)
	if ctx.Verbosity() >= 3 {
		ctx.Log().Printf("creating fifo %q with mode 0600\n", path)
	}
	return syscall.Mkfifo(path, 0600)
}

func (n *localReceiver) Mksock(ctx receivers.ReceiveContext, path string, ino uint64) error {
	path = n.resolvePath(ctx, path)
	if ctx.Verbosity() >= 3 {
		ctx.Log().Printf("creating socket %q with mode 0600\n", path)
	}
	return syscall.Mknod(path, 0600|syscall.S_IFSOCK, 0)
}

func (n *localReceiver) Symlink(ctx receivers.ReceiveContext, path string, ino uint64, linkTo string) error {
	path = n.resolvePath(ctx, path)
	linkTo = n.resolvePath(ctx, linkTo)
	if ctx.Verbosity() >= 3 {
		ctx.Log().Printf("creating symlink %q -> %q\n", path, linkTo)
	}
	return os.Symlink(linkTo, path)
}

func (n *localReceiver) Rename(ctx receivers.ReceiveContext, oldPath string, newPath string) error {
	oldPath = n.resolvePath(ctx, oldPath)
	newPath = n.resolvePath(ctx, newPath)
	if ctx.Verbosity() >= 3 {
		ctx.Log().Printf("rename %q to %q\n", oldPath, newPath)
	}
	return os.Rename(oldPath, newPath)
}

func (n *localReceiver) Link(ctx receivers.ReceiveContext, path string, linkTo string) error {
	path = n.resolvePath(ctx, path)
	linkTo = n.resolvePath(ctx, linkTo)
	if ctx.Verbosity() >= 3 {
		ctx.Log().Printf("link %q -> %q\n", path, linkTo)
	}
	return os.Link(linkTo, path)
}

func (n *localReceiver) Unlink(ctx receivers.ReceiveContext, path string) error {
	path = n.resolvePath(ctx, path)
	if ctx.Verbosity() >= 3 {
		ctx.Log().Printf("unlinking %q\n", path)
	}
	return os.Remove(path)
}

func (n *localReceiver) Rmdir(ctx receivers.ReceiveContext, path string) error {
	path = n.resolvePath(ctx, path)
	if ctx.Verbosity() >= 3 {
		ctx.Log().Printf("removing directory %q\n", path)
	}
	return os.RemoveAll(path)
}

func (n *localReceiver) Write(ctx receivers.ReceiveContext, path string, offset uint64, data []byte) error {
	path = n.resolvePath(ctx, path)
	if ctx.Verbosity() >= 3 {
		ctx.Log().Printf("write %d bytes to %q at offset %d\n", len(data), path, offset)
	}
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

func (n *localReceiver) EncodedWrite(ctx receivers.ReceiveContext, path string, op *btrfs.EncodedWriteOp, forceDecompress bool) error {
	fullpath := n.resolvePath(ctx, path)
	if ctx.Verbosity() >= 3 {
		ctx.Log().Printf("encoded write to %q at offset %d\n", fullpath, op.Offset)
	}
	if !forceDecompress {
		if ctx.Verbosity() >= 3 {
			ctx.Log().Println("  using encoded write ioctl")
		}
		return btrfs.EncodedWrite(fullpath, op)
	}
	if ctx.Verbosity() >= 3 {
		ctx.Log().Println("  force decompressing encoded data")
	}
	data, err := op.Decompress()
	if err != nil {
		return err
	}
	return n.Write(ctx, path, op.Offset, data)
}

func (n *localReceiver) Clone(ctx receivers.ReceiveContext, path string, offset uint64, len uint64, cloneUUID uuid.UUID, cloneCtransid uint64, clonePath string, cloneOffset uint64) error {
	var subvolPath string
	if cloneUUID == ctx.CurrentSubvolume().UUID {
		subvolPath = filepath.Join(n.destPath, ctx.CurrentSubvolume().Path)
	} else {
		parent, err := btrfs.SubvolumeSearch(btrfs.SearchWithRootMount(n.destPath), btrfs.SearchWithReceivedUUID(cloneUUID))
		if err != nil {
			return fmt.Errorf("cannot find parent subvolume for clone: %w", err)
		}
		subvolPath = filepath.Join(n.destPath, parent.Path)
	}
	clonePath = filepath.Join(subvolPath, clonePath)
	destPath := n.resolvePath(ctx, path)
	if ctx.Verbosity() >= 3 {
		ctx.Log().Printf("clone %d bytes from %q at offset %d to %q at offset %d\n", len, clonePath, cloneOffset, destPath, offset)
	}
	return btrfs.Clone(clonePath, destPath, cloneOffset, offset, len)
}

func (n *localReceiver) SetXattr(ctx receivers.ReceiveContext, path string, name string, data []byte) error {
	path = n.resolvePath(ctx, path)
	if ctx.Verbosity() >= 3 {
		ctx.Log().Printf("setting xattr %q on %q\n", name, path)
	}
	return syscall.Setxattr(path, name, data, 0)
}

func (n *localReceiver) RemoveXattr(ctx receivers.ReceiveContext, path string, name string) error {
	path = n.resolvePath(ctx, path)
	if ctx.Verbosity() >= 3 {
		ctx.Log().Printf("removing xattr %q on %q\n", name, path)
	}
	return syscall.Removexattr(path, name)
}

func (n *localReceiver) Truncate(ctx receivers.ReceiveContext, path string, size uint64) error {
	path = n.resolvePath(ctx, path)
	if ctx.Verbosity() >= 3 {
		ctx.Log().Printf("truncating %q to %d bytes\n", path, size)
	}
	return os.Truncate(path, int64(size))
}

func (n *localReceiver) Chmod(ctx receivers.ReceiveContext, path string, mode fs.FileMode) error {
	path = n.resolvePath(ctx, path)
	if ctx.Verbosity() >= 3 {
		ctx.Log().Printf("chmod %q to %o\n", path, mode)
	}
	return os.Chmod(path, mode)
}

func (n *localReceiver) Chown(ctx receivers.ReceiveContext, path string, uid uint64, gid uint64) error {
	path = n.resolvePath(ctx, path)
	if ctx.Verbosity() >= 3 {
		ctx.Log().Printf("chown %q to %d:%d\n", path, uid, gid)
	}
	return os.Chown(path, int(uid), int(gid))
}

func (n *localReceiver) Utimes(ctx receivers.ReceiveContext, path string, atime, mtime, ctime time.Time) error {
	path = n.resolvePath(ctx, path)
	if ctx.Verbosity() >= 3 {
		ctx.Log().Printf("utimes %q to %v:%v", path, atime, mtime)
	}
	return os.Chtimes(path, atime, mtime)
}

func (n *localReceiver) UpdateExtent(ctx receivers.ReceiveContext, path string, fileOffset uint64, tmpSize uint64) error {
	/*
	 * Sent with BTRFS_SEND_FLAG_NO_FILE_DATA, nothing to do.
	 */
	if ctx.Verbosity() >= 3 {
		ctx.Log().Printf("update extent %q at offset %d with %d bytes\n", path, fileOffset, tmpSize)
	}
	return nil
}

func (n *localReceiver) EnableVerity(ctx receivers.ReceiveContext, path string, algorithm uint8, blockSize uint32, salt []byte, sig []byte) error {
	path = n.resolvePath(ctx, path)
	if ctx.Verbosity() >= 3 {
		ctx.Log().Printf("enable verity %q with algorithm %d, block size %d, salt %v, signature %v\n", path, algorithm, blockSize, salt, sig)
	}
	return btrfs.EnableVerity(path, uint32(algorithm), blockSize, salt, sig)
}

func (n *localReceiver) Fallocate(ctx receivers.ReceiveContext, path string, mode fs.FileMode, offset uint64, len uint64) error {
	path = n.resolvePath(ctx, path)
	if ctx.Verbosity() >= 3 {
		ctx.Log().Printf("fallocate %q to %d bytes at offset %d\n", path, len, offset)
	}
	f, err := os.OpenFile(path, os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	return syscall.Fallocate(int(f.Fd()), uint32(mode), int64(offset), int64(len))
}

func (n *localReceiver) Fileattr(ctx receivers.ReceiveContext, path string, attr uint32) error {
	// From source it looks like this just makes sure it can open the file for writing
	path = n.resolvePath(ctx, path)
	if ctx.Verbosity() >= 3 {
		ctx.Log().Printf("fileattr %q to %d\n", path, attr)
	}
	f, err := os.OpenFile(path, os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	return f.Close()
}

func (n *localReceiver) FinishSubvolume(ctx receivers.ReceiveContext) error {
	curVol := ctx.CurrentSubvolume()
	path := filepath.Join(n.destPath, curVol.Path)
	isReadOnly, err := btrfs.IsSubvolumeReadOnly(path)
	if err != nil {
		return err
	}
	if isReadOnly {
		if ctx.Verbosity() >= 3 {
			ctx.Log().Printf("setting subvolume %q read-write temporarily to finish operations\n", path)
		}
		err = btrfs.SetSubvolumeReadOnly(path, false)
		if err != nil {
			return err
		}
	}
	if ctx.Verbosity() >= 2 {
		ctx.Log().Printf("finish subvolume %s with uuid=%s ctransid=%d\n", curVol.Path, curVol.UUID, curVol.Ctransid)
	}
	if err := btrfs.SetReceivedSubvolume(path, curVol.UUID, curVol.Ctransid); err != nil {
		return err
	}
	if err := btrfs.SetSubvolumeReadOnly(path, true); err != nil {
		return err
	}
	return btrfs.SyncFilesystem(path)
}

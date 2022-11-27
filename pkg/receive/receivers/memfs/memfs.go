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

// Package memfs implements a simple in-memory filesystem from a btrfs send stream.
// The filesystem can optionally be exported for FUSE mounting.
package memfs

import (
	"context"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/blang/vfs/memfs"
	"github.com/google/uuid"
	fusefs "github.com/hanwen/go-fuse/v2/fs"

	"github.com/tinyzimmer/btrsync/pkg/btrfs"
	"github.com/tinyzimmer/btrsync/pkg/receive/receivers"
)

type fmode int

const fmodeDir fmode = 1
const fmodeFile fmode = 2

type MemFSReceiver struct {
	fusefs.Inode
	Fs       *memfs.MemFS
	curFiles map[string]fmode
	symlinks map[string]string
}

func New() *MemFSReceiver {
	return &MemFSReceiver{
		Fs:       memfs.Create(),
		curFiles: map[string]fmode{},
		symlinks: map[string]string{},
	}
}

func (n *MemFSReceiver) Subvol(ctx receivers.ReceiveContext, path string, uuid uuid.UUID, ctransid uint64) error {
	this := "."
	for _, dir := range filepath.SplitList(path) {
		this = filepath.Join(this, dir)
		if err := n.Fs.Mkdir(this, fs.ModeDir); err != nil && !os.IsExist(err) {
			return err
		}
	}
	n.curFiles[path] = fmodeDir
	return nil
}

func (n *MemFSReceiver) Snapshot(ctx receivers.ReceiveContext, path string, uuid uuid.UUID, ctransid uint64, cloneUUID uuid.UUID, cloneCtransid uint64) error {
	// could technically be supported with symlinks
	return receivers.ErrNotSupported
}

func (n *MemFSReceiver) Mkfile(ctx receivers.ReceiveContext, path string, ino uint64) error {
	n.curFiles[ctx.ResolvePath(path)] = fmodeFile
	f, err := n.Fs.OpenFile(ctx.ResolvePath(path), os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	return f.Close()
}

func (n *MemFSReceiver) Mkdir(ctx receivers.ReceiveContext, path string, ino uint64) error {
	n.curFiles[ctx.ResolvePath(path)] = fmodeDir
	return n.Fs.Mkdir(ctx.ResolvePath(path), 0755)
}

func (n *MemFSReceiver) Mknod(ctx receivers.ReceiveContext, path string, ino uint64, mode uint32, rdev uint64) error {
	return receivers.ErrNotSupported
}

func (n *MemFSReceiver) Mkfifo(ctx receivers.ReceiveContext, path string, ino uint64) error {
	return receivers.ErrNotSupported
}

func (n *MemFSReceiver) Mksock(ctx receivers.ReceiveContext, path string, ino uint64) error {
	return receivers.ErrNotSupported
}

func (n *MemFSReceiver) Symlink(ctx receivers.ReceiveContext, path string, ino uint64, linkTo string) error {
	n.symlinks[ctx.ResolvePath(path)] = ctx.ResolvePath(linkTo)
	return nil
}

func (n *MemFSReceiver) Rename(ctx receivers.ReceiveContext, oldPath string, newPath string) error {
	cur := n.curFiles[ctx.ResolvePath(oldPath)]
	n.curFiles[ctx.ResolvePath(newPath)] = cur
	delete(n.curFiles, ctx.ResolvePath(oldPath))
	return n.Fs.Rename(ctx.ResolvePath(oldPath), ctx.ResolvePath(newPath))
}

func (n *MemFSReceiver) Link(ctx receivers.ReceiveContext, path string, linkTo string) error {
	return receivers.ErrNotSupported
}

func (n *MemFSReceiver) Unlink(ctx receivers.ReceiveContext, path string) error {
	delete(n.curFiles, ctx.ResolvePath(path))
	delete(n.symlinks, ctx.ResolvePath(path))
	return n.Fs.Remove(ctx.ResolvePath(path))
}

func (n *MemFSReceiver) Rmdir(ctx receivers.ReceiveContext, path string) error {
	delete(n.curFiles, ctx.ResolvePath(path))
	return n.Fs.Remove(ctx.ResolvePath(path))
}

func (n *MemFSReceiver) Write(ctx receivers.ReceiveContext, path string, offset uint64, data []byte) error {
	f, err := n.Fs.OpenFile(ctx.ResolvePath(path), os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Seek(int64(offset), io.SeekStart); err != nil {
		return err
	}
	_, err = f.Write(data)
	n.curFiles[ctx.ResolvePath(path)] = fmodeFile
	return err
}

func (n *MemFSReceiver) EncodedWrite(ctx receivers.ReceiveContext, path string, op *btrfs.EncodedWriteOp) error {
	data, err := op.Decompress()
	if err != nil {
		return err
	}
	return n.Write(ctx, path, op.Offset, data)
}

func (n *MemFSReceiver) Clone(ctx receivers.ReceiveContext, path string, offset uint64, len uint64, cloneUUID uuid.UUID, cloneCtransid uint64, clonePath string, cloneOffset uint64) error {
	return receivers.ErrNotSupported
}

func (n *MemFSReceiver) SetXattr(ctx receivers.ReceiveContext, path string, name string, data []byte) error {
	return nil
}

func (n *MemFSReceiver) RemoveXattr(ctx receivers.ReceiveContext, path string, name string) error {
	return nil
}

func (n *MemFSReceiver) Truncate(ctx receivers.ReceiveContext, path string, size uint64) error {
	f, err := n.Fs.OpenFile(ctx.ResolvePath(path), os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Truncate(int64(size))
}

func (n *MemFSReceiver) Chmod(ctx receivers.ReceiveContext, path string, mode uint64) error {
	return nil
}

func (n *MemFSReceiver) Chown(ctx receivers.ReceiveContext, path string, uid uint64, gid uint64) error {
	return nil
}

func (n *MemFSReceiver) Utimes(ctx receivers.ReceiveContext, path string, atime, mtime, ctime time.Time) error {
	return nil
}

func (n *MemFSReceiver) UpdateExtent(ctx receivers.ReceiveContext, path string, fileOffset uint64, tmpSize uint64) error {
	return nil
}

func (n *MemFSReceiver) EnableVerity(ctx receivers.ReceiveContext, path string, algorithm uint8, blockSize uint32, salt []byte, sig []byte) error {
	return receivers.ErrNotSupported
}

func (n *MemFSReceiver) Fallocate(ctx receivers.ReceiveContext, path string, mode uint32, offset uint64, len uint64) error {
	n.curFiles[ctx.ResolvePath(path)] = fmodeFile
	return nil
}

func (n *MemFSReceiver) Fileattr(ctx receivers.ReceiveContext, path string, attr uint32) error {
	return nil
}

func (n *MemFSReceiver) FinishSubvolume(rctx receivers.ReceiveContext) error {
	return nil
}

func (n *MemFSReceiver) OnAdd(ctx context.Context) {
	for name, mode := range n.curFiles {
		dir, base := filepath.Split(name)
		p := n.mkdirAll(ctx, dir)
		switch mode {
		case fmodeDir:
			log.Println("creating directory reference", name)
			ch := p.NewPersistentInode(ctx, &fusefs.Inode{},
				fusefs.StableAttr{Mode: syscall.S_IFDIR})
			p.AddChild(base, ch, true)
		case fmodeFile:
			log.Println("creating file reference", name)
			f, err := n.Fs.OpenFile(name, os.O_RDONLY, 0600)
			if err != nil {
				continue
			}
			data, err := io.ReadAll(f)
			if err != nil {
				continue
			}
			embedder := &fusefs.MemRegularFile{Data: data}
			ch := p.NewPersistentInode(ctx, embedder, fusefs.StableAttr{})
			p.AddChild(base, ch, true)
		}
	}
	for name, target := range n.symlinks {
		dir, base := filepath.Split(name)
		p := n.mkdirAll(ctx, dir)
		embedder := &fusefs.MemSymlink{
			Inode: n.findInode(ctx, target),
		}
		ch := p.NewPersistentInode(ctx, embedder, fusefs.StableAttr{})
		p.AddChild(base, ch, true)
	}
	n.curFiles = make(map[string]fmode)
	n.symlinks = make(map[string]string)
}

func (n *MemFSReceiver) findInode(ctx context.Context, path string) fusefs.Inode {
	p := &n.Inode
	for _, name := range filepath.SplitList(path) {
		if name == "" {
			continue
		}
		p = p.GetChild(name)
		if p == nil {
			return fusefs.Inode{}
		}
	}
	return *p
}

func (n *MemFSReceiver) mkdirAll(ctx context.Context, path string) *fusefs.Inode {
	p := &n.Inode
	for _, component := range strings.Split(path, "/") {
		if len(component) == 0 {
			continue
		}
		ch := p.GetChild(component)
		if ch == nil {
			// Create a directory
			ch = p.NewPersistentInode(ctx, &fusefs.Inode{},
				fusefs.StableAttr{Mode: syscall.S_IFDIR})
			// Add it
			p.AddChild(component, ch, true)
		}
		p = ch
	}
	return p
}

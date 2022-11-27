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

// Package sshdir provides a receiver that can receive snapshots over ssh. It behaves like the
// directory receiver, but writes the data over an SSH connection. The receiver depends on linux
// coreutils being available on the destination. For receiving to a btrfs volume over SSH, it is
// better to pipe the data into btrfs receive or btrsync, as it will be much faster than writing
// the data to disk and then receiving it. Btrsync does this automatically if the mirror is set
// to an ssh:// path with the subvolume format.
package sshdir

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"

	"github.com/tinyzimmer/btrsync/pkg/btrfs"
	"github.com/tinyzimmer/btrsync/pkg/receive/receivers"
	"github.com/tinyzimmer/btrsync/pkg/sendstream"
)

type sshReceiver struct {
	sshClient       *ssh.Client
	destPath        string
	currentOffset   uint64
	offsetDirectory string
}

func New(client *ssh.Client, path string, offsetDirectory string) receivers.Receiver {
	return &sshReceiver{
		sshClient:       client,
		destPath:        path,
		currentOffset:   0,
		offsetDirectory: offsetDirectory,
	}
}

func (n *sshReceiver) resolvePath(ctx receivers.ReceiveContext, path string) string {
	return filepath.Join(n.destPath, path)
}

func (n *sshReceiver) currentOffsetPath(ctx receivers.ReceiveContext) string {
	return filepath.Join(n.destPath, n.offsetDirectory, ctx.CurrentSubvolume().UUID.String())
}

func (n *sshReceiver) runCommand(ctx receivers.ReceiveContext, cmd string) ([]byte, error) {
	ctx.LogVerbose(4, "running command %q", cmd)
	sess, err := n.sshClient.NewSession()
	if err != nil {
		return nil, err
	}
	defer sess.Close()
	out, err := sess.CombinedOutput(cmd)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", err, out)
	}
	return out, nil
}

func (n *sshReceiver) runCommandWithStdin(ctx receivers.ReceiveContext, cmd string, stdin []byte) error {
	ctx.LogVerbose(4, "running command %q with %d bytes of stdin", cmd, len(stdin))
	sess, err := n.sshClient.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()
	sess.Stdin = bytes.NewReader(stdin)
	out, err := sess.CombinedOutput(cmd)
	if err != nil {
		return fmt.Errorf("%s: %s", err, out)
	}
	return nil
}

func (n *sshReceiver) readCurrentOffset(ctx receivers.ReceiveContext) error {
	path := n.currentOffsetPath(ctx)
	out, err := n.runCommand(ctx, fmt.Sprintf("cat %q", path))
	if err != nil {
		if strings.Contains(err.Error(), "No such file or directory") {
			return nil
		}
		return err
	}
	var offset uint64
	if _, err := fmt.Fscanf(bytes.NewReader(out), "%d", &offset); err != nil {
		return err
	}
	n.currentOffset = offset
	return nil
}

func (n *sshReceiver) writeCurrentOffset(ctx receivers.ReceiveContext) error {
	path := n.currentOffsetPath(ctx)
	ctx.LogVerbose(4, "writing offset %d to %q\n", ctx.CurrentOffset(), path)
	_, err := n.runCommand(ctx, fmt.Sprintf("echo %d > %q", ctx.CurrentOffset(), path))
	return err
}

func (n *sshReceiver) writeFinishedOffset(ctx receivers.ReceiveContext) error {
	path := n.currentOffsetPath(ctx)
	offset := uint64(math.MaxUint64 - 1)
	ctx.LogVerbose(4, "writing max offset to %q\n", path)
	_, err := n.runCommand(ctx, fmt.Sprintf("echo %d > %q", offset, path))
	return err
}

func (n *sshReceiver) PreOp(ctx receivers.ReceiveContext, hdr sendstream.CmdHeader, attrs sendstream.CmdAttrs) error {
	if n.currentOffset > ctx.CurrentOffset() {
		ctx.LogVerbose(4, "skipping preop for %q, already at offset %d", attrs.GetPath(), n.currentOffset)
		return receivers.ErrSkipCommand
	}
	return nil
}

func (n *sshReceiver) PostOp(ctx receivers.ReceiveContext, hdr sendstream.CmdHeader, attrs sendstream.CmdAttrs) error {
	if ctx.CurrentOffset() < n.currentOffset {
		return nil
	}
	return n.writeCurrentOffset(ctx)
}

func (n *sshReceiver) Subvol(ctx receivers.ReceiveContext, path string, uuid uuid.UUID, ctransid uint64) error {
	ctx.LogVerbose(3, "creating directory at %q\n", n.destPath)
	offsetPath := n.currentOffsetPath(ctx)
	if _, err := n.runCommand(ctx, fmt.Sprintf("mkdir -p %q", filepath.Dir(offsetPath))); err != nil {
		return err
	}
	return n.readCurrentOffset(ctx)
}

func (n *sshReceiver) Snapshot(ctx receivers.ReceiveContext, path string, uuid uuid.UUID, ctransid uint64, cloneUUID uuid.UUID, cloneCtransid uint64) error {
	return n.readCurrentOffset(ctx)
}

func (n *sshReceiver) Mkfile(ctx receivers.ReceiveContext, path string, ino uint64) error {
	path = n.resolvePath(ctx, path)
	ctx.LogVerbose(3, "creating file at %q\n", path)
	_, err := n.runCommand(ctx, fmt.Sprintf("touch %q", path))
	return err
}

func (n *sshReceiver) Mkdir(ctx receivers.ReceiveContext, path string, ino uint64) error {
	path = n.resolvePath(ctx, path)
	ctx.LogVerbose(3, "creating directory at %q\n", path)
	_, err := n.runCommand(ctx, fmt.Sprintf("mkdir %q", path))
	return err
}

func (n *sshReceiver) Mknod(ctx receivers.ReceiveContext, path string, ino uint64, mode uint32, rdev uint64) error {
	path = n.resolvePath(ctx, path)
	ctx.LogVerbose(3, "creating node at %q\n", path)
	_, err := n.runCommand(ctx, fmt.Sprintf("mknod %q %o %d", path, mode, rdev))
	return err
}

func (n *sshReceiver) Mkfifo(ctx receivers.ReceiveContext, path string, ino uint64) error {
	path = n.resolvePath(ctx, path)
	ctx.LogVerbose(3, "creating fifo at %q\n", path)
	_, err := n.runCommand(ctx, fmt.Sprintf("mkfifo %q", path))
	return err
}

func (n *sshReceiver) Mksock(ctx receivers.ReceiveContext, path string, ino uint64) error {
	// There is probably a better way to do this, but I don't know what it is - use python
	path = n.resolvePath(ctx, path)
	ctx.LogVerbose(3, "creating unix domain socket at %q\n", path)
	_, err := n.runCommand(ctx, fmt.Sprintf("python -c \"import socket; socket.socket(socket.AF_UNIX, socket.SOCK_STREAM).bind('%s')\"", path))
	return err
}

func (n *sshReceiver) Symlink(ctx receivers.ReceiveContext, path string, ino uint64, linkTo string) error {
	path = n.resolvePath(ctx, path)
	ctx.LogVerbose(3, "creating symlink at %q -> %q\n", path, linkTo)
	_, err := n.runCommand(ctx, fmt.Sprintf("ln -s %q %q", linkTo, path))
	return err
}

func (n *sshReceiver) Rename(ctx receivers.ReceiveContext, oldPath string, newPath string) error {
	oldPath = n.resolvePath(ctx, oldPath)
	newPath = n.resolvePath(ctx, newPath)
	ctx.LogVerbose(3, "renaming %q -> %q\n", oldPath, newPath)
	_, err := n.runCommand(ctx, fmt.Sprintf("mv %q %q", oldPath, newPath))
	return err
}

func (n *sshReceiver) Link(ctx receivers.ReceiveContext, path string, linkTo string) error {
	path = n.resolvePath(ctx, path)
	ctx.LogVerbose(3, "creating hard link at %q -> %q\n", path, linkTo)
	_, err := n.runCommand(ctx, fmt.Sprintf("ln %q %q", linkTo, path))
	return err
}

func (n *sshReceiver) Unlink(ctx receivers.ReceiveContext, path string) error {
	path = n.resolvePath(ctx, path)
	ctx.LogVerbose(3, "removing %q\n", path)
	_, err := n.runCommand(ctx, fmt.Sprintf("rm -rf %q", path))
	return err
}

func (n *sshReceiver) Rmdir(ctx receivers.ReceiveContext, path string) error {
	path = n.resolvePath(ctx, path)
	ctx.LogVerbose(3, "removing %q\n", path)
	_, err := n.runCommand(ctx, fmt.Sprintf("rm -rf %q", path))
	return err
}

func (n *sshReceiver) Write(ctx receivers.ReceiveContext, path string, offset uint64, data []byte) error {
	// This is slow as shit - we should probably use sftp or buffer the writes.
	// But more realistically, people should be receiving to subvolumes over ssh.
	// This is, however, suitable for small files.
	path = n.resolvePath(ctx, path)
	ctx.LogVerbose(3, "writing %d bytes to %q at offset %d\n", len(data), path, offset)
	err := n.runCommandWithStdin(ctx, fmt.Sprintf("dd of=%q bs=1 seek=%d count=%d conv=notrunc", path, offset, len(data)), data)
	return err
}

func (n *sshReceiver) EncodedWrite(ctx receivers.ReceiveContext, path string, op *btrfs.EncodedWriteOp) error {
	data, err := op.Decompress()
	if err != nil {
		return err
	}
	return n.Write(ctx, path, op.Offset, data)
}

func (n *sshReceiver) Clone(ctx receivers.ReceiveContext, path string, offset uint64, len uint64, cloneUUID uuid.UUID, cloneCtransid uint64, clonePath string, cloneOffset uint64) error {
	return receivers.ErrNotSupported
}

func (n *sshReceiver) SetXattr(ctx receivers.ReceiveContext, path string, name string, data []byte) error {
	path = n.resolvePath(ctx, path)
	ctx.LogVerbose(3, "setting xattr %q on %q\n", name, path)
	_, err := n.runCommand(ctx, fmt.Sprintf("setfattr -n %s -v 0S%s %q",
		name, base64.StdEncoding.EncodeToString(data), path))
	return err
}

func (n *sshReceiver) RemoveXattr(ctx receivers.ReceiveContext, path string, name string) error {
	path = n.resolvePath(ctx, path)
	ctx.LogVerbose(3, "removing xattr %q on %q\n", name, path)
	_, err := n.runCommand(ctx, fmt.Sprintf("setfattr -x %s %q", name, path))
	return err
}

func (n *sshReceiver) Truncate(ctx receivers.ReceiveContext, path string, size uint64) error {
	path = n.resolvePath(ctx, path)
	ctx.LogVerbose(3, "truncating %q to %d bytes\n", path, size)
	_, err := n.runCommand(ctx, fmt.Sprintf("truncate -s %d %q", size, path))
	return err
}

func (n *sshReceiver) Chmod(ctx receivers.ReceiveContext, path string, mode uint64) error {
	path = n.resolvePath(ctx, path)
	ctx.LogVerbose(3, "chmod %q to %d\n", path, mode)
	_, err := n.runCommand(ctx, fmt.Sprintf("chmod %o %q", mode, path))
	return err
}

func (n *sshReceiver) Chown(ctx receivers.ReceiveContext, path string, uid uint64, gid uint64) error {
	path = n.resolvePath(ctx, path)
	ctx.LogVerbose(3, "chown %q to %d:%d\n", path, uid, gid)
	_, err := n.runCommand(ctx, fmt.Sprintf("chown %d:%d %q", uid, gid, path))
	return err
}

func (n *sshReceiver) Utimes(ctx receivers.ReceiveContext, path string, atime, mtime, ctime time.Time) error {
	path = n.resolvePath(ctx, path)
	ctx.LogVerbose(3, "utimes %q to %s:%s:%s\n", path, atime, mtime, ctime)
	_, err := n.runCommand(ctx, fmt.Sprintf("touch -a -m -t %s %q", mtime.Format("200601021504.05"), path))
	return err
}

func (n *sshReceiver) UpdateExtent(ctx receivers.ReceiveContext, path string, fileOffset uint64, tmpSize uint64) error {
	/*
	 * Sent with BTRFS_SEND_FLAG_NO_FILE_DATA, nothing to do.
	 */
	ctx.LogVerbose(3, "update extent %q at offset %d with %d bytes\n", path, fileOffset, tmpSize)
	return nil
}

func (n *sshReceiver) EnableVerity(ctx receivers.ReceiveContext, path string, algorithm uint8, blockSize uint32, salt []byte, sig []byte) error {
	return receivers.ErrNotSupported
}

func (n *sshReceiver) Fallocate(ctx receivers.ReceiveContext, path string, mode uint32, offset uint64, len uint64) error {
	path = n.resolvePath(ctx, path)
	ctx.LogVerbose(3, "fallocate %q to %d bytes at offset %d\n", path, len, offset)
	_, err := n.runCommand(ctx, fmt.Sprintf("fallocate -l %d -o %d %q", len, offset, path))
	if err != nil {
		return err
	}
	return n.Chmod(ctx, path, uint64(mode))
}

func (n *sshReceiver) Fileattr(ctx receivers.ReceiveContext, path string, attr uint32) error {
	// Ignore
	return nil
}

func (n *sshReceiver) FinishSubvolume(ctx receivers.ReceiveContext) error {
	return n.writeFinishedOffset(ctx)
}

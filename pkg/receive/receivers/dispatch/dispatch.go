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

// Package dispatch provides a receiver that dispatches to multiple receivers.
package dispatch

import (
	"time"

	"github.com/google/uuid"

	"github.com/tinyzimmer/btrsync/pkg/btrfs"
	"github.com/tinyzimmer/btrsync/pkg/receive/receivers"
)

type dispatchReceiver struct {
	receivers []receivers.Receiver
}

func New(receivers ...receivers.Receiver) receivers.Receiver {
	return &dispatchReceiver{receivers: receivers}
}

func (n *dispatchReceiver) Subvol(ctx receivers.ReceiveContext, path string, uuid uuid.UUID, ctransid uint64) error {
	for _, r := range n.receivers {
		if err := r.Subvol(ctx, path, uuid, ctransid); err != nil {
			return err
		}
	}
	return nil
}

func (n *dispatchReceiver) Snapshot(ctx receivers.ReceiveContext, path string, uuid uuid.UUID, ctransid uint64, cloneUUID uuid.UUID, cloneCtransid uint64) error {
	for _, r := range n.receivers {
		if err := r.Snapshot(ctx, path, uuid, ctransid, cloneUUID, cloneCtransid); err != nil {
			return err
		}
	}
	return nil
}

func (n *dispatchReceiver) Mkfile(ctx receivers.ReceiveContext, path string, ino uint64) error {
	for _, r := range n.receivers {
		if err := r.Mkfile(ctx, path, ino); err != nil {
			return err
		}
	}
	return nil
}

func (n *dispatchReceiver) Mkdir(ctx receivers.ReceiveContext, path string, ino uint64) error {
	for _, r := range n.receivers {
		if err := r.Mkdir(ctx, path, ino); err != nil {
			return err
		}
	}
	return nil
}

func (n *dispatchReceiver) Mknod(ctx receivers.ReceiveContext, path string, ino uint64, mode uint32, rdev uint64) error {
	for _, r := range n.receivers {
		if err := r.Mknod(ctx, path, ino, mode, rdev); err != nil {
			return err
		}
	}
	return nil
}

func (n *dispatchReceiver) Mkfifo(ctx receivers.ReceiveContext, path string, ino uint64) error {
	for _, r := range n.receivers {
		if err := r.Mkfifo(ctx, path, ino); err != nil {
			return err
		}
	}
	return nil
}

func (n *dispatchReceiver) Mksock(ctx receivers.ReceiveContext, path string, ino uint64) error {
	for _, r := range n.receivers {
		if err := r.Mksock(ctx, path, ino); err != nil {
			return err
		}
	}
	return nil
}

func (n *dispatchReceiver) Symlink(ctx receivers.ReceiveContext, path string, ino uint64, linkTo string) error {
	for _, r := range n.receivers {
		if err := r.Symlink(ctx, path, ino, linkTo); err != nil {
			return err
		}
	}
	return nil
}

func (n *dispatchReceiver) Rename(ctx receivers.ReceiveContext, oldPath string, newPath string) error {
	for _, r := range n.receivers {
		if err := r.Rename(ctx, oldPath, newPath); err != nil {
			return err
		}
	}
	return nil
}

func (n *dispatchReceiver) Link(ctx receivers.ReceiveContext, path string, linkTo string) error {
	for _, r := range n.receivers {
		if err := r.Link(ctx, path, linkTo); err != nil {
			return err
		}
	}
	return nil
}

func (n *dispatchReceiver) Unlink(ctx receivers.ReceiveContext, path string) error {
	for _, r := range n.receivers {
		if err := r.Unlink(ctx, path); err != nil {
			return err
		}
	}
	return nil
}

func (n *dispatchReceiver) Rmdir(ctx receivers.ReceiveContext, path string) error {
	for _, r := range n.receivers {
		if err := r.Rmdir(ctx, path); err != nil {
			return err
		}
	}
	return nil
}

func (n *dispatchReceiver) Write(ctx receivers.ReceiveContext, path string, offset uint64, data []byte) error {
	for _, r := range n.receivers {
		if err := r.Write(ctx, path, offset, data); err != nil {
			return err
		}
	}
	return nil
}

func (n *dispatchReceiver) EncodedWrite(ctx receivers.ReceiveContext, path string, op *btrfs.EncodedWriteOp) error {
	for _, r := range n.receivers {
		o := op
		if err := r.EncodedWrite(ctx, path, o); err != nil {
			return err
		}
	}
	return nil
}

func (n *dispatchReceiver) Clone(ctx receivers.ReceiveContext, path string, offset uint64, len uint64, cloneUUID uuid.UUID, cloneCtransid uint64, clonePath string, cloneOffset uint64) error {
	for _, r := range n.receivers {
		if err := r.Clone(ctx, path, offset, len, cloneUUID, cloneCtransid, clonePath, cloneOffset); err != nil {
			return err
		}
	}
	return nil
}

func (n *dispatchReceiver) SetXattr(ctx receivers.ReceiveContext, path string, name string, data []byte) error {
	for _, r := range n.receivers {
		if err := r.SetXattr(ctx, path, name, data); err != nil {
			return err
		}
	}
	return nil
}

func (n *dispatchReceiver) RemoveXattr(ctx receivers.ReceiveContext, path string, name string) error {
	for _, r := range n.receivers {
		if err := r.RemoveXattr(ctx, path, name); err != nil {
			return err
		}
	}
	return nil
}

func (n *dispatchReceiver) Truncate(ctx receivers.ReceiveContext, path string, size uint64) error {
	for _, r := range n.receivers {
		if err := r.Truncate(ctx, path, size); err != nil {
			return err
		}
	}
	return nil
}

func (n *dispatchReceiver) Chmod(ctx receivers.ReceiveContext, path string, mode uint64) error {
	for _, r := range n.receivers {
		if err := r.Chmod(ctx, path, mode); err != nil {
			return err
		}
	}
	return nil
}

func (n *dispatchReceiver) Chown(ctx receivers.ReceiveContext, path string, uid uint64, gid uint64) error {
	for _, r := range n.receivers {
		if err := r.Chown(ctx, path, uid, gid); err != nil {
			return err
		}
	}
	return nil
}

func (n *dispatchReceiver) Utimes(ctx receivers.ReceiveContext, path string, atime, mtime, ctime time.Time) error {
	for _, r := range n.receivers {
		if err := r.Utimes(ctx, path, atime, mtime, ctime); err != nil {
			return err
		}
	}
	return nil
}

func (n *dispatchReceiver) UpdateExtent(ctx receivers.ReceiveContext, path string, fileOffset uint64, tmpSize uint64) error {
	for _, r := range n.receivers {
		if err := r.UpdateExtent(ctx, path, fileOffset, tmpSize); err != nil {
			return err
		}
	}
	return nil
}

func (n *dispatchReceiver) EnableVerity(ctx receivers.ReceiveContext, path string, algorithm uint8, blockSize uint32, salt []byte, sig []byte) error {
	for _, r := range n.receivers {
		if err := r.EnableVerity(ctx, path, algorithm, blockSize, salt, sig); err != nil {
			return err
		}
	}
	return nil
}

func (n *dispatchReceiver) Fallocate(ctx receivers.ReceiveContext, path string, mode uint32, offset uint64, len uint64) error {
	for _, r := range n.receivers {
		if err := r.Fallocate(ctx, path, mode, offset, len); err != nil {
			return err
		}
	}
	return nil
}

func (n *dispatchReceiver) Fileattr(ctx receivers.ReceiveContext, path string, attr uint32) error {
	for _, r := range n.receivers {
		if err := r.Fileattr(ctx, path, attr); err != nil {
			return err
		}
	}
	return nil
}

func (n *dispatchReceiver) FinishSubvolume(ctx receivers.ReceiveContext) error {
	for _, r := range n.receivers {
		if err := r.FinishSubvolume(ctx); err != nil {
			return err
		}
	}
	return nil
}

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

package receive

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io/fs"

	"github.com/google/uuid"

	"github.com/tinyzimmer/btrsync/pkg/btrfs"
	"github.com/tinyzimmer/btrsync/pkg/receive/receivers"
	"github.com/tinyzimmer/btrsync/pkg/sendstream"
)

type processFunc func(*receiveCtx, sendstream.CmdAttrs) error

var processFuncs = map[sendstream.SendCommand]processFunc{
	sendstream.BTRFS_SEND_C_SUBVOL:        processSubvol,
	sendstream.BTRFS_SEND_C_SNAPSHOT:      processSnapshot,
	sendstream.BTRFS_SEND_C_MKFILE:        processMkfile,
	sendstream.BTRFS_SEND_C_MKDIR:         processMkdir,
	sendstream.BTRFS_SEND_C_MKNOD:         processMknod,
	sendstream.BTRFS_SEND_C_MKFIFO:        processMkfifo,
	sendstream.BTRFS_SEND_C_MKSOCK:        processMksock,
	sendstream.BTRFS_SEND_C_SYMLINK:       processSymlink,
	sendstream.BTRFS_SEND_C_RENAME:        processRename,
	sendstream.BTRFS_SEND_C_LINK:          processLink,
	sendstream.BTRFS_SEND_C_UNLINK:        processUnlink,
	sendstream.BTRFS_SEND_C_RMDIR:         processRmdir,
	sendstream.BTRFS_SEND_C_WRITE:         processWrite,
	sendstream.BTRFS_SEND_C_ENCODED_WRITE: processEncodedWrite,
	sendstream.BTRFS_SEND_C_CLONE:         processClone,
	sendstream.BTRFS_SEND_C_SET_XATTR:     processSetXattr,
	sendstream.BTRFS_SEND_C_REMOVE_XATTR:  processRemoveXattr,
	sendstream.BTRFS_SEND_C_TRUNCATE:      processTruncate,
	sendstream.BTRFS_SEND_C_CHMOD:         processChmod,
	sendstream.BTRFS_SEND_C_CHOWN:         processChown,
	sendstream.BTRFS_SEND_C_UTIMES:        processUtimes,
	sendstream.BTRFS_SEND_C_UPDATE_EXTENT: processUpdateExtent,
	sendstream.BTRFS_SEND_C_ENABLE_VERITY: processEnableVerity,
	sendstream.BTRFS_SEND_C_FALLOCATE:     processFallocate,
	sendstream.BTRFS_SEND_C_FILEATTR:      processFileattr,
}

func processSubvol(ctx *receiveCtx, attrs sendstream.CmdAttrs) error {
	if err := ensureAttrs(attrs, []sendstream.SendAttribute{
		sendstream.BTRFS_SEND_A_PATH, sendstream.BTRFS_SEND_A_UUID, sendstream.BTRFS_SEND_A_CTRANSID,
	}); err != nil {
		return fmt.Errorf("processSubvol: %w", err)
	}
	if ctx.currentSubvolInfo != nil {
		if err := ctx.receiver.FinishSubvolume(ctx); err != nil {
			return fmt.Errorf("processSubvol: error finishing in-process subvolume: %w", err)
		}
		ctx.currentSubvolInfo = nil
	}
	path := string(attrs[sendstream.BTRFS_SEND_A_PATH])
	ctransid := binary.LittleEndian.Uint64(attrs[sendstream.BTRFS_SEND_A_CTRANSID])
	uuid, err := uuid.FromBytes(attrs[sendstream.BTRFS_SEND_A_UUID])
	if err != nil {
		return fmt.Errorf("processSubvol: error parsing uuid %w", err)
	}
	ctx.LogVerbose(0, "At subvol %q\n", path)
	ctx.LogVerbose(2, "receiving subvol %q uuid=%s, stransid=%d\n", path, uuid, ctransid)
	if err := ctx.receiver.Subvol(ctx, path, uuid, ctransid); err != nil {
		return err
	}
	ctx.currentSubvolInfo = &receivers.ReceivingSubvolume{
		Path: path, UUID: uuid, Ctransid: ctransid,
	}
	return nil
}

func processSnapshot(ctx *receiveCtx, attrs sendstream.CmdAttrs) error {
	if err := ensureAttrs(attrs, []sendstream.SendAttribute{
		sendstream.BTRFS_SEND_A_PATH, sendstream.BTRFS_SEND_A_UUID, sendstream.BTRFS_SEND_A_CTRANSID,
		sendstream.BTRFS_SEND_A_CLONE_UUID, sendstream.BTRFS_SEND_A_CLONE_CTRANSID,
	}); err != nil {
		return fmt.Errorf("processSnapshot: %w", err)
	}
	if ctx.currentSubvolInfo != nil {
		if err := ctx.receiver.FinishSubvolume(ctx); err != nil {
			return fmt.Errorf("processSnapshot: error finishing in-process subvolume: %w", err)
		}
		ctx.currentSubvolInfo = nil
	}
	path := string(attrs[sendstream.BTRFS_SEND_A_PATH])
	snapuuid, err := uuid.FromBytes(attrs[sendstream.BTRFS_SEND_A_UUID])
	if err != nil {
		return fmt.Errorf("processSnapshot: error parsing uuid %w", err)
	}
	ctransid := binary.LittleEndian.Uint64(attrs[sendstream.BTRFS_SEND_A_CTRANSID])
	cloneUUID, err := uuid.FromBytes(attrs[sendstream.BTRFS_SEND_A_CLONE_UUID])
	if err != nil {
		return fmt.Errorf("processSnapshot: error parsing clone uuid %w", err)
	}
	cloneCtransid := binary.LittleEndian.Uint64(attrs[sendstream.BTRFS_SEND_A_CLONE_CTRANSID])
	ctx.LogVerbose(0, "At snapshot %q\n", path)
	ctx.LogVerbose(2, "receiving snapshot %q uuid=%s, stransid=%d, clone_uuid=%s, clone_stransid=%d\n",
		path, snapuuid, ctransid, cloneUUID, cloneCtransid)
	if err := ctx.receiver.Snapshot(ctx, path, snapuuid, ctransid, cloneUUID, cloneCtransid); err != nil {
		return err
	}
	ctx.currentSubvolInfo = &receivers.ReceivingSubvolume{
		Path: path, UUID: snapuuid, Ctransid: ctransid,
	}
	return nil
}

func processMkfile(ctx *receiveCtx, attrs sendstream.CmdAttrs) error {
	if err := ensureAttrs(attrs, []sendstream.SendAttribute{
		sendstream.BTRFS_SEND_A_PATH, sendstream.BTRFS_SEND_A_INO,
	}); err != nil {
		return fmt.Errorf("processMkfile: %w", err)
	}
	path := string(attrs[sendstream.BTRFS_SEND_A_PATH])
	ino := binary.LittleEndian.Uint64(attrs[sendstream.BTRFS_SEND_A_INO])
	if ctx.verbosity >= 2 {
		ctx.LogVerbose(2, "receiving mkfile %q ino=%d\n", path, ino)
	}
	return ctx.receiver.Mkfile(ctx, path, ino)
}

func processMkdir(ctx *receiveCtx, attrs sendstream.CmdAttrs) error {
	if err := ensureAttrs(attrs, []sendstream.SendAttribute{
		sendstream.BTRFS_SEND_A_PATH, sendstream.BTRFS_SEND_A_INO,
	}); err != nil {
		return fmt.Errorf("processMkfile: %w", err)
	}
	path := string(attrs[sendstream.BTRFS_SEND_A_PATH])
	ino := binary.LittleEndian.Uint64(attrs[sendstream.BTRFS_SEND_A_INO])
	ctx.LogVerbose(2, "receiving mkfile %q ino=%d\n", path, ino)
	return ctx.receiver.Mkdir(ctx, path, ino)
}

func processMknod(ctx *receiveCtx, attrs sendstream.CmdAttrs) error {
	if err := ensureAttrs(attrs, []sendstream.SendAttribute{
		sendstream.BTRFS_SEND_A_PATH, sendstream.BTRFS_SEND_A_INO, sendstream.BTRFS_SEND_A_MODE, sendstream.BTRFS_SEND_A_RDEV,
	}); err != nil {
		return fmt.Errorf("processMknod: %w", err)
	}
	path := string(attrs[sendstream.BTRFS_SEND_A_PATH])
	ino := binary.LittleEndian.Uint64(attrs[sendstream.BTRFS_SEND_A_INO])
	mode := binary.LittleEndian.Uint32(attrs[sendstream.BTRFS_SEND_A_MODE])
	rdev := binary.LittleEndian.Uint64(attrs[sendstream.BTRFS_SEND_A_RDEV])
	ctx.LogVerbose(2, "receiving mknod %q ino=%d mode=%o rdev=%d\n", path, ino, mode, rdev)
	return ctx.receiver.Mknod(ctx, path, ino, fs.FileMode(mode), rdev)
}

func processMkfifo(ctx *receiveCtx, attrs sendstream.CmdAttrs) error {
	if err := ensureAttrs(attrs, []sendstream.SendAttribute{
		sendstream.BTRFS_SEND_A_PATH, sendstream.BTRFS_SEND_A_INO,
	}); err != nil {
		return fmt.Errorf("processMkfifo: %w", err)
	}
	path := string(attrs[sendstream.BTRFS_SEND_A_PATH])
	ino := binary.LittleEndian.Uint64(attrs[sendstream.BTRFS_SEND_A_INO])
	ctx.LogVerbose(2, "receiving mkfifo %q ino=%d\n", path, ino)
	return ctx.receiver.Mkfifo(ctx, path, ino)
}

func processMksock(ctx *receiveCtx, attrs sendstream.CmdAttrs) error {
	if err := ensureAttrs(attrs, []sendstream.SendAttribute{
		sendstream.BTRFS_SEND_A_PATH, sendstream.BTRFS_SEND_A_INO,
	}); err != nil {
		return fmt.Errorf("processMksock: %w", err)
	}
	path := string(attrs[sendstream.BTRFS_SEND_A_PATH])
	ino := binary.LittleEndian.Uint64(attrs[sendstream.BTRFS_SEND_A_INO])
	ctx.LogVerbose(2, "receiving mksock %q ino=%d\n", path, ino)
	return ctx.receiver.Mksock(ctx, path, ino)
}

func processSymlink(ctx *receiveCtx, attrs sendstream.CmdAttrs) error {
	if err := ensureAttrs(attrs, []sendstream.SendAttribute{
		sendstream.BTRFS_SEND_A_PATH, sendstream.BTRFS_SEND_A_INO, sendstream.BTRFS_SEND_A_PATH_LINK,
	}); err != nil {
		return fmt.Errorf("processSymlink: %w", err)
	}
	path := string(attrs[sendstream.BTRFS_SEND_A_PATH])
	ino := binary.LittleEndian.Uint64(attrs[sendstream.BTRFS_SEND_A_INO])
	pathLink := string(attrs[sendstream.BTRFS_SEND_A_PATH_LINK])
	ctx.LogVerbose(2, "receiving symlink %q ino=%d -> %q\n", path, ino, pathLink)
	return ctx.receiver.Symlink(ctx, path, ino, pathLink)
}

func processRename(ctx *receiveCtx, attrs sendstream.CmdAttrs) error {
	if err := ensureAttrs(attrs, []sendstream.SendAttribute{
		sendstream.BTRFS_SEND_A_PATH, sendstream.BTRFS_SEND_A_PATH_TO,
	}); err != nil {
		return fmt.Errorf("processRename: %w", err)
	}
	path := string(attrs[sendstream.BTRFS_SEND_A_PATH])
	pathTo := string(attrs[sendstream.BTRFS_SEND_A_PATH_TO])
	ctx.LogVerbose(2, "receiving rename %q -> %q", path, pathTo)
	return ctx.receiver.Rename(ctx, path, pathTo)
}

func processLink(ctx *receiveCtx, attrs sendstream.CmdAttrs) error {
	if err := ensureAttrs(attrs, []sendstream.SendAttribute{
		sendstream.BTRFS_SEND_A_PATH, sendstream.BTRFS_SEND_A_PATH_LINK,
	}); err != nil {
		return fmt.Errorf("processLink: %w", err)
	}
	path := string(attrs[sendstream.BTRFS_SEND_A_PATH])
	pathLink := string(attrs[sendstream.BTRFS_SEND_A_PATH_LINK])
	ctx.LogVerbose(2, "receiving link %q -> %q", path, pathLink)
	return ctx.receiver.Link(ctx, path, pathLink)
}

func processUnlink(ctx *receiveCtx, attrs sendstream.CmdAttrs) error {
	if err := ensureAttrs(attrs, []sendstream.SendAttribute{
		sendstream.BTRFS_SEND_A_PATH,
	}); err != nil {
		return fmt.Errorf("processUnlink: %w", err)
	}
	path := string(attrs[sendstream.BTRFS_SEND_A_PATH])
	ctx.LogVerbose(2, "receiving unlink %q", path)
	return ctx.receiver.Unlink(ctx, path)
}

func processRmdir(ctx *receiveCtx, attrs sendstream.CmdAttrs) error {
	if err := ensureAttrs(attrs, []sendstream.SendAttribute{
		sendstream.BTRFS_SEND_A_PATH,
	}); err != nil {
		return fmt.Errorf("processRmdir: %w", err)
	}
	path := string(attrs[sendstream.BTRFS_SEND_A_PATH])
	ctx.LogVerbose(2, "receiving rmdir %q", path)
	return ctx.receiver.Rmdir(ctx, path)
}

func processWrite(ctx *receiveCtx, attrs sendstream.CmdAttrs) error {
	if err := ensureAttrs(attrs, []sendstream.SendAttribute{
		sendstream.BTRFS_SEND_A_PATH, sendstream.BTRFS_SEND_A_FILE_OFFSET, sendstream.BTRFS_SEND_A_DATA,
	}); err != nil {
		return fmt.Errorf("processWrite: %w", err)
	}
	path := string(attrs[sendstream.BTRFS_SEND_A_PATH])
	offset := binary.LittleEndian.Uint64(attrs[sendstream.BTRFS_SEND_A_FILE_OFFSET])
	data := attrs[sendstream.BTRFS_SEND_A_DATA]
	ctx.LogVerbose(2, "receiving write %q offset=%d len=%d", path, offset, len(data))
	return ctx.receiver.Write(ctx, path, offset, data)
}

func processEncodedWrite(ctx *receiveCtx, attrs sendstream.CmdAttrs) error {
	if err := ensureAttrs(attrs, []sendstream.SendAttribute{
		sendstream.BTRFS_SEND_A_PATH, sendstream.BTRFS_SEND_A_FILE_OFFSET, sendstream.BTRFS_SEND_A_UNENCODED_FILE_LEN, sendstream.BTRFS_SEND_A_UNENCODED_LEN,
		sendstream.BTRFS_SEND_A_UNENCODED_OFFSET, sendstream.BTRFS_SEND_A_DATA, // Optional: sendstream.BTRFS_SEND_A_COMPRESSION, sendstream.BTRFS_SEND_A_ENCRYPTION
	}); err != nil {
		return fmt.Errorf("processEncodedWrite: %w", err)
	}
	path := string(attrs[sendstream.BTRFS_SEND_A_PATH])
	var op btrfs.EncodedWriteOp
	op.Offset = binary.LittleEndian.Uint64(attrs[sendstream.BTRFS_SEND_A_FILE_OFFSET])
	op.Data = attrs[sendstream.BTRFS_SEND_A_DATA]
	op.UnencodedFileLength = binary.LittleEndian.Uint64(attrs[sendstream.BTRFS_SEND_A_UNENCODED_FILE_LEN])
	op.UnencodedLength = binary.LittleEndian.Uint64(attrs[sendstream.BTRFS_SEND_A_UNENCODED_LEN])
	op.UnencodedOffset = binary.LittleEndian.Uint64(attrs[sendstream.BTRFS_SEND_A_UNENCODED_OFFSET])
	if len(attrs[sendstream.BTRFS_SEND_A_COMPRESSION]) > 0 {
		op.Compression = btrfs.CompressionType(binary.LittleEndian.Uint32(attrs[sendstream.BTRFS_SEND_A_COMPRESSION]))
	}
	if len(attrs[sendstream.BTRFS_SEND_A_ENCRYPTION]) > 0 {
		op.Encryption = binary.LittleEndian.Uint32(attrs[sendstream.BTRFS_SEND_A_ENCRYPTION])
	}
	ctx.LogVerbose(2, "receiving encoded write %q offset=%d len=%d", path, op.Offset, len(op.Data))
	if ctx.forceDecompress {
		ctx.LogVerbose(1, "forcing decompression of encoded write")
		data, err := op.Decompress()
		if err != nil {
			return fmt.Errorf("processEncodedWrite: failed to decompress data: %w", err)
		}
		return ctx.receiver.Write(ctx, path, op.Offset, data)
	}
	err := ctx.receiver.EncodedWrite(ctx, path, &op)
	if err != nil && errors.Is(err, receivers.ErrNotSupported) {
		ctx.LogVerbose(1, "receiver does not support encoded writes, forcing decompression")
		data, err := op.Decompress()
		if err != nil {
			return fmt.Errorf("processEncodedWrite: failed to decompress data: %w", err)
		}
		return ctx.receiver.Write(ctx, path, op.Offset, data)
	}
	return err
}

func processClone(ctx *receiveCtx, attrs sendstream.CmdAttrs) error {
	if err := ensureAttrs(attrs, []sendstream.SendAttribute{
		sendstream.BTRFS_SEND_A_PATH, sendstream.BTRFS_SEND_A_FILE_OFFSET, sendstream.BTRFS_SEND_A_CLONE_LEN, sendstream.BTRFS_SEND_A_CLONE_UUID,
		sendstream.BTRFS_SEND_A_CLONE_CTRANSID, sendstream.BTRFS_SEND_A_CLONE_PATH, sendstream.BTRFS_SEND_A_CLONE_OFFSET,
	}); err != nil {
		return fmt.Errorf("processClone: %w", err)
	}
	cloneUUID, err := uuid.FromBytes(attrs[sendstream.BTRFS_SEND_A_CLONE_UUID])
	if err != nil {
		return fmt.Errorf("processClone: error parsing cloneUUID: %w", err)
	}
	path := string(attrs[sendstream.BTRFS_SEND_A_PATH])
	offset := binary.LittleEndian.Uint64(attrs[sendstream.BTRFS_SEND_A_FILE_OFFSET])
	cloneLen := binary.LittleEndian.Uint64(attrs[sendstream.BTRFS_SEND_A_CLONE_LEN])
	cloneCTransID := binary.LittleEndian.Uint64(attrs[sendstream.BTRFS_SEND_A_CLONE_CTRANSID])
	clonePath := string(attrs[sendstream.BTRFS_SEND_A_CLONE_PATH])
	cloneOffset := binary.LittleEndian.Uint64(attrs[sendstream.BTRFS_SEND_A_CLONE_OFFSET])
	ctx.LogVerbose(2, "receiving clone %q offset=%d len=%d cloneUUID=%s cloneCTransID=%d clonePath=%q cloneOffset=%d",
		path, offset, cloneLen, cloneUUID, cloneCTransID, clonePath, cloneOffset)
	return ctx.receiver.Clone(ctx, path, offset, cloneLen, cloneUUID, cloneCTransID, clonePath, cloneOffset)
}

func processSetXattr(ctx *receiveCtx, attrs sendstream.CmdAttrs) error {
	if err := ensureAttrs(attrs, []sendstream.SendAttribute{
		sendstream.BTRFS_SEND_A_PATH, sendstream.BTRFS_SEND_A_XATTR_NAME, sendstream.BTRFS_SEND_A_XATTR_DATA,
	}); err != nil {
		return fmt.Errorf("processSetXattr: %w", err)
	}
	path := string(attrs[sendstream.BTRFS_SEND_A_PATH])
	name := string(attrs[sendstream.BTRFS_SEND_A_XATTR_NAME])
	data := attrs[sendstream.BTRFS_SEND_A_XATTR_DATA]
	ctx.LogVerbose(2, "receiving setxattr %q name=%q len=%d", path, name, len(data))
	return ctx.receiver.SetXattr(ctx, path, name, data)
}

func processRemoveXattr(ctx *receiveCtx, attrs sendstream.CmdAttrs) error {
	if err := ensureAttrs(attrs, []sendstream.SendAttribute{
		sendstream.BTRFS_SEND_A_PATH, sendstream.BTRFS_SEND_A_XATTR_NAME,
	}); err != nil {
		return fmt.Errorf("processRemoveXattr: %w", err)
	}
	path := string(attrs[sendstream.BTRFS_SEND_A_PATH])
	name := string(attrs[sendstream.BTRFS_SEND_A_XATTR_NAME])
	ctx.LogVerbose(2, "receiving removexattr %q name=%q", path, name)
	return ctx.receiver.RemoveXattr(ctx, path, name)
}

func processTruncate(ctx *receiveCtx, attrs sendstream.CmdAttrs) error {
	if err := ensureAttrs(attrs, []sendstream.SendAttribute{
		sendstream.BTRFS_SEND_A_PATH, sendstream.BTRFS_SEND_A_SIZE,
	}); err != nil {
		return fmt.Errorf("processTruncate: %w", err)
	}
	path := string(attrs[sendstream.BTRFS_SEND_A_PATH])
	size := binary.LittleEndian.Uint64(attrs[sendstream.BTRFS_SEND_A_SIZE])
	ctx.LogVerbose(2, "receiving truncate %q size=%d", path, size)
	return ctx.receiver.Truncate(ctx, path, size)
}

func processChmod(ctx *receiveCtx, attrs sendstream.CmdAttrs) error {
	if err := ensureAttrs(attrs, []sendstream.SendAttribute{
		sendstream.BTRFS_SEND_A_PATH, sendstream.BTRFS_SEND_A_MODE,
	}); err != nil {
		return fmt.Errorf("processChmod: %w", err)
	}
	path := string(attrs[sendstream.BTRFS_SEND_A_PATH])
	mode := binary.LittleEndian.Uint64(attrs[sendstream.BTRFS_SEND_A_MODE])
	ctx.LogVerbose(2, "receiving chmod %q mode=%o", path, mode)
	return ctx.receiver.Chmod(ctx, path, fs.FileMode(mode))
}

func processChown(ctx *receiveCtx, attrs sendstream.CmdAttrs) error {
	if err := ensureAttrs(attrs, []sendstream.SendAttribute{
		sendstream.BTRFS_SEND_A_PATH, sendstream.BTRFS_SEND_A_UID, sendstream.BTRFS_SEND_A_GID,
	}); err != nil {
		return fmt.Errorf("processChown: %w", err)
	}
	path := string(attrs[sendstream.BTRFS_SEND_A_PATH])
	var uid, gid uint64
	if err := binary.Read(bytes.NewReader(attrs[sendstream.BTRFS_SEND_A_UID]), binary.LittleEndian, &uid); err != nil {
		return fmt.Errorf("processChown: error parsing uid: %w", err)
	}
	if err := binary.Read(bytes.NewReader(attrs[sendstream.BTRFS_SEND_A_GID]), binary.LittleEndian, &gid); err != nil {
		return fmt.Errorf("processChown: error parsing gid: %w", err)
	}
	ctx.LogVerbose(2, "receiving chown %q uid=%d gid=%d", path, uid, gid)
	return ctx.receiver.Chown(ctx, path, uid, gid)
}

func processUtimes(ctx *receiveCtx, attrs sendstream.CmdAttrs) error {
	if err := ensureAttrs(attrs, []sendstream.SendAttribute{
		sendstream.BTRFS_SEND_A_PATH, sendstream.BTRFS_SEND_A_ATIME, sendstream.BTRFS_SEND_A_MTIME, sendstream.BTRFS_SEND_A_CTIME,
	}); err != nil {
		return fmt.Errorf("processUtimes: %w", err)
	}
	path := string(attrs[sendstream.BTRFS_SEND_A_PATH])
	var atimespec, mtimespec, ctimespec btrfs.BtrfsTimespec
	if err := binary.Read(bytes.NewReader(attrs[sendstream.BTRFS_SEND_A_ATIME]), binary.LittleEndian, &atimespec); err != nil {
		return fmt.Errorf("processUtimes: error parsing atime: %w", err)
	}
	if err := binary.Read(bytes.NewReader(attrs[sendstream.BTRFS_SEND_A_MTIME]), binary.LittleEndian, &mtimespec); err != nil {
		return fmt.Errorf("processUtimes: error parsing mtime: %w", err)
	}
	if err := binary.Read(bytes.NewReader(attrs[sendstream.BTRFS_SEND_A_CTIME]), binary.LittleEndian, &ctimespec); err != nil {
		return fmt.Errorf("processUtimes: error parsing ctime: %w", err)
	}
	atime := atimespec.Time()
	mtime := mtimespec.Time()
	ctime := ctimespec.Time()
	ctx.LogVerbose(2, "receiving utimes %q atime=%v mtime=%v ctime=%v", path, atime, mtime, ctime)
	return ctx.receiver.Utimes(ctx, path, atime, mtime, ctime)
}

func processUpdateExtent(ctx *receiveCtx, attrs sendstream.CmdAttrs) error {
	if err := ensureAttrs(attrs, []sendstream.SendAttribute{
		sendstream.BTRFS_SEND_A_PATH, sendstream.BTRFS_SEND_A_FILE_OFFSET, sendstream.BTRFS_SEND_A_SIZE,
	}); err != nil {
		return fmt.Errorf("processUpdateExtent: %w", err)
	}
	path := string(attrs[sendstream.BTRFS_SEND_A_PATH])
	offset := binary.LittleEndian.Uint64(attrs[sendstream.BTRFS_SEND_A_FILE_OFFSET])
	size := binary.LittleEndian.Uint64(attrs[sendstream.BTRFS_SEND_A_SIZE])
	ctx.LogVerbose(2, "receiving update_extent %q offset=%d size=%d", path, offset, size)
	return ctx.receiver.UpdateExtent(ctx, path, offset, size)
}

func processEnableVerity(ctx *receiveCtx, attrs sendstream.CmdAttrs) error {
	if err := ensureAttrs(attrs, []sendstream.SendAttribute{
		sendstream.BTRFS_SEND_A_PATH, sendstream.BTRFS_SEND_A_VERITY_ALGORITHM, sendstream.BTRFS_SEND_A_VERITY_BLOCK_SIZE,
		sendstream.BTRFS_SEND_A_VERITY_SALT_DATA, sendstream.BTRFS_SEND_A_VERITY_SIG_DATA,
	}); err != nil {
		return fmt.Errorf("processEnableVerity: %w", err)
	}
	path := string(attrs[sendstream.BTRFS_SEND_A_PATH])
	blockSize := binary.LittleEndian.Uint32(attrs[sendstream.BTRFS_SEND_A_VERITY_BLOCK_SIZE])
	salt := attrs[sendstream.BTRFS_SEND_A_VERITY_SALT_DATA]
	sig := attrs[sendstream.BTRFS_SEND_A_VERITY_SIG_DATA]
	var algorithm uint8
	if err := binary.Read(bytes.NewReader(attrs[sendstream.BTRFS_SEND_A_VERITY_ALGORITHM]), binary.LittleEndian, &algorithm); err != nil {
		return fmt.Errorf("processEnableVerity: error parsing algorithm: %w", err)
	}
	return ctx.receiver.EnableVerity(ctx, path, algorithm, blockSize, salt, sig)
}

func processFallocate(ctx *receiveCtx, attrs sendstream.CmdAttrs) error {
	if err := ensureAttrs(attrs, []sendstream.SendAttribute{
		sendstream.BTRFS_SEND_A_PATH, sendstream.BTRFS_SEND_A_FALLOCATE_MODE, sendstream.BTRFS_SEND_A_FILE_OFFSET, sendstream.BTRFS_SEND_A_SIZE,
	}); err != nil {
		return fmt.Errorf("processFallocate: %w", err)
	}
	path := string(attrs[sendstream.BTRFS_SEND_A_PATH])
	mode := binary.LittleEndian.Uint32(attrs[sendstream.BTRFS_SEND_A_FALLOCATE_MODE])
	offset := binary.LittleEndian.Uint64(attrs[sendstream.BTRFS_SEND_A_FILE_OFFSET])
	size := binary.LittleEndian.Uint64(attrs[sendstream.BTRFS_SEND_A_SIZE])
	ctx.LogVerbose(2, "receiving fallocate %q mode=%d offset=%d size=%d", path, mode, offset, size)
	return ctx.receiver.Fallocate(ctx, path, fs.FileMode(mode), offset, size)
}

func processFileattr(ctx *receiveCtx, attrs sendstream.CmdAttrs) error {
	if err := ensureAttrs(attrs, []sendstream.SendAttribute{
		sendstream.BTRFS_SEND_A_PATH, sendstream.BTRFS_SEND_A_FILEATTR,
	}); err != nil {
		return fmt.Errorf("processFileattr: %w", err)
	}
	path := string(attrs[sendstream.BTRFS_SEND_A_PATH])
	fileattr := binary.LittleEndian.Uint32(attrs[sendstream.BTRFS_SEND_A_FILEATTR])
	ctx.LogVerbose(2, "receiving fileattr %q fileattr=%d", path, fileattr)
	return ctx.receiver.Fileattr(ctx, path, fileattr)
}

func ensureAttrs(attrs sendstream.CmdAttrs, keys []sendstream.SendAttribute) error {
	for _, key := range keys {
		if _, ok := attrs[key]; !ok {
			return fmt.Errorf("missing attribute in send stream: %s", key)
		}
	}
	return nil
}

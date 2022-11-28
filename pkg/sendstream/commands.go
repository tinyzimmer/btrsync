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

package sendstream

import (
	"bytes"
	"encoding/binary"
	"time"

	"github.com/google/uuid"
	"github.com/tinyzimmer/btrsync/pkg/btrfs"
)

const (
	BTRFS_SEND_STREAM_MAGIC          = "btrfs-stream\x00"
	BTRFS_SEND_STREAM_VERSION uint32 = 2
)

var (
	BTRFS_SEND_STREAM_MAGIC_ENCODED [13]byte = func() [13]byte {
		var arr [13]byte
		copy(arr[:], BTRFS_SEND_STREAM_MAGIC)
		return arr
	}()
)

// Send commands
//go:generate stringer -type=SendCommand

type SendCommand uint16

const (
	BTRFS_SEND_C_UNSPEC SendCommand = 0

	/* Version 1 */
	BTRFS_SEND_C_SUBVOL   SendCommand = 1
	BTRFS_SEND_C_SNAPSHOT SendCommand = 2

	BTRFS_SEND_C_MKFILE  SendCommand = 3
	BTRFS_SEND_C_MKDIR   SendCommand = 4
	BTRFS_SEND_C_MKNOD   SendCommand = 5
	BTRFS_SEND_C_MKFIFO  SendCommand = 6
	BTRFS_SEND_C_MKSOCK  SendCommand = 7
	BTRFS_SEND_C_SYMLINK SendCommand = 8

	BTRFS_SEND_C_RENAME SendCommand = 9
	BTRFS_SEND_C_LINK   SendCommand = 10
	BTRFS_SEND_C_UNLINK SendCommand = 11
	BTRFS_SEND_C_RMDIR  SendCommand = 12

	BTRFS_SEND_C_SET_XATTR    SendCommand = 13
	BTRFS_SEND_C_REMOVE_XATTR SendCommand = 14

	BTRFS_SEND_C_WRITE SendCommand = 15
	BTRFS_SEND_C_CLONE SendCommand = 16

	BTRFS_SEND_C_TRUNCATE SendCommand = 17
	BTRFS_SEND_C_CHMOD    SendCommand = 18
	BTRFS_SEND_C_CHOWN    SendCommand = 19
	BTRFS_SEND_C_UTIMES   SendCommand = 20

	BTRFS_SEND_C_END           SendCommand = 21
	BTRFS_SEND_C_UPDATE_EXTENT SendCommand = 22
	BTRFS_SEND_C_MAX_V1        SendCommand = 22

	/* Version 2 */
	BTRFS_SEND_C_FALLOCATE     SendCommand = 23
	BTRFS_SEND_C_FILEATTR      SendCommand = 24
	BTRFS_SEND_C_ENCODED_WRITE SendCommand = 25
	BTRFS_SEND_C_MAX_V2        SendCommand = 25

	BTRFS_SEND_C_ENABLE_VERITY SendCommand = 26
	BTRFS_SEND_C_MAX_V3        SendCommand = 26
	/* End */
	BTRFS_SEND_C_MAX SendCommand = 26
)

// Send Attributes
//go:generate stringer -type=SendAttribute

type SendAttribute uint16

const (
	BTRFS_SEND_A_UNSPEC SendAttribute = 0

	/* Version 1 */
	BTRFS_SEND_A_UUID     SendAttribute = 1
	BTRFS_SEND_A_CTRANSID SendAttribute = 2

	BTRFS_SEND_A_INO   SendAttribute = 3
	BTRFS_SEND_A_SIZE  SendAttribute = 4
	BTRFS_SEND_A_MODE  SendAttribute = 5
	BTRFS_SEND_A_UID   SendAttribute = 6
	BTRFS_SEND_A_GID   SendAttribute = 7
	BTRFS_SEND_A_RDEV  SendAttribute = 8
	BTRFS_SEND_A_CTIME SendAttribute = 9
	BTRFS_SEND_A_MTIME SendAttribute = 10
	BTRFS_SEND_A_ATIME SendAttribute = 11
	BTRFS_SEND_A_OTIME SendAttribute = 12

	BTRFS_SEND_A_XATTR_NAME SendAttribute = 13
	BTRFS_SEND_A_XATTR_DATA SendAttribute = 14

	BTRFS_SEND_A_PATH      SendAttribute = 15
	BTRFS_SEND_A_PATH_TO   SendAttribute = 16
	BTRFS_SEND_A_PATH_LINK SendAttribute = 17

	BTRFS_SEND_A_FILE_OFFSET SendAttribute = 18
	/*
	 * As of send stream v2, this attribute is special: it must be the last
	 * attribute in a command, its header contains only the type, and its
	 * length is implicitly the remaining length of the command.
	 */
	BTRFS_SEND_A_DATA SendAttribute = 19

	BTRFS_SEND_A_CLONE_UUID     SendAttribute = 20
	BTRFS_SEND_A_CLONE_CTRANSID SendAttribute = 21
	BTRFS_SEND_A_CLONE_PATH     SendAttribute = 22
	BTRFS_SEND_A_CLONE_OFFSET   SendAttribute = 23
	BTRFS_SEND_A_CLONE_LEN      SendAttribute = 24

	BTRFS_SEND_A_MAX_V1 SendAttribute = 24

	/* Version 2 */
	BTRFS_SEND_A_FALLOCATE_MODE SendAttribute = 25

	/*
	 * File attributes from the FS_*_FL namespace (i_flags, xflags),
	 * translated to BTRFS_INODE_* bits (BTRFS_INODE_FLAG_MASK) and stored
	 * in btrfs_inode_item::flags (represented by btrfs_inode::flags and
	 * btrfs_inode::ro_flags).
	 */
	BTRFS_SEND_A_FILEATTR SendAttribute = 26

	BTRFS_SEND_A_UNENCODED_FILE_LEN SendAttribute = 27
	BTRFS_SEND_A_UNENCODED_LEN      SendAttribute = 28
	BTRFS_SEND_A_UNENCODED_OFFSET   SendAttribute = 29
	/*
	 * COMPRESSION and ENCRYPTION default to NONE (0) if omitted from
	 * BTRFS_SEND_C_ENCODED_WRITE.
	 */
	BTRFS_SEND_A_COMPRESSION SendAttribute = 30
	BTRFS_SEND_A_ENCRYPTION  SendAttribute = 31
	BTRFS_SEND_A_MAX_V2      SendAttribute = 31

	/* Version 3 */
	BTRFS_SEND_A_VERITY_ALGORITHM  SendAttribute = 32
	BTRFS_SEND_A_VERITY_BLOCK_SIZE SendAttribute = 33
	BTRFS_SEND_A_VERITY_SALT_DATA  SendAttribute = 34
	BTRFS_SEND_A_VERITY_SIG_DATA   SendAttribute = 35
	BTRFS_SEND_A_MAX_V3            SendAttribute = 35

	/* End */
	BTRFS_SEND_A_MAX SendAttribute = 35
)

type StreamHeader struct {
	Magic   [13]byte
	Version uint32
}

type CmdHeader struct {
	Len uint32
	Cmd SendCommand
	Crc uint32
}

func (c CmdHeader) IsZero() bool {
	return c.Len == 0 && c.Cmd == 0 && c.Crc == 0
}

type CmdAttrs map[SendAttribute][]byte

// NewCmdAttrs allocates a new CmdAttrs map.
func NewCmdAttrs() CmdAttrs {
	return make(map[SendAttribute][]byte)
}

// NewSubvolCommand creates a new subvolume command. Most streams will need to begin with either
// this or a snapshot command.
func NewSubvolCommand(path string, uu uuid.UUID, ctransid uint64) (SendCommand, CmdAttrs) {
	attrs := NewCmdAttrs()
	attrs.SetPath(path)
	attrs.SetUUID(uu)
	attrs.SetCtransid(ctransid)
	return BTRFS_SEND_C_SUBVOL, attrs
}

// NewSnapshotCommand creates a new snapshot command. Most streams will need to begin with either
// this or a subvolume command.
func NewSnapshotCommand(path string, uu uuid.UUID, ctransid uint64, cloneUU uuid.UUID, cloneCtransid uint64) (SendCommand, CmdAttrs) {
	attrs := NewCmdAttrs()
	attrs.SetPath(path)
	attrs.SetUUID(uu)
	attrs.SetCtransid(ctransid)
	attrs.SetCloneUUID(cloneUU)
	attrs.SetCloneCtransid(cloneCtransid)
	return BTRFS_SEND_C_SNAPSHOT, attrs
}

// NewMkfileCommand creates a new mkfile command.
func NewMkfileCommand(path string, ino uint64) (SendCommand, CmdAttrs) {
	attrs := NewCmdAttrs()
	attrs.SetPath(path)
	attrs.SetIno(ino)
	return BTRFS_SEND_C_MKFILE, attrs
}

// NewMkdirCommand creates a new mkdir command.
func NewMkdirCommand(path string, ino uint64) (SendCommand, CmdAttrs) {
	attrs := NewCmdAttrs()
	attrs.SetPath(path)
	attrs.SetIno(ino)
	return BTRFS_SEND_C_MKDIR, attrs
}

// NewMknodCommand creates a new mknod command.
func NewMknodCommand(path string, ino uint64, mode uint32, rdev uint64) (SendCommand, CmdAttrs) {
	attrs := NewCmdAttrs()
	attrs.SetPath(path)
	attrs.SetIno(ino)
	attrs.SetMode32(mode)
	attrs.SetRdev(rdev)
	return BTRFS_SEND_C_MKNOD, attrs
}

// NewMkfifoCommand creates a new mkfifo command.
func NewMkfifoCommand(path string, ino uint64) (SendCommand, CmdAttrs) {
	attrs := NewCmdAttrs()
	attrs.SetPath(path)
	attrs.SetIno(ino)
	return BTRFS_SEND_C_MKFIFO, attrs
}

// NewMksockCommand creates a new mksock command.
func NewMksockCommand(path string, ino uint64) (SendCommand, CmdAttrs) {
	attrs := NewCmdAttrs()
	attrs.SetPath(path)
	attrs.SetIno(ino)
	return BTRFS_SEND_C_MKSOCK, attrs
}

// NewSymlinkCommand creates a new symlink command.
func NewSymlinkCommand(path, link string, ino uint64) (SendCommand, CmdAttrs) {
	attrs := NewCmdAttrs()
	attrs.SetPath(path)
	attrs.SetIno(ino)
	attrs.SetPathLink(link)
	return BTRFS_SEND_C_SYMLINK, attrs
}

// NewRenameCommand creates a new rename command.
func NewRenameCommand(path, pathTo string) (SendCommand, CmdAttrs) {
	attrs := NewCmdAttrs()
	attrs.SetPath(path)
	attrs.SetPathTo(pathTo)
	return BTRFS_SEND_C_RENAME, attrs
}

// NewLinkCommand creates a new link command.
func NewLinkCommand(path, link string) (SendCommand, CmdAttrs) {
	attrs := NewCmdAttrs()
	attrs.SetPath(path)
	attrs.SetPathLink(link)
	return BTRFS_SEND_C_LINK, attrs
}

// NewUnlinkCommand creates a new unlink command.
func NewUnlinkCommand(path string) (SendCommand, CmdAttrs) {
	attrs := NewCmdAttrs()
	attrs.SetPath(path)
	return BTRFS_SEND_C_UNLINK, attrs
}

// NewRmdirCommand creates a new rmdir command.
func NewRmdirCommand(path string) (SendCommand, CmdAttrs) {
	attrs := NewCmdAttrs()
	attrs.SetPath(path)
	return BTRFS_SEND_C_RMDIR, attrs
}

// NewWriteCommand creates a new write command.
func NewWriteCommand(path string, offset uint64, data []byte) (SendCommand, CmdAttrs) {
	attrs := NewCmdAttrs()
	attrs.SetPath(path)
	attrs.SetFileOffset(offset)
	attrs.SetData(data)
	return BTRFS_SEND_C_WRITE, attrs
}

// NewEncodedWriteCommand creates a new encoded write command. This will only work on target
// btrfs filesystems that support compression.
func NewEncodedWriteCommand(path string, op *btrfs.EncodedWriteOp) (SendCommand, CmdAttrs) {
	attrs := NewCmdAttrs()
	attrs.SetPath(path)
	attrs.SetFileOffset(op.Offset)
	attrs.SetData(op.Data)
	attrs.SetUnencodedFileLen(op.UnencodedFileLength)
	attrs.SetUnencodedLen(op.UnencodedLength)
	attrs.SetUnencodedOffset(op.UnencodedOffset)
	attrs.SetCompressionType(op.Compression)
	attrs.SetEncryptionType(op.Encryption)
	return BTRFS_SEND_C_ENCODED_WRITE, attrs
}

// NewCloneCommand creates a new clone command.
func NewCloneCommand(
	path string, offset uint64, cloneLen uint64, cloneUUID uuid.UUID, cloneCtransid uint64, clonePath string, cloneOffset uint64,
) (SendCommand, CmdAttrs) {
	attrs := NewCmdAttrs()
	attrs.SetPath(path)
	attrs.SetFileOffset(offset)
	attrs.SetCloneLen(cloneLen)
	attrs.SetCloneUUID(cloneUUID)
	attrs.SetCloneCtransid(cloneCtransid)
	attrs.SetClonePath(clonePath)
	attrs.SetCloneOffset(cloneOffset)
	return BTRFS_SEND_C_CLONE, attrs
}

// NewSetXattrCommand creates a new set-xattr command.
func NewSetXattrCommand(path string, name string, data []byte) (SendCommand, CmdAttrs) {
	attrs := NewCmdAttrs()
	attrs.SetPath(path)
	attrs.SetXattrName(name)
	attrs.SetXattrData(data)
	return BTRFS_SEND_C_SET_XATTR, attrs
}

// NewRemoveXattrCommand creates a new remove-xattr command.
func NewRemoveXattrCommand(path string, name string) (SendCommand, CmdAttrs) {
	attrs := NewCmdAttrs()
	attrs.SetPath(path)
	attrs.SetXattrName(name)
	return BTRFS_SEND_C_REMOVE_XATTR, attrs
}

// NewTruncateCommand creates a new truncate command.
func NewTruncateCommand(path string, size uint64) (SendCommand, CmdAttrs) {
	attrs := NewCmdAttrs()
	attrs.SetPath(path)
	attrs.SetSize(size)
	return BTRFS_SEND_C_TRUNCATE, attrs
}

// NewChmodCommand creates a new chmod command.
func NewChmodCommand(path string, mode uint64) (SendCommand, CmdAttrs) {
	attrs := NewCmdAttrs()
	attrs.SetPath(path)
	attrs.SetMode64(mode)
	return BTRFS_SEND_C_CHMOD, attrs
}

// NewChownCommand creates a new chown command.
func NewChownCommand(path string, uid uint64, gid uint64) (SendCommand, CmdAttrs) {
	attrs := NewCmdAttrs()
	attrs.SetPath(path)
	attrs.SetUid(uid)
	attrs.SetGid(gid)
	return BTRFS_SEND_C_CHOWN, attrs
}

// NewUtimesCommand creates a new utimes command.
func NewUtimesCommand(path string, atime, mtime, ctime time.Time) (SendCommand, CmdAttrs) {
	attrs := NewCmdAttrs()
	attrs.SetPath(path)
	attrs.SetAtime(atime)
	attrs.SetMtime(mtime)
	attrs.SetCtime(ctime)
	return BTRFS_SEND_C_UTIMES, attrs
}

// NewUpdateExtentCommand creates a new update extent command.
func NewUpdateExtentCommand(path string, offset uint64, size uint64) (SendCommand, CmdAttrs) {
	attrs := NewCmdAttrs()
	attrs.SetPath(path)
	attrs.SetFileOffset(offset)
	attrs.SetSize(size)
	return BTRFS_SEND_C_UPDATE_EXTENT, attrs
}

// NewEnableVerityCommand creates a new enable verity command.
func NewEnableVerityCommand(path string, alg uint8, blockSize uint32, salt []byte, sig []byte) (SendCommand, CmdAttrs) {
	attrs := NewCmdAttrs()
	attrs.SetPath(path)
	attrs.SetVerityAlgorithm(alg)
	attrs.SetVerityBlockSize(blockSize)
	attrs.SetVeritySalt(salt)
	attrs.SetVeritySig(sig)
	return BTRFS_SEND_C_ENABLE_VERITY, attrs
}

// NewFallocateCommand creates a new fallocate command.
func NewFallocateCommand(path string, mode uint32, offset uint64, size uint64) (SendCommand, CmdAttrs) {
	attrs := NewCmdAttrs()
	attrs.SetPath(path)
	attrs.SetFallocateMode(mode)
	attrs.SetFileOffset(offset)
	attrs.SetSize(size)
	return BTRFS_SEND_C_FALLOCATE, attrs
}

// NewFileAttrCommand creates a new file-attr command.
func NewFileAttrCommand(path string, attr uint32) (SendCommand, CmdAttrs) {
	attrs := NewCmdAttrs()
	attrs.SetPath(path)
	attrs.SetFileAttr(attr)
	return BTRFS_SEND_C_FILEATTR, attrs
}

// NewEndCommand creates a new end command.
func NewEndCommand() (SendCommand, CmdAttrs) {
	return BTRFS_SEND_C_END, NewCmdAttrs()
}

// BinarySize returns the encoded length of the command attributes
// to be included in a command header.
func (c CmdAttrs) BinarySize() uint32 {
	var size uint32
	for k, v := range c {
		// The length of the attribute
		size += uint32(binary.Size(k))
		if k != BTRFS_SEND_A_DATA {
			// If not sending data, the length of the attribute value
			// is included in the size
			size += uint32(binary.Size(uint16(len(v))))
		}
		// The length of the data itself
		size += uint32(len(v))
	}
	return size
}

// Encode encodes the command attributes into a byte slice.
func (c CmdAttrs) Encode() ([]byte, error) {
	buf := new(bytes.Buffer)
	for k, v := range c {
		// Data is always sent last
		if k == BTRFS_SEND_A_DATA {
			continue
		}
		if err := binary.Write(buf, binary.LittleEndian, k); err != nil {
			return nil, err
		}
		if err := binary.Write(buf, binary.LittleEndian, uint16(len(v))); err != nil {
			return nil, err
		}
		if _, err := buf.Write(v); err != nil {
			return nil, err
		}
	}
	// Send data if any
	if data, ok := c[BTRFS_SEND_A_DATA]; ok {
		if err := binary.Write(buf, binary.LittleEndian, BTRFS_SEND_A_DATA); err != nil {
			return nil, err
		}
		if _, err := buf.Write(data); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func (c CmdAttrs) GetData() []byte {
	return c[BTRFS_SEND_A_DATA]
}

func (c CmdAttrs) SetData(bb []byte) {
	c[BTRFS_SEND_A_DATA] = bb
}

func (c CmdAttrs) GetFileOffset() uint64 {
	return binary.LittleEndian.Uint64(c[BTRFS_SEND_A_FILE_OFFSET])
}

func (c CmdAttrs) SetFileOffset(off uint64) {
	bb := make([]byte, 8)
	binary.LittleEndian.PutUint64(bb, off)
	c[BTRFS_SEND_A_FILE_OFFSET] = bb
}

func (c CmdAttrs) GetPath() string {
	return string(c[BTRFS_SEND_A_PATH])
}

func (c CmdAttrs) SetPath(path string) {
	c[BTRFS_SEND_A_PATH] = []byte(path)
}

func (c CmdAttrs) GetPathLink() string {
	return string(c[BTRFS_SEND_A_PATH_LINK])
}

func (c CmdAttrs) SetPathLink(path string) {
	c[BTRFS_SEND_A_PATH_LINK] = []byte(path)
}

func (c CmdAttrs) GetPathTo() string {
	return string(c[BTRFS_SEND_A_PATH_TO])
}

func (c CmdAttrs) SetPathTo(path string) {
	c[BTRFS_SEND_A_PATH_TO] = []byte(path)
}

func (c CmdAttrs) GetUUID() (uuid.UUID, error) {
	return uuid.FromBytes(c[BTRFS_SEND_A_UUID])
}

func (c CmdAttrs) SetUUID(uuid uuid.UUID) {
	c[BTRFS_SEND_A_UUID] = uuid[:]
}

func (c CmdAttrs) GetCloneUUID() (uuid.UUID, error) {
	return uuid.FromBytes(c[BTRFS_SEND_A_CLONE_UUID])
}

func (c CmdAttrs) SetCloneUUID(uuid uuid.UUID) {
	c[BTRFS_SEND_A_CLONE_UUID] = uuid[:]
}

func (c CmdAttrs) GetCtransid() uint64 {
	return binary.LittleEndian.Uint64(c[BTRFS_SEND_A_CTRANSID])
}

func (c CmdAttrs) SetCtransid(ctransid uint64) {
	bb := make([]byte, 8)
	binary.LittleEndian.PutUint64(bb, ctransid)
	c[BTRFS_SEND_A_CTRANSID] = bb
}

func (c CmdAttrs) GetCloneCtransid() uint64 {
	return binary.LittleEndian.Uint64(c[BTRFS_SEND_A_CLONE_CTRANSID])
}

func (c CmdAttrs) SetCloneCtransid(ctransid uint64) {
	bb := make([]byte, 8)
	binary.LittleEndian.PutUint64(bb, ctransid)
	c[BTRFS_SEND_A_CLONE_CTRANSID] = bb
}

func (c CmdAttrs) GetIno() uint64 {
	return binary.LittleEndian.Uint64(c[BTRFS_SEND_A_INO])
}

func (c CmdAttrs) SetIno(ino uint64) {
	bb := make([]byte, 8)
	binary.LittleEndian.PutUint64(bb, ino)
	c[BTRFS_SEND_A_INO] = bb
}

func (c CmdAttrs) GetMode32() uint32 {
	return binary.LittleEndian.Uint32(c[BTRFS_SEND_A_MODE])
}

func (c CmdAttrs) SetMode32(mode uint32) {
	bb := make([]byte, 4)
	binary.LittleEndian.PutUint32(bb, mode)
	c[BTRFS_SEND_A_MODE] = bb
}

func (c CmdAttrs) GetMode64() uint64 {
	return binary.LittleEndian.Uint64(c[BTRFS_SEND_A_MODE])
}

func (c CmdAttrs) SetMode64(mode uint64) {
	bb := make([]byte, 8)
	binary.LittleEndian.PutUint64(bb, mode)
	c[BTRFS_SEND_A_MODE] = bb
}

func (c CmdAttrs) GetRdev() uint64 {
	return binary.LittleEndian.Uint64(c[BTRFS_SEND_A_RDEV])
}

func (c CmdAttrs) SetRdev(rdev uint64) {
	bb := make([]byte, 8)
	binary.LittleEndian.PutUint64(bb, rdev)
	c[BTRFS_SEND_A_RDEV] = bb
}

func (c CmdAttrs) GetUnencodedFileLen() uint64 {
	return binary.LittleEndian.Uint64(c[BTRFS_SEND_A_UNENCODED_FILE_LEN])
}

func (c CmdAttrs) SetUnencodedFileLen(len uint64) {
	bb := make([]byte, 8)
	binary.LittleEndian.PutUint64(bb, len)
	c[BTRFS_SEND_A_UNENCODED_FILE_LEN] = bb
}

func (c CmdAttrs) GetUnencodedLen() uint64 {
	return binary.LittleEndian.Uint64(c[BTRFS_SEND_A_UNENCODED_LEN])
}

func (c CmdAttrs) SetUnencodedLen(len uint64) {
	bb := make([]byte, 8)
	binary.LittleEndian.PutUint64(bb, len)
	c[BTRFS_SEND_A_UNENCODED_LEN] = bb
}

func (c CmdAttrs) GetUnencodedOffset() uint64 {
	return binary.LittleEndian.Uint64(c[BTRFS_SEND_A_UNENCODED_OFFSET])
}

func (c CmdAttrs) SetUnencodedOffset(off uint64) {
	bb := make([]byte, 8)
	binary.LittleEndian.PutUint64(bb, off)
	c[BTRFS_SEND_A_UNENCODED_OFFSET] = bb
}

func (c CmdAttrs) GetCompressionType() btrfs.CompressionType {
	return btrfs.CompressionType(binary.LittleEndian.Uint32(c[BTRFS_SEND_A_COMPRESSION]))
}

func (c CmdAttrs) SetCompressionType(ct btrfs.CompressionType) {
	bb := make([]byte, 4)
	binary.LittleEndian.PutUint32(bb, uint32(ct))
	c[BTRFS_SEND_A_COMPRESSION] = bb
}

func (c CmdAttrs) GetEncryptionType() uint32 {
	return binary.LittleEndian.Uint32(c[BTRFS_SEND_A_ENCRYPTION])
}

func (c CmdAttrs) SetEncryptionType(et uint32) {
	bb := make([]byte, 4)
	binary.LittleEndian.PutUint32(bb, et)
	c[BTRFS_SEND_A_ENCRYPTION] = bb
}

func (c CmdAttrs) GetCloneLen() uint64 {
	return binary.LittleEndian.Uint64(c[BTRFS_SEND_A_CLONE_LEN])
}

func (c CmdAttrs) SetCloneLen(len uint64) {
	bb := make([]byte, 8)
	binary.LittleEndian.PutUint64(bb, len)
	c[BTRFS_SEND_A_CLONE_LEN] = bb
}

func (c CmdAttrs) GetCloneOffset() uint64 {
	return binary.LittleEndian.Uint64(c[BTRFS_SEND_A_CLONE_OFFSET])
}

func (c CmdAttrs) SetCloneOffset(off uint64) {
	bb := make([]byte, 8)
	binary.LittleEndian.PutUint64(bb, off)
	c[BTRFS_SEND_A_CLONE_OFFSET] = bb
}

func (c CmdAttrs) GetClonePath() string {
	return string(c[BTRFS_SEND_A_CLONE_PATH])
}

func (c CmdAttrs) SetClonePath(path string) {
	c[BTRFS_SEND_A_CLONE_PATH] = []byte(path)
}

func (c CmdAttrs) GetCloneCTransid() uint64 {
	return binary.LittleEndian.Uint64(c[BTRFS_SEND_A_CLONE_CTRANSID])
}

func (c CmdAttrs) SetCloneCTransid(ctransid uint64) {
	bb := make([]byte, 8)
	binary.LittleEndian.PutUint64(bb, ctransid)
	c[BTRFS_SEND_A_CLONE_CTRANSID] = bb
}

func (c CmdAttrs) GetXattrName() string {
	return string(c[BTRFS_SEND_A_XATTR_NAME])
}

func (c CmdAttrs) SetXattrName(name string) {
	c[BTRFS_SEND_A_XATTR_NAME] = []byte(name)
}

func (c CmdAttrs) GetXattrData() []byte {
	return c[BTRFS_SEND_A_XATTR_DATA]
}

func (c CmdAttrs) SetXattrData(data []byte) {
	c[BTRFS_SEND_A_XATTR_DATA] = data
}

func (c CmdAttrs) GetSize() uint64 {
	return binary.LittleEndian.Uint64(c[BTRFS_SEND_A_SIZE])
}

func (c CmdAttrs) SetSize(size uint64) {
	bb := make([]byte, 8)
	binary.LittleEndian.PutUint64(bb, size)
	c[BTRFS_SEND_A_SIZE] = bb
}

func (c CmdAttrs) GetUid() uint64 {
	return binary.LittleEndian.Uint64(c[BTRFS_SEND_A_UID])
}

func (c CmdAttrs) SetUid(uid uint64) {
	bb := make([]byte, 8)
	binary.LittleEndian.PutUint64(bb, uid)
	c[BTRFS_SEND_A_UID] = bb
}

func (c CmdAttrs) GetGid() uint64 {
	return binary.LittleEndian.Uint64(c[BTRFS_SEND_A_GID])
}

func (c CmdAttrs) SetGid(gid uint64) {
	bb := make([]byte, 8)
	binary.LittleEndian.PutUint64(bb, gid)
	c[BTRFS_SEND_A_GID] = bb
}

func (c CmdAttrs) GetAtime() (time.Time, error) {
	var ts btrfs.BtrfsTimespec
	if err := binary.Read(bytes.NewReader(c[BTRFS_SEND_A_ATIME]), binary.LittleEndian, &ts); err != nil {
		return time.Time{}, err
	}
	return ts.Time(), nil
}

func (c CmdAttrs) SetAtime(atime time.Time) {
	var buf bytes.Buffer
	ts := btrfs.BtrfsTimespec{
		Sec:  uint64(atime.Unix()),
		Nsec: uint32(atime.Nanosecond()),
	}
	binary.Write(&buf, binary.LittleEndian, &ts)
	c[BTRFS_SEND_A_ATIME] = buf.Bytes()
}

func (c CmdAttrs) GetMtime() (time.Time, error) {
	var ts btrfs.BtrfsTimespec
	if err := binary.Read(bytes.NewReader(c[BTRFS_SEND_A_MTIME]), binary.LittleEndian, &ts); err != nil {
		return time.Time{}, err
	}
	return ts.Time(), nil
}

func (c CmdAttrs) SetMtime(mtime time.Time) {
	var buf bytes.Buffer
	ts := btrfs.BtrfsTimespec{
		Sec:  uint64(mtime.Unix()),
		Nsec: uint32(mtime.Nanosecond()),
	}
	binary.Write(&buf, binary.LittleEndian, &ts)
	c[BTRFS_SEND_A_MTIME] = buf.Bytes()
}

func (c CmdAttrs) GetOtime() (time.Time, error) {
	var ts btrfs.BtrfsTimespec
	if err := binary.Read(bytes.NewReader(c[BTRFS_SEND_A_OTIME]), binary.LittleEndian, &ts); err != nil {
		return time.Time{}, err
	}
	return ts.Time(), nil
}

func (c CmdAttrs) SetOtime(otime time.Time) {
	var buf bytes.Buffer
	ts := btrfs.BtrfsTimespec{
		Sec:  uint64(otime.Unix()),
		Nsec: uint32(otime.Nanosecond()),
	}
	binary.Write(&buf, binary.LittleEndian, &ts)
	c[BTRFS_SEND_A_OTIME] = buf.Bytes()
}

func (c CmdAttrs) GetCtime() (time.Time, error) {
	var ts btrfs.BtrfsTimespec
	if err := binary.Read(bytes.NewReader(c[BTRFS_SEND_A_CTIME]), binary.LittleEndian, &ts); err != nil {
		return time.Time{}, err
	}
	return ts.Time(), nil
}

func (c CmdAttrs) SetCtime(ctime time.Time) {
	var buf bytes.Buffer
	ts := btrfs.BtrfsTimespec{
		Sec:  uint64(ctime.Unix()),
		Nsec: uint32(ctime.Nanosecond()),
	}
	binary.Write(&buf, binary.LittleEndian, &ts)
	c[BTRFS_SEND_A_CTIME] = buf.Bytes()
}

func (c CmdAttrs) GetVerityBlockSize() uint32 {
	return binary.LittleEndian.Uint32(c[BTRFS_SEND_A_VERITY_BLOCK_SIZE])
}

func (c CmdAttrs) SetVerityBlockSize(blockSize uint32) {
	bb := make([]byte, 4)
	binary.LittleEndian.PutUint32(bb, blockSize)
	c[BTRFS_SEND_A_VERITY_BLOCK_SIZE] = bb
}

func (c CmdAttrs) GetVeritySalt() []byte {
	return c[BTRFS_SEND_A_VERITY_SALT_DATA]
}

func (c CmdAttrs) SetVeritySalt(salt []byte) {
	c[BTRFS_SEND_A_VERITY_SALT_DATA] = salt
}

func (c CmdAttrs) GetVeritySig() []byte {
	return c[BTRFS_SEND_A_VERITY_SIG_DATA]
}

func (c CmdAttrs) SetVeritySig(sig []byte) {
	c[BTRFS_SEND_A_VERITY_SIG_DATA] = sig
}

func (c CmdAttrs) GetVerityAlgorithm() uint8 {
	return c[BTRFS_SEND_A_VERITY_ALGORITHM][0]
}

func (c CmdAttrs) SetVerityAlgorithm(algorithm uint8) {
	c[BTRFS_SEND_A_VERITY_ALGORITHM] = []byte{algorithm}
}

func (c CmdAttrs) GetFallocateMode() uint32 {
	return binary.LittleEndian.Uint32(c[BTRFS_SEND_A_FALLOCATE_MODE])
}

func (c CmdAttrs) SetFallocateMode(mode uint32) {
	bb := make([]byte, 4)
	binary.LittleEndian.PutUint32(bb, mode)
	c[BTRFS_SEND_A_FALLOCATE_MODE] = bb
}

func (c CmdAttrs) GetFileAttr() uint32 {
	return binary.LittleEndian.Uint32(c[BTRFS_SEND_A_FILEATTR])
}

func (c CmdAttrs) SetFileAttr(fileattr uint32) {
	bb := make([]byte, 4)
	binary.LittleEndian.PutUint32(bb, fileattr)
	c[BTRFS_SEND_A_FILEATTR] = bb
}

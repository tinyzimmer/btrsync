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
	"encoding/binary"
	"fmt"

	"github.com/google/uuid"
	"github.com/tinyzimmer/btrsync/pkg/receive/receivers"
)

const (
	BTRFS_SEND_STREAM_MAGIC          = "btrfs-stream\x00"
	BTRFS_SEND_STREAM_VERSION uint32 = 2
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

func (c CmdAttrs) SubvolInfo(cmd SendCommand) (*receivers.ReceivingSubvolume, error) {
	if cmd != BTRFS_SEND_C_SUBVOL && cmd != BTRFS_SEND_C_SNAPSHOT {
		return nil, fmt.Errorf("not a subvol or snapshot command")
	}
	uuid, err := uuid.FromBytes(c[BTRFS_SEND_A_UUID])
	if err != nil {
		return nil, err
	}
	return &receivers.ReceivingSubvolume{
		Path:     string(c[BTRFS_SEND_A_PATH]),
		UUID:     uuid,
		Ctransid: binary.LittleEndian.Uint64(c[BTRFS_SEND_A_CTRANSID]),
	}, nil
}

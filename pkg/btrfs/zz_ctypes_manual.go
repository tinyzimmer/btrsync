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

package btrfs

import "time"

const BTRFS_SUPER_MAGIC = 0x9123683E

type BtrfsDirItem struct {
	Location BtrfsDiskKey
	Type     uint8
}

type BtrfsRootItem_V0 struct {
	Inode        BtrfsInodeItem
	Generation   uint64
	RootDirID    uint64
	ByteNR       uint64
	ByteLimit    uint64
	BytesUsed    uint64
	LastSnapshot uint64
	Flags        uint64
	Refs         uint32
	DropProgress BtrfsDiskKey
	DropLevel    uint8
	Level        uint8
}

type BtrfsDiskKey struct {
	Objectid uint64
	Type     uint8
	Offset   uint64
}

type BtrfsInodeItem struct {
	Generation uint64
	Transid    uint64
	Size       uint64
	Nbytes     uint64
	Group      uint64
	Nlink      uint32
	Uid        uint32
	Gid        uint32
	Mode       uint32
	Rdev       uint64
	Flags      uint64
	Sequence   uint64
	Reserved   [4]uint64
	Atime      BtrfsTimespec
	Ctime      BtrfsTimespec
	Mtime      BtrfsTimespec
	Otime      BtrfsTimespec
}

type BtrfsRootItem struct {
	Inode          BtrfsInodeItem
	Generation     uint64
	Root_dirid     uint64
	Bytenr         uint64
	Byte_limit     uint64
	Bytes_used     uint64
	Last_snapshot  uint64
	Flags          uint64
	Refs           uint32
	DropProgress   BtrfsDiskKey
	Drop_level     uint8
	Level          uint8
	GenerationV2   uint64
	Uuid           [16]uint8
	Parent_uuid    [16]uint8
	Received_uuid  [16]uint8
	Ctransid       uint64
	Otransid       uint64
	Stransid       uint64
	Rtransid       uint64
	Ctime          BtrfsTimespec
	Otime          BtrfsTimespec
	Stime          BtrfsTimespec
	Rtime          BtrfsTimespec
	Global_tree_id uint64
	Reserved       [7]uint64
}

func (b BtrfsTimespec) Time() time.Time {
	return time.Unix(int64(b.Sec), int64(b.Nsec))
}

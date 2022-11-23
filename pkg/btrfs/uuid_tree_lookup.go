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

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"

	"github.com/google/uuid"
)

var (
	ErrInvalidUUID = errors.New("invalid UUID")
	ErrNotFound    = errors.New("not found")
)

type LookupType uint8

const (
	LookupUUIDKeySubvol         LookupType = 251
	LookupUUIDKeyReceivedSubvol LookupType = 252
)

// UUIDTreeLookupID looks up the subvolume ID for the given UUID.
func UUIDTreeLookupID(path string, uuid uuid.UUID, typ LookupType) (id uint64, err error) {
	key, err := uuidToBtrfsKey(uuid)
	if err != nil {
		return
	}
	key.Type = uint8(typ)
	args := &SearchArgs{
		Key: SearchParams{
			Tree_id:      uint64(UUIDTreeObjectID),
			Min_objectid: key.ObjID,
			Max_objectid: key.ObjID,
			Min_offset:   key.Offset,
			Max_offset:   key.Offset,
			Min_transid:  0,
			Max_transid:  math.MaxUint64,
			Min_type:     uint32(typ),
			Max_type:     uint32(typ),
			Nr_items:     1,
		},
	}
	f, err := os.OpenFile(path, os.O_RDONLY, os.ModeDir)
	if err != nil {
		return
	}
	defer f.Close()
	err = callWriteIoctl(f.Fd(), BTRFS_IOC_TREE_SEARCH, args)
	if err != nil {
		return
	}
	if args.Key.Nr_items < 1 {
		err = fmt.Errorf("no item found for UUID %s", uuid)
		return
	}
	var hdr SearchHeader
	rdr := bytes.NewReader(convertSearchArgsBuf(args))
	if err = binary.Read(rdr, binary.LittleEndian, &hdr); err != nil {
		return
	}
	if hdr.Len == 0 {
		err = fmt.Errorf("no item found for UUID %s", uuid)
		return
	}
	// Read the first ID off the buffer
	err = binary.Read(rdr, binary.LittleEndian, &id)
	return
}

type btrfsKey struct {
	ObjID  uint64
	Type   uint8
	Offset uint64
}

func uuidToBtrfsKey(uuid uuid.UUID) (key btrfsKey, err error) {
	key.ObjID = binary.LittleEndian.Uint64(uuid[:8])
	key.Offset = binary.LittleEndian.Uint64(uuid[8:])
	return
}

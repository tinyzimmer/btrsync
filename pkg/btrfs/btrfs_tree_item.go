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
)

func (s *SearchHeader) ItemType() SearchKey {
	return SearchKey(s.Type)
}

type TreeItem struct {
	Data []byte
	Name string
}

func (t TreeItem) decode(out any) error {
	return binary.Read(bytes.NewReader(t.Data), binary.LittleEndian, out)
}

func (t TreeItem) DirItem() (BtrfsDirItem, error) {
	var out BtrfsDirItem
	return out, t.decode(&out)
}

func (t TreeItem) InodeItem() (BtrfsInodeItem, error) {
	var out BtrfsInodeItem
	return out, t.decode(&out)
}

func (t TreeItem) InodeRef() (BtrfsInodeRef, error) {
	var out BtrfsInodeRef
	return out, t.decode(&out)
}

func (t TreeItem) RootItem() (BtrfsRootItem, error) {
	var out BtrfsRootItem
	return out, t.decode(&out)
}

func (t TreeItem) RootRef() (BtrfsRootRef, string, error) {
	var out BtrfsRootRef
	return out, t.Name, t.decode(&out)
}

func (t TreeItem) DevItem() (BtrfsDevItem, error) {
	var out BtrfsDevItem
	return out, t.decode(&out)
}

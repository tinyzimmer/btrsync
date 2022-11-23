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

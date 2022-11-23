package btrfs

import (
	"os"

	"github.com/google/uuid"
)

type FilesystemInfo struct {
	MaxID        uint64
	NumDevices   uint64
	FSID         uuid.UUID
	NodeSize     uint32
	SectorSize   uint32
	CloneAlign   uint32
	CsumType     uint16
	CsumSize     uint16
	Flags        uint64
	Generate     uint64
	MetadataUUID uuid.UUID
}

// GetFilesystemInfo returns metadata about the filesystem at the given path.
// If the path is not a BTRFS filesystem, an error will be returned.
func GetFilesystemInfo(path string) (*FilesystemInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	rawInfo, err := getFilesystemInfo(f.Fd())
	if err != nil {
		return nil, err
	}
	return &FilesystemInfo{
		MaxID:        rawInfo.Max_id,
		NumDevices:   rawInfo.Num_devices,
		FSID:         uuid.UUID(rawInfo.Fsid),
		NodeSize:     rawInfo.Nodesize,
		SectorSize:   rawInfo.Sectorsize,
		CloneAlign:   rawInfo.Clone_alignment,
		CsumType:     rawInfo.Csum_type,
		CsumSize:     rawInfo.Csum_size,
		Flags:        rawInfo.Flags,
		Generate:     rawInfo.Generation,
		MetadataUUID: uuid.UUID(rawInfo.Metadata_uuid),
	}, nil
}

func getFilesystemInfo(fd uintptr) (*filesystemInfoArgs, error) {
	args := &filesystemInfoArgs{}
	return args, callReadIoctl(fd, BTRFS_IOC_FS_INFO, args)
}

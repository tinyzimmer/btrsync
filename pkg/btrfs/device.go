package btrfs

import (
	"os"

	"github.com/google/uuid"
)

type DeviceInfo struct {
	DeviceID   uint64
	UUID       uuid.UUID
	BytesUsed  uint64
	TotalBytes uint64
	Path       string
}

func GetDeviceInfo(path string) (*DeviceInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	rawInfo, err := getDeviceInfo(f.Fd())
	if err != nil {
		return nil, err
	}
	return &DeviceInfo{
		DeviceID:   rawInfo.Devid,
		UUID:       uuid.UUID(rawInfo.Uuid),
		BytesUsed:  rawInfo.Bytes_used,
		TotalBytes: rawInfo.Total_bytes,
		Path:       string(rawInfo.Path[:]),
	}, nil
}

func getDeviceInfo(fd uintptr) (*deviceInfoArgs, error) {
	args := &deviceInfoArgs{}
	return args, callReadIoctl(fd, BTRFS_IOC_DEV_INFO, args)
}

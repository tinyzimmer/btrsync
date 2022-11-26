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
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/google/uuid"
)

type DeviceInfo struct {
	DeviceID   uint64
	UUID       uuid.UUID
	BytesUsed  uint64
	TotalBytes uint64
	Path       string
}

type DeviceStats struct {
	WriteIOErrors    uint64
	ReadIOErrors     uint64
	FlushIOErrors    uint64
	CorruptionErrors uint64
	GenerationErrors uint64
}

// GetDeviceInfo returns information about the device at the given path or device.
func GetDeviceInfo(path string) (*DeviceInfo, error) {
	var info *BtrfsMount
	var err error
	if strings.HasPrefix(path, "/dev/") {
		info, err = FindMountForDevice(path)
	} else {
		info, err = FindRootMount(path)
	}
	if err != nil {
		return nil, err
	}
	return getDeviceInfoFromRoot(info.Path)
}

func getDeviceInfoFromRoot(rootPath string) (*DeviceInfo, error) {
	f, err := os.OpenFile(rootPath, os.O_RDONLY, os.ModeDir)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	// Find the device ID
	devid, err := getDevID(f.Fd())
	if err != nil {
		return nil, fmt.Errorf("could not get device ID: %w", err)
	}
	rawInfo, err := getDeviceInfo(f.Fd(), devid)
	if err != nil {
		return nil, fmt.Errorf("could not get device info: %w", err)
	}
	return &DeviceInfo{
		DeviceID:   rawInfo.Devid,
		UUID:       uuid.UUID(rawInfo.Uuid),
		BytesUsed:  rawInfo.Bytes_used,
		TotalBytes: rawInfo.Total_bytes,
		Path:       string(rawInfo.Path[:]),
	}, nil
}

// GetDeviceStats returns statistics about the device at the given path or device.
func GetDeviceStats(path string) (*DeviceStats, error) {
	var info *BtrfsMount
	var err error
	if strings.HasPrefix(path, "/dev/") {
		info, err = FindMountForDevice(path)
	} else {
		info, err = FindRootMount(path)
	}
	if err != nil {
		return nil, err
	}
	return getDeviceStatsFromRoot(info.Path)
}

func getDeviceStatsFromRoot(rootPath string) (*DeviceStats, error) {
	f, err := os.OpenFile(rootPath, os.O_RDONLY, os.ModeDir)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	// Find the device ID
	devid, err := getDevID(f.Fd())
	if err != nil {
		return nil, fmt.Errorf("could not get device ID: %w", err)
	}
	rawInfo, err := readDeviceStats(f.Fd(), devid)
	if err != nil {
		return nil, fmt.Errorf("could not get device stats: %w", err)
	}
	return &DeviceStats{
		WriteIOErrors:    rawInfo.Values[0],
		ReadIOErrors:     rawInfo.Values[1],
		FlushIOErrors:    rawInfo.Values[2],
		CorruptionErrors: rawInfo.Values[3],
		GenerationErrors: rawInfo.Values[4],
	}, nil
}

func getDevID(fd uintptr) (uint64, error) {
	params := SearchParams{
		Tree_id:      uint64(ChunkTreeObjectID),
		Min_type:     uint32(DeviceItemKey),
		Max_type:     uint32(DeviceItemKey),
		Min_objectid: uint64(DevItemsObjectID),
		Max_objectid: uint64(DevItemsObjectID),
		Max_offset:   math.MaxUint64,
		Max_transid:  math.MaxUint64,
	}
	var devid uint64
	err := walkBtrfsTreeFd(fd, params, func(hdr SearchHeader, item TreeItem, lastErr error) error {
		if lastErr == nil {
			return lastErr
		}
		if hdr.Type == uint32(DeviceItemKey) {
			item, err := item.DevItem()
			if err != nil {
				return err
			}
			devid = item.Devid
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	if devid == 0 {
		devid++
	}
	return devid, nil
}

func getDeviceInfo(fd uintptr, devid uint64) (*deviceInfoArgs, error) {
	args := &deviceInfoArgs{Devid: devid}
	return args, callWriteIoctl(fd, BTRFS_IOC_DEV_INFO, args)
}

func readDeviceStats(fd uintptr, devid uint64) (*getDeviceStats, error) {
	args := &getDeviceStats{Devid: devid, Items: 5, Flags: 0}
	return args, callWriteIoctl(fd, BTRFS_IOC_GET_DEV_STATS, args)
}

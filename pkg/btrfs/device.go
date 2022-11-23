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

package main

import (
	"fmt"
	"strings"
)

// #include <linux/btrfs.h>
// #include <linux/fsverity.h>
import "C"

// Generic ioctl constants
const (
	IOC_NONE  = 0x0
	IOC_WRITE = 0x1
	IOC_READ  = 0x2

	IOC_NRBITS   = 8
	IOC_TYPEBITS = 8

	IOC_SIZEBITS = 14
	IOC_DIRBITS  = 2

	IOC_NRSHIFT   = 0
	IOC_TYPESHIFT = IOC_NRSHIFT + IOC_NRBITS
	IOC_SIZESHIFT = IOC_TYPESHIFT + IOC_TYPEBITS
	IOC_DIRSHIFT  = IOC_SIZESHIFT + IOC_SIZEBITS

	IOC_NRMASK   = ((1 << IOC_NRBITS) - 1)
	IOC_TYPEMASK = ((1 << IOC_TYPEBITS) - 1)
	IOC_SIZEMASK = ((1 << IOC_SIZEBITS) - 1)
	IOC_DIRMASK  = ((1 << IOC_DIRBITS) - 1)
)

// BTRFS ioctl constants
const (
	BTRFS_IOCTL_MAGIC uintptr = 0x94
)

// cmd is a type cast of uintptr to make it more clear that it is an ioctl.
type IoctlCmd uintptr

func (c IoctlCmd) Size() uintptr {
	return (uintptr(c) >> IOC_SIZESHIFT) & IOC_SIZEMASK
}

// Fsverity ioctl commands
var (
	FS_IOC_ENABLE_VERITY        = _IOW('f', 133, C.sizeof_struct_fsverity_enable_arg)
	FS_IOC_MEASURE_VERITY       = _IOWR('f', 134, C.sizeof_struct_fsverity_digest)
	FS_IOC_READ_VERITY_METADATA = _IOWR('f', 135, C.sizeof_struct_fsverity_read_metadata_arg)
)

// BTRFS ioctl commands
var (
	BTRFS_IOC_SNAP_CREATE         = _IOW(BTRFS_IOCTL_MAGIC, 1, C.sizeof_struct_btrfs_ioctl_vol_args)
	BTRFS_IOC_DEFRAG              = _IOW(BTRFS_IOCTL_MAGIC, 2, C.sizeof_struct_btrfs_ioctl_vol_args)
	BTRFS_IOC_RESIZE              = _IOW(BTRFS_IOCTL_MAGIC, 3, C.sizeof_struct_btrfs_ioctl_vol_args)
	BTRFS_IOC_SCAN_DEV            = _IOW(BTRFS_IOCTL_MAGIC, 4, C.sizeof_struct_btrfs_ioctl_vol_args)
	BTRFS_IOC_FORGET_DEV          = _IOW(BTRFS_IOCTL_MAGIC, 5, C.sizeof_struct_btrfs_ioctl_vol_args)
	BTRFS_IOC_TRANS_START         = _IO(BTRFS_IOCTL_MAGIC, 6)
	BTRFS_IOC_TRANS_END           = _IO(BTRFS_IOCTL_MAGIC, 7)
	BTRFS_IOC_SYNC                = _IO(BTRFS_IOCTL_MAGIC, 8)
	BTRFS_IOC_CLONE               = _IOW(BTRFS_IOCTL_MAGIC, 9, C.sizeof_int)
	BTRFS_IOC_ADD_DEV             = _IOW(BTRFS_IOCTL_MAGIC, 10, C.sizeof_struct_btrfs_ioctl_vol_args)
	BTRFS_IOC_RM_DEV              = _IOW(BTRFS_IOCTL_MAGIC, 11, C.sizeof_struct_btrfs_ioctl_vol_args)
	BTRFS_IOC_BALANCE             = _IOW(BTRFS_IOCTL_MAGIC, 12, C.sizeof_struct_btrfs_ioctl_vol_args)
	BTRFS_IOC_CLONE_RANGE         = _IOW(BTRFS_IOCTL_MAGIC, 13, C.sizeof_struct_btrfs_ioctl_clone_range_args)
	BTRFS_IOC_SUBVOL_CREATE       = _IOW(BTRFS_IOCTL_MAGIC, 14, C.sizeof_struct_btrfs_ioctl_vol_args)
	BTRFS_IOC_SNAP_DESTROY        = _IOW(BTRFS_IOCTL_MAGIC, 15, C.sizeof_struct_btrfs_ioctl_vol_args)
	BTRFS_IOC_DEFRAG_RANGE        = _IOW(BTRFS_IOCTL_MAGIC, 16, C.sizeof_struct_btrfs_ioctl_defrag_range_args)
	BTRFS_IOC_TREE_SEARCH         = _IOWR(BTRFS_IOCTL_MAGIC, 17, C.sizeof_struct_btrfs_ioctl_search_args)
	BTRFS_IOC_TREE_SEARCH_V2      = _IOWR(BTRFS_IOCTL_MAGIC, 17, C.sizeof_struct_btrfs_ioctl_search_args_v2)
	BTRFS_IOC_INO_LOOKUP          = _IOWR(BTRFS_IOCTL_MAGIC, 18, C.sizeof_struct_btrfs_ioctl_ino_lookup_args)
	BTRFS_IOC_DEFAULT_SUBVOL      = _IOW(BTRFS_IOCTL_MAGIC, 19, C.sizeof___u64)
	BTRFS_IOC_SPACE_INFO          = _IOWR(BTRFS_IOCTL_MAGIC, 20, C.sizeof_struct_btrfs_ioctl_space_args)
	BTRFS_IOC_START_SYNC          = _IOR(BTRFS_IOCTL_MAGIC, 24, C.sizeof___u64)
	BTRFS_IOC_WAIT_SYNC           = _IOW(BTRFS_IOCTL_MAGIC, 22, C.sizeof___u64)
	BTRFS_IOC_SNAP_CREATE_V2      = _IOW(BTRFS_IOCTL_MAGIC, 23, C.sizeof_struct_btrfs_ioctl_vol_args_v2)
	BTRFS_IOC_SUBVOL_CREATE_V2    = _IOW(BTRFS_IOCTL_MAGIC, 24, C.sizeof_struct_btrfs_ioctl_vol_args_v2)
	BTRFS_IOC_SUBVOL_GETFLAGS     = _IOR(BTRFS_IOCTL_MAGIC, 25, C.sizeof___u64)
	BTRFS_IOC_SUBVOL_SETFLAGS     = _IOW(BTRFS_IOCTL_MAGIC, 26, C.sizeof___u64)
	BTRFS_IOC_SCRUB               = _IOWR(BTRFS_IOCTL_MAGIC, 27, C.sizeof_struct_btrfs_ioctl_scrub_args)
	BTRFS_IOC_SCRUB_CANCEL        = _IO(BTRFS_IOCTL_MAGIC, 28)
	BTRFS_IOC_SCRUB_PROGRESS      = _IOWR(BTRFS_IOCTL_MAGIC, 29, C.sizeof_struct_btrfs_ioctl_scrub_args)
	BTRFS_IOC_DEV_INFO            = _IOWR(BTRFS_IOCTL_MAGIC, 30, C.sizeof_struct_btrfs_ioctl_dev_info_args)
	BTRFS_IOC_FS_INFO             = _IOR(BTRFS_IOCTL_MAGIC, 31, C.sizeof_struct_btrfs_ioctl_fs_info_args)
	BTRFS_IOC_BALANCE_V2          = _IOWR(BTRFS_IOCTL_MAGIC, 32, C.sizeof_struct_btrfs_ioctl_balance_args)
	BTRFS_IOC_BALANCE_CTL         = _IOW(BTRFS_IOCTL_MAGIC, 33, C.sizeof_int)
	BTRFS_IOC_BALANCE_PROGRESS    = _IOR(BTRFS_IOCTL_MAGIC, 34, C.sizeof_struct_btrfs_ioctl_balance_args)
	BTRFS_IOC_INO_PATHS           = _IOWR(BTRFS_IOCTL_MAGIC, 35, C.sizeof_struct_btrfs_ioctl_ino_path_args)
	BTRFS_IOC_LOGICAL_INO         = _IOWR(BTRFS_IOCTL_MAGIC, 36, C.sizeof_struct_btrfs_ioctl_logical_ino_args)
	BTRFS_IOC_SET_RECEIVED_SUBVOL = _IOWR(BTRFS_IOCTL_MAGIC, 37, C.sizeof_struct_btrfs_ioctl_received_subvol_args)
	BTRFS_IOC_SEND                = _IOW(BTRFS_IOCTL_MAGIC, 38, C.sizeof_struct_btrfs_ioctl_send_args)
	BTRFS_IOC_DEVICES_READY       = _IOR(BTRFS_IOCTL_MAGIC, 39, C.sizeof_struct_btrfs_ioctl_vol_args)
	BTRFS_IOC_QUOTA_CTL           = _IOWR(BTRFS_IOCTL_MAGIC, 40, C.sizeof_struct_btrfs_ioctl_quota_ctl_args)
	BTRFS_IOC_QGROUP_ASSIGN       = _IOW(BTRFS_IOCTL_MAGIC, 41, C.sizeof_struct_btrfs_ioctl_qgroup_assign_args)
	BTRFS_IOC_QGROUP_CREATE       = _IOW(BTRFS_IOCTL_MAGIC, 42, C.sizeof_struct_btrfs_ioctl_qgroup_create_args)
	BTRFS_IOC_QGROUP_LIMIT        = _IOR(BTRFS_IOCTL_MAGIC, 43, C.sizeof_struct_btrfs_ioctl_qgroup_limit_args)
	BTRFS_IOC_QUOTA_RESCAN        = _IOW(BTRFS_IOCTL_MAGIC, 44, C.sizeof_struct_btrfs_ioctl_quota_rescan_args)
	BTRFS_IOC_QUOTA_RESCAN_STATUS = _IOR(BTRFS_IOCTL_MAGIC, 45, C.sizeof_struct_btrfs_ioctl_quota_rescan_args)
	BTRFS_IOC_QUOTA_RESCAN_WAIT   = _IO(BTRFS_IOCTL_MAGIC, 46)
	BTRFS_IOC_GET_DEV_STATS       = _IOWR(BTRFS_IOCTL_MAGIC, 52, C.sizeof_struct_btrfs_ioctl_get_dev_stats)
	BTRFS_IOC_DEV_REPLACE         = _IOWR(BTRFS_IOCTL_MAGIC, 53, C.sizeof_struct_btrfs_ioctl_dev_replace_args)
	BTRFS_IOC_FILE_EXTENT_SAME    = _IOWR(BTRFS_IOCTL_MAGIC, 54, C.sizeof_struct_btrfs_ioctl_same_args)
	BTRFS_IOC_RM_DEV_V2           = _IOW(BTRFS_IOCTL_MAGIC, 58, C.sizeof_struct_btrfs_ioctl_vol_args_v2)
	BTRFS_IOC_LOGICAL_INO_V2      = _IOWR(BTRFS_IOCTL_MAGIC, 59, C.sizeof_struct_btrfs_ioctl_logical_ino_args)
	BTRFS_IOC_GET_SUBVOL_INFO     = _IOR(BTRFS_IOCTL_MAGIC, 60, C.sizeof_struct_btrfs_ioctl_get_subvol_info_args)
	BTRFS_IOC_GET_SUBVOL_ROOTREF  = _IOWR(BTRFS_IOCTL_MAGIC, 61, C.sizeof_struct_btrfs_ioctl_get_subvol_rootref_args)
	BTRFS_IOC_INO_LOOKUP_USER     = _IOWR(BTRFS_IOCTL_MAGIC, 62, C.sizeof_struct_btrfs_ioctl_ino_lookup_user_args)
	BTRFS_IOC_SNAP_DESTROY_V2     = _IOW(BTRFS_IOCTL_MAGIC, 63, C.sizeof_struct_btrfs_ioctl_vol_args_v2)
	BTRFS_IOC_ENCODED_READ        = _IOR(BTRFS_IOCTL_MAGIC, 64, C.sizeof_struct_btrfs_ioctl_encoded_io_args)
	BTRFS_IOC_ENCODED_WRITE       = _IOW(BTRFS_IOCTL_MAGIC, 64, C.sizeof_struct_btrfs_ioctl_encoded_io_args)
)

// _IOC generates an IOC command.
func _IOC(dir, t, nr, size uintptr) IoctlCmd {
	return IoctlCmd((dir << IOC_DIRSHIFT) | (t << IOC_TYPESHIFT) |
		(nr << IOC_NRSHIFT) | (size << IOC_SIZESHIFT))
}

// _IOR generates an IOR command.
func _IOR(t, nr, size uintptr) IoctlCmd {
	return _IOC(IOC_READ, t, nr, size)
}

// _IOW generates an IOW command.
func _IOW(t, nr, size uintptr) IoctlCmd {
	return _IOC(IOC_WRITE, t, nr, size)
}

// _IOWR generates an IOWR command.
func _IOWR(t, nr, size uintptr) IoctlCmd {
	return _IOC(IOC_READ|IOC_WRITE, t, nr, size)
}

// _IO generates an IO command.
func _IO(t, nr uintptr) IoctlCmd {
	return _IOC(IOC_NONE, t, nr, 0)
}

func main() {
	var sb strings.Builder
	sb.WriteString(`
// Code generated by gen/ioctlgen/main.go; DO NOT EDIT.

package btrfs

// Generic ioctl constants
const (
	IOC_NONE  = 0x0
	IOC_WRITE = 0x1
	IOC_READ  = 0x2

	IOC_NRBITS   = 8
	IOC_TYPEBITS = 8

	IOC_SIZEBITS = 14
	IOC_DIRBITS  = 2

	IOC_NRSHIFT   = 0
	IOC_TYPESHIFT = IOC_NRSHIFT + IOC_NRBITS
	IOC_SIZESHIFT = IOC_TYPESHIFT + IOC_TYPEBITS
	IOC_DIRSHIFT  = IOC_SIZESHIFT + IOC_SIZEBITS

	IOC_NRMASK   = ((1 << IOC_NRBITS) - 1)
	IOC_TYPEMASK = ((1 << IOC_TYPEBITS) - 1)
	IOC_SIZEMASK = ((1 << IOC_SIZEBITS) - 1)
	IOC_DIRMASK  = ((1 << IOC_DIRBITS) - 1)
)

// BTRFS ioctl constants
const (
	BTRFS_IOCTL_MAGIC uintptr = 0x94
)

// IoctlCmd is a type cast of uintptr to make it more clear that it is an ioctl.
type IoctlCmd uintptr

func (c IoctlCmd) Size() uintptr {
	return (uintptr(c) >> IOC_SIZESHIFT) & IOC_SIZEMASK
}
`)
	sb.WriteString(fmt.Sprintf(`
// Fsverity ioctl commands
const (
	FS_IOC_ENABLE_VERITY        IoctlCmd = 0x%02x
	FS_IOC_MEASURE_VERITY       IoctlCmd = 0x%02x
	FS_IOC_READ_VERITY_METADATA IoctlCmd = 0x%02x
)

// BTRFS ioctl commands
const (
	BTRFS_IOC_SNAP_CREATE         IoctlCmd = 0x%02x
	BTRFS_IOC_DEFRAG              IoctlCmd = 0x%02x
	BTRFS_IOC_RESIZE              IoctlCmd = 0x%02x
	BTRFS_IOC_SCAN_DEV            IoctlCmd = 0x%02x
	BTRFS_IOC_FORGET_DEV          IoctlCmd = 0x%02x
	BTRFS_IOC_TRANS_START         IoctlCmd = 0x%02x
	BTRFS_IOC_TRANS_END           IoctlCmd = 0x%02x
	BTRFS_IOC_SYNC                IoctlCmd = 0x%02x
	BTRFS_IOC_CLONE               IoctlCmd = 0x%02x
	BTRFS_IOC_ADD_DEV             IoctlCmd = 0x%02x
	BTRFS_IOC_RM_DEV              IoctlCmd = 0x%02x
	BTRFS_IOC_BALANCE             IoctlCmd = 0x%02x
	BTRFS_IOC_CLONE_RANGE         IoctlCmd = 0x%02x
	BTRFS_IOC_SUBVOL_CREATE       IoctlCmd = 0x%02x
	BTRFS_IOC_SNAP_DESTROY        IoctlCmd = 0x%02x
	BTRFS_IOC_DEFRAG_RANGE        IoctlCmd = 0x%02x
	BTRFS_IOC_TREE_SEARCH         IoctlCmd = 0x%02x
	BTRFS_IOC_TREE_SEARCH_V2      IoctlCmd = 0x%02x
	BTRFS_IOC_INO_LOOKUP          IoctlCmd = 0x%02x
	BTRFS_IOC_DEFAULT_SUBVOL      IoctlCmd = 0x%02x
	BTRFS_IOC_SPACE_INFO          IoctlCmd = 0x%02x
	BTRFS_IOC_START_SYNC          IoctlCmd = 0x%02x
	BTRFS_IOC_WAIT_SYNC           IoctlCmd = 0x%02x
	BTRFS_IOC_SNAP_CREATE_V2      IoctlCmd = 0x%02x
	BTRFS_IOC_SUBVOL_CREATE_V2    IoctlCmd = 0x%02x
	BTRFS_IOC_SUBVOL_GETFLAGS     IoctlCmd = 0x%02x
	BTRFS_IOC_SUBVOL_SETFLAGS     IoctlCmd = 0x%02x
	BTRFS_IOC_SCRUB               IoctlCmd = 0x%02x
	BTRFS_IOC_SCRUB_CANCEL        IoctlCmd = 0x%02x
	BTRFS_IOC_SCRUB_PROGRESS      IoctlCmd = 0x%02x
	BTRFS_IOC_DEV_INFO            IoctlCmd = 0x%02x
	BTRFS_IOC_FS_INFO             IoctlCmd = 0x%02x
	BTRFS_IOC_BALANCE_V2          IoctlCmd = 0x%02x
	BTRFS_IOC_BALANCE_CTL         IoctlCmd = 0x%02x
	BTRFS_IOC_BALANCE_PROGRESS    IoctlCmd = 0x%02x
	BTRFS_IOC_INO_PATHS           IoctlCmd = 0x%02x
	BTRFS_IOC_LOGICAL_INO         IoctlCmd = 0x%02x
	BTRFS_IOC_SET_RECEIVED_SUBVOL IoctlCmd = 0x%02x
	BTRFS_IOC_SEND                IoctlCmd = 0x%02x
	BTRFS_IOC_DEVICES_READY       IoctlCmd = 0x%02x
	BTRFS_IOC_QUOTA_CTL           IoctlCmd = 0x%02x
	BTRFS_IOC_QGROUP_ASSIGN       IoctlCmd = 0x%02x
	BTRFS_IOC_QGROUP_CREATE       IoctlCmd = 0x%02x
	BTRFS_IOC_QGROUP_LIMIT        IoctlCmd = 0x%02x
	BTRFS_IOC_QUOTA_RESCAN        IoctlCmd = 0x%02x
	BTRFS_IOC_QUOTA_RESCAN_STATUS IoctlCmd = 0x%02x
	BTRFS_IOC_QUOTA_RESCAN_WAIT   IoctlCmd = 0x%02x
	BTRFS_IOC_GET_DEV_STATS       IoctlCmd = 0x%02x
	BTRFS_IOC_DEV_REPLACE         IoctlCmd = 0x%02x
	BTRFS_IOC_FILE_EXTENT_SAME    IoctlCmd = 0x%02x
	BTRFS_IOC_RM_DEV_V2           IoctlCmd = 0x%02x
	BTRFS_IOC_LOGICAL_INO_V2      IoctlCmd = 0x%02x
	BTRFS_IOC_GET_SUBVOL_INFO     IoctlCmd = 0x%02x
	BTRFS_IOC_GET_SUBVOL_ROOTREF  IoctlCmd = 0x%02x
	BTRFS_IOC_INO_LOOKUP_USER     IoctlCmd = 0x%02x
	BTRFS_IOC_SNAP_DESTROY_V2     IoctlCmd = 0x%02x
	BTRFS_IOC_ENCODED_READ        IoctlCmd = 0x%02x
	BTRFS_IOC_ENCODED_WRITE       IoctlCmd = 0x%02x
)
`,
		FS_IOC_ENABLE_VERITY,
		FS_IOC_MEASURE_VERITY,
		FS_IOC_READ_VERITY_METADATA,

		BTRFS_IOC_SNAP_CREATE,
		BTRFS_IOC_DEFRAG,
		BTRFS_IOC_RESIZE,
		BTRFS_IOC_SCAN_DEV,
		BTRFS_IOC_FORGET_DEV,
		BTRFS_IOC_TRANS_START,
		BTRFS_IOC_TRANS_END,
		BTRFS_IOC_SYNC,
		BTRFS_IOC_CLONE,
		BTRFS_IOC_ADD_DEV,
		BTRFS_IOC_RM_DEV,
		BTRFS_IOC_BALANCE,
		BTRFS_IOC_CLONE_RANGE,
		BTRFS_IOC_SUBVOL_CREATE,
		BTRFS_IOC_SNAP_DESTROY,
		BTRFS_IOC_DEFRAG_RANGE,
		BTRFS_IOC_TREE_SEARCH,
		BTRFS_IOC_TREE_SEARCH_V2,
		BTRFS_IOC_INO_LOOKUP,
		BTRFS_IOC_DEFAULT_SUBVOL,
		BTRFS_IOC_SPACE_INFO,
		BTRFS_IOC_START_SYNC,
		BTRFS_IOC_WAIT_SYNC,
		BTRFS_IOC_SNAP_CREATE_V2,
		BTRFS_IOC_SUBVOL_CREATE_V2,
		BTRFS_IOC_SUBVOL_GETFLAGS,
		BTRFS_IOC_SUBVOL_SETFLAGS,
		BTRFS_IOC_SCRUB,
		BTRFS_IOC_SCRUB_CANCEL,
		BTRFS_IOC_SCRUB_PROGRESS,
		BTRFS_IOC_DEV_INFO,
		BTRFS_IOC_FS_INFO,
		BTRFS_IOC_BALANCE_V2,
		BTRFS_IOC_BALANCE_CTL,
		BTRFS_IOC_BALANCE_PROGRESS,
		BTRFS_IOC_INO_PATHS,
		BTRFS_IOC_LOGICAL_INO,
		BTRFS_IOC_SET_RECEIVED_SUBVOL,
		BTRFS_IOC_SEND,
		BTRFS_IOC_DEVICES_READY,
		BTRFS_IOC_QUOTA_CTL,
		BTRFS_IOC_QGROUP_ASSIGN,
		BTRFS_IOC_QGROUP_CREATE,
		BTRFS_IOC_QGROUP_LIMIT,
		BTRFS_IOC_QUOTA_RESCAN,
		BTRFS_IOC_QUOTA_RESCAN_STATUS,
		BTRFS_IOC_QUOTA_RESCAN_WAIT,
		BTRFS_IOC_GET_DEV_STATS,
		BTRFS_IOC_DEV_REPLACE,
		BTRFS_IOC_FILE_EXTENT_SAME,
		BTRFS_IOC_RM_DEV_V2,
		BTRFS_IOC_LOGICAL_INO_V2,
		BTRFS_IOC_GET_SUBVOL_INFO,
		BTRFS_IOC_GET_SUBVOL_ROOTREF,
		BTRFS_IOC_INO_LOOKUP_USER,
		BTRFS_IOC_SNAP_DESTROY_V2,
		BTRFS_IOC_ENCODED_READ,
		BTRFS_IOC_ENCODED_WRITE,
	))

	fmt.Println(sb.String())
}

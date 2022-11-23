package btrfs

// #include <linux/btrfs.h>
// #include <linux/btrfs_tree.h>
// #include <linux/fsverity.h>
import "C"
import "fmt"

type ObjectID uint64

type SearchKey uint32

type CompressionType uint32

// Flags
const (
	// Subvolume flags
	SubvolReadOnly = C.BTRFS_SUBVOL_RDONLY

	// SendFlags
	NoFileData       = 0x1
	OmitStreamHeader = 0x2
	OmitEndCommand   = 0x3
	SendVersion      = 0x8
	SendCompressed   = 0x10

	// Lookup Flags
	RootTreeObjectID       ObjectID = C.BTRFS_ROOT_TREE_OBJECTID
	ExtentTreeObjectID     ObjectID = C.BTRFS_EXTENT_TREE_OBJECTID
	ChunkTreeObjectID      ObjectID = C.BTRFS_CHUNK_TREE_OBJECTID
	DevTreeObjectID        ObjectID = C.BTRFS_DEV_TREE_OBJECTID
	FSTreeObjectID         ObjectID = C.BTRFS_FS_TREE_OBJECTID
	RootTreeDirObjectID    ObjectID = C.BTRFS_ROOT_TREE_DIR_OBJECTID
	CSumTreeObjectID       ObjectID = C.BTRFS_CSUM_TREE_OBJECTID
	QuotaTreeObjectID      ObjectID = C.BTRFS_QUOTA_TREE_OBJECTID
	UUIDTreeObjectID       ObjectID = C.BTRFS_UUID_TREE_OBJECTID
	FreeSpaceTreeObjectID  ObjectID = C.BTRFS_FREE_SPACE_TREE_OBJECTID
	BlockGroupTreeObjectID ObjectID = C.BTRFS_BLOCK_GROUP_TREE_OBJECTID
	DevStatsObjectID       ObjectID = C.BTRFS_DEV_STATS_OBJECTID
	BalanceObjectID        ObjectID = C.BTRFS_BALANCE_OBJECTID
	OrphanObjectID         ObjectID = C.BTRFS_ORPHAN_OBJECTID
	TreeLogObjectID        ObjectID = C.BTRFS_TREE_LOG_OBJECTID
	TreeLogFixupObjectID   ObjectID = C.BTRFS_TREE_LOG_FIXUP_OBJECTID
	TreeRelocObjectID      ObjectID = C.BTRFS_TREE_RELOC_OBJECTID
	DataRelocTreeObjectID  ObjectID = C.BTRFS_DATA_RELOC_TREE_OBJECTID
	ExtentCSumObjectID     ObjectID = C.BTRFS_EXTENT_CSUM_OBJECTID
	FreeSpaceObjectID      ObjectID = C.BTRFS_FREE_SPACE_OBJECTID
	FreeInoObjectID        ObjectID = C.BTRFS_FREE_INO_OBJECTID
	MultipleObjectIDs      ObjectID = C.BTRFS_MULTIPLE_OBJECTIDS

	FirstFreeObjectID ObjectID = C.BTRFS_FIRST_FREE_OBJECTID
	LastFreeObjectID  ObjectID = C.BTRFS_LAST_FREE_OBJECTID

	DirItemKey     SearchKey = C.BTRFS_DIR_ITEM_KEY
	InodeRefKey    SearchKey = C.BTRFS_INODE_REF_KEY
	InodeItemKey   SearchKey = C.BTRFS_INODE_ITEM_KEY
	RootItemKey    SearchKey = C.BTRFS_ROOT_ITEM_KEY
	RootRefKey     SearchKey = C.BTRFS_ROOT_REF_KEY
	RootBackrefKey SearchKey = C.BTRFS_ROOT_BACKREF_KEY

	// CompressionTypes
	CompressionNone   CompressionType = C.BTRFS_ENCODED_IO_COMPRESSION_NONE
	CompressionZLib   CompressionType = C.BTRFS_ENCODED_IO_COMPRESSION_ZLIB
	CompressionZSTD   CompressionType = C.BTRFS_ENCODED_IO_COMPRESSION_ZSTD
	CompressionLZO4k  CompressionType = C.BTRFS_ENCODED_IO_COMPRESSION_LZO_4K
	CompressionLZO8k  CompressionType = C.BTRFS_ENCODED_IO_COMPRESSION_LZO_8K
	CompressionLZO16k CompressionType = C.BTRFS_ENCODED_IO_COMPRESSION_LZO_16K
	CompressionLZO32k CompressionType = C.BTRFS_ENCODED_IO_COMPRESSION_LZO_32K
	CompressionLZO64k CompressionType = C.BTRFS_ENCODED_IO_COMPRESSION_LZO_64K
)

func (o ObjectID) IntString() string {
	return fmt.Sprintf("%d", o)
}

type balanceArgs C.struct_btrfs_balance_args

type balanceProgress C.struct_btrfs_balance_progress

// type BtrfsDiskKey C.struct_btrfs_disk_key

// type BtrfsInodeItem C.struct_btrfs_inode_item

// type BtrfsRootItem C.struct_btrfs_root_item

type BtrfsRootRef C.struct_btrfs_root_ref

type BtrfsInodeRef C.struct_btrfs_inode_ref

type BtrfsTimespec C.struct_btrfs_timespec

type cloneRangeArgs C.struct_btrfs_ioctl_clone_range_args

type defragRangeArgs C.struct_btrfs_ioctl_defrag_range_args

type deviceInfoArgs C.struct_btrfs_ioctl_dev_info_args

type deviceReplaceArgs C.struct_btrfs_ioctl_dev_replace_args

type deviceReplaceStartParams C.struct_btrfs_ioctl_dev_replace_start_params

type featureFlags C.struct_btrfs_ioctl_feature_flags

type filesystemInfoArgs C.struct_btrfs_ioctl_fs_info_args

type fsVerityDigest C.struct_fsverity_digest

type fsVerityEnableArg C.struct_fsverity_enable_arg

type fsVerityReadMetadataArg C.struct_fsverity_read_metadata_arg

type getDeviceStats C.struct_btrfs_ioctl_get_dev_stats

type getSubvolumeInfoArgs C.struct_btrfs_ioctl_get_subvol_info_args

type getSubvolumeRootRefArgs C.struct_btrfs_ioctl_get_subvol_rootref_args

type inoLookupArgs C.struct_btrfs_ioctl_ino_lookup_args

type inoLookupUserArgs C.struct_btrfs_ioctl_ino_lookup_user_args

type inoPathArgs C.struct_btrfs_ioctl_ino_path_args

type ioctlBalanceArgs C.struct_btrfs_ioctl_balance_args

type logicalINOArgs C.struct_btrfs_ioctl_logical_ino_args

type qgroupAssignArgs C.struct_btrfs_ioctl_qgroup_assign_args

type qgroupCreateArgs C.struct_btrfs_ioctl_qgroup_create_args

type qgroupLimit C.struct_btrfs_qgroup_limit

type qgroupLimitArgs C.struct_btrfs_ioctl_qgroup_limit_args

type quotaCTLArgs C.struct_btrfs_ioctl_quota_ctl_args

type quotaRescanArgs C.struct_btrfs_ioctl_quota_rescan_args

type receivedSubvolArgs C.struct_btrfs_ioctl_received_subvol_args

type rootRef C.struct___0

type sameArgs C.struct_btrfs_ioctl_same_args

type scrubArgs C.struct_btrfs_ioctl_scrub_args

type scrubProgress C.struct_btrfs_scrub_progress

type SearchHeader C.struct_btrfs_ioctl_search_header

type SearchParams C.struct_btrfs_ioctl_search_key

type SearchArgs C.struct_btrfs_ioctl_search_args

type searchArgsV2 C.struct_btrfs_ioctl_search_args_v2

type sendArgs C.struct_btrfs_ioctl_send_args

type spaceArgs C.struct_btrfs_ioctl_space_args

type timespec C.struct_btrfs_ioctl_timespec

type volumeArgs C.struct_btrfs_ioctl_vol_args

type volumeArgsV2 C.struct_btrfs_ioctl_vol_args_v2

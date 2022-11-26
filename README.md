# btrsync

[![Go Reference](https://pkg.go.dev/badge/github.com/tinyzimmer/btrsync.svg)](https://pkg.go.dev/github.com/tinyzimmer/btrsync)

A library and tool for working with btrfs filesystems and snapshots in Golang.

## Features

Beyond the native (no CGO*) bindings for working with BTRFS file systems provided in `pkg`, the `btrsync` utility included has the following features:

 * Manage and sync snapshots to local and remote locations
 * Automatic volume and subvolume discovery for easy config generation
 * Time machine app for browsing local snapshots
 * Recovery of interrupted transfers by natively scanning the btrfs send streams
 * Mount a btrfs sendfile as an in-memory FUSE filesystem
 * More, but I'm too lazy to document right now

Btrsync can be run either as a daemon process, cron job, or from the command line. 
It will manage snapshots and their mirrors according to its configuration or command line flags.

**Cgo is used to generate certain constants and structures in the codebase, but not at compile time*

## Library Usage

For comprehensive usage of the bindings, see the go.dev. But below are overviews of some common operations:

### Volumes

Volume-wide bindings (such as interacting with RAID levels) are basically non-existant at the moment, but more will potentially come.

```go
package main

import (
	"fmt"
	"path/filepath"

	"github.com/tinyzimmer/btrsync/pkg/btrfs"
)

func main() {
	// List all mounted Btrfs paths
	mounts, err := btrfs.ListBtrfsMounts()
	if err != nil {
		panic(err)
	}
	for _, mount := range mounts {
		fmt.Println(mount.Device) // The device the filesystem is on
		isBtrfs, err := btrfs.IsBtrfs(mount.Path)
		fmt.Println(isBtrfs, err) // Would print true for all mounts

		path := filepath.Join(mount.Path, "some-directory")
		root, err := btrfs.FindRootMount(path)
		fmt.Println(root, err) // Would print the mount itself

		// Get usage information about the device
		info, err := btrfs.GetDeviceInfo(root.Path) // or root.DeviceInfo()
		fmt.Println(info, err)

		// Get statistics about the device
		stats, err := btrfs.GetDeviceStats(root.Path) // or root.DeviceStats()
		fmt.Println(stats, err)
	}
}
```

### Subvolumes

```go
package main

import (
	"fmt"

	"github.com/tinyzimmer/btrsync/pkg/btrfs"
)

func main() {
	// Create a subvolume
	err := btrfs.CreateSubvolume("/mnt/btrfs/subvol")
	if err != nil {
		panic(err)
	}
	fmt.Println(btrfs.IsSubvolume("/mnt/btrfs/subvol")) // true
	// Retrieve information about the subvolume. See docs for other search options.
	info, err := btrfs.SubvolumeSearch(btrfs.SearchWithPath("/mnt/btrfs/subvol"))
	if err != nil {
		panic(err)
	}
	fmt.Println(info)

	// Make the subvolume read-only
	err = btrfs.SetSubvolumeReadOnly("/mnt/btrfs/subvol")
	if err != nil {
		panic(err)
	}

	// Try to delete the subvolume
	err = btrfs.DeleteSubvolume("/mnt/btrfs/subvol", false)
	fmt.Println(err) // fails: subvolume is read-only
	// Force delete removes read-only flag
	err = btrfs.DeleteSubvolume("/mnt/btrfs/subvol", true)
	if err != nil {
		panic(err)
	}

	// Build a red-black tree of a volume or subvolume
	tree, err := btrfs.BuildRBTree("/mnt/btrfs")
	if err != nil {
		panic(err)
	}

	// Iterate the tree in-order
	tree.InOrderIterate(func(info *btrfs.RootInfo, lastErr error) error {
		fmt.Println(info.UUID)
		return nil
	})
}
```

### Snapshots

```go
package main

import (
	"fmt"

	"github.com/tinyzimmer/btrsync/pkg/btrfs"
)

func main() {
	// Create a snapshot at /mnt/btrfs/subvol/snapshot
	err := btrfs.CreateSnapshot("/mnt/btrfs/subvol", btrfs.WithSnapshotName("snapshot"))
	if err != nil { 
		return err 
	}
	// Create a snapshot using the full path to the snapshot (must reside on the same BTRFS volume)
	err = btrfs.CreateSnapshot("/mnt/btrfs/subvol", btrfs.WithSnapshotPath("/mnt/btrfs/subvol/snapshots/snapshot-1"))
	if err != nil { 
		return err 
	}
	// Delete a snapshot
	err = btrfs.DeleteSnapshot("/mnt/btrfs/subvol/snapshots/snapshot-1")
	if err != nil {
		panic(err)
	}
}
```

### Sending

_TODO: For now see the go.dev docs_

### Receiving

_TODO: For now see the go.dev docs_

## Contributing

PRs are welcome! Feel free to open issues for found bugs, but for simple addition of an `ioctl` or two it would be preferable to open a PR. Also, feel free to open issues for feature and/or bug discussions about the `btrsync` and `btrtm` utilities.
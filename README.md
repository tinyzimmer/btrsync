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

Volume-wide bindings (such as interacting with RAID levels) are very barebones at the moment, but more will potentially come.

_TODO_

### Subvolumes

_TODO_

TODO: Scrubbing/Balancing

### Snapshots

_TODO_

### Sending

_TODO_

### Receiving

_TODO_

## Contributing

PRs are welcome! Feel free to open issues for found bugs, but for simple addition of an `ioctl` or two it would be preferable to open a PR. Also, feel free to open issues for feature and/or bug discussions about the `btrsync` and `btrtm` utilities.
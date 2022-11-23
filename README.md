# btrsync

A library and tool for working with btrfs filesystems and snapshots in Golang.

## Features

Beyond the native (no CGO*) bindings for working with BTRFS file systems provided in `pkg`, the `btrsync` utility included has the following features:

 * Mount a Btrfs sendfile as an in-memory FUSE filesystem
 * More, but I'm too lazy to document right now

**Cgo is used to generate certain constants and structures in the codebase, but not at compile time*
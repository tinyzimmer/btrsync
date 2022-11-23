#!/bin/bash

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
set -ex

# Generate ioctl definitions
go run "$SCRIPT_DIR/ioctlgen/main.go" > zz_ioctl_defs.go

# Generate ioctl structures
go tool cgo -godefs "${SCRIPT_DIR}/ioctl_ctypes.go" > zz_ioctl_types.go

# Patch Clone_sources in sendArgs
sed -i 's/Clone_sources\t.*/Clone_sources  uint64/' zz_ioctl_types.go

# Generate Stringers
stringer -type IoctlCmd,ObjectID,SearchKey,CompressionType -output zz_stringers.go

# Run gofmt on generated files
go fmt zz_ioctl_defs.go 1> /dev/null
go fmt zz_ioctl_types.go 1> /dev/null

# Remove object files
rm -rf _obj
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
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"unsafe"
)

type sendCtx struct {
	context.Context
	args      *sendArgs
	osPipe    *os.File
	logger    *log.Logger
	verbosity int
}

type SendOption func(*sendCtx) error

// SendWithLogger will log the send operation to the given logger.
func SendWithLogger(logger *log.Logger, verbosity int) SendOption {
	return func(ctx *sendCtx) error {
		ctx.logger = logger
		ctx.verbosity = verbosity
		return nil
	}
}

// SendWithCloneSources will use the given snapshots as clone sources for an
// incremental send.
func SendWithCloneSources(sources ...string) SendOption {
	return func(ctx *sendCtx) error {
		ctx.args.Clone_sources_count = uint64(len(sources))
		srcs := make([]uint64, len(sources))
		for i, source := range sources {
			f, err := os.OpenFile(source, os.O_RDONLY, os.ModeDir)
			if err != nil {
				return err
			}
			rootID, err := lookupRootIDFromFd(f.Fd())
			if err != nil {
				return err
			}
			srcs[i] = rootID
			if err := f.Close(); err != nil {
				return err
			}
		}
		srcptr := uintptr(unsafe.Pointer(&srcs[0]))
		ctx.args.Clone_sources = uint64(srcptr)
		return nil
	}
}

// SendWithParentRoot will send an incremental send from the given parent root.
func SendWithParentRoot(root string) SendOption {
	return func(ctx *sendCtx) error {
		f, err := os.OpenFile(root, os.O_RDONLY, os.ModeDir)
		if err != nil {
			return err
		}
		defer f.Close()
		id, err := lookupRootIDFromFd(f.Fd())
		if err != nil {
			return err
		}
		ctx.args.Parent_root = id
		return nil
	}
}

// SendWithoutData will send a send stream without any data. This is useful for
// getting a list of files that have changed.
func SendWithoutData() SendOption {
	return func(ctx *sendCtx) error {
		ctx.args.Flags |= NoFileData
		return nil
	}
}

// SendCompressedData
func SendCompressedData() SendOption {
	return func(ctx *sendCtx) error {
		ctx.args.Flags |= SendCompressed
		return nil
	}
}

// SendToPath will send a send stream to the given path as a file.
func SendToPath(path string) SendOption {
	return func(ctx *sendCtx) error {
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		return SendToFile(f)(ctx)
	}
}

// SendToFile will send a send stream to the given os.File.
func SendToFile(f *os.File) SendOption {
	return func(ctx *sendCtx) error {
		ctx.args.Send_fd = int64(f.Fd())
		ctx.osPipe = f
		return nil
	}
}

// SendToPipe creates and returns an option and pipe to read the stream from.
func SendToPipe() (SendOption, *os.File, error) {
	rf, wf, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	return SendToFile(wf), rf, nil
}

// Send will send the snapshot at source with the given options.
// Source must be a path to a read-only snapshot.
func Send(source string, opts ...SendOption) error {
	ctx := &sendCtx{
		Context: context.Background(),
		args:    &sendArgs{Version: 2},
		logger:  log.New(io.Discard, "", 0),
	}
	for _, opt := range opts {
		opt(ctx)
	}
	// We only do version 2 so we always send the version flag
	ctx.args.Flags |= SendVersion
	if ctx.args.Send_fd == 0 {
		return errors.New("no send target specified")
	}
	if ctx.verbosity >= 2 {
		ctx.logger.Printf("opening snapshot at %q for send", source)
	}
	f, err := os.OpenFile(source, os.O_RDONLY, os.ModeDir)
	if err != nil {
		return err
	}
	defer f.Close()
	if ctx.osPipe != nil {
		defer ctx.osPipe.Close()
	}
	if ctx.verbosity > 1 {
		ctx.logger.Printf("sending snapshot %s", source)
	}
	if err := callWriteIoctl(f.Fd(), BTRFS_IOC_SEND, ctx.args); err != nil {
		return fmt.Errorf("error sending snapshot: %w", err)
	}
	return nil
}

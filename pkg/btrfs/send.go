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
	"sync"
	"unsafe"
)

type sendCtx struct {
	context.Context
	args      *sendArgs
	wg        sync.WaitGroup
	osPipe    *os.File
	errors    chan (error)
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

// SendToPath will send a send stream to the given path.
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

// SendToWriter will send a send stream to the given io.Writer.
func SendToWriter(w io.Writer) SendOption {
	return func(ctx *sendCtx) error {
		rf, wf, err := os.Pipe()
		if err != nil {
			return err
		}
		ctx.osPipe = wf
		ctx.args.Send_fd = int64(wf.Fd())
		ctx.wg.Add(1)
		go func() {
			defer ctx.wg.Done()
			defer rf.Close()
			_, err := io.Copy(w, rf)
			if err != nil {
				ctx.errors <- fmt.Errorf("error copying send stream to writer: %w", err)
				return
			}
		}()
		return nil
	}
}

// SendWithContext applies a context to the send operation.
func SendWithContext(ctx context.Context) SendOption {
	return func(ctx *sendCtx) error {
		ctx.Context = ctx
		return nil
	}
}

// Send will send the snapshot at source with the given options.
// Source must be a path to a read-only snapshot.
func Send(source string, opts ...SendOption) error {
	ctx := &sendCtx{
		Context: context.Background(),
		args:    &sendArgs{Version: 2},
		errors:  make(chan (error), 1),
		wg:      sync.WaitGroup{},
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
	go func() {
		if ctx.osPipe != nil {
			defer ctx.osPipe.Close()
		}
		if ctx.verbosity > 1 {
			ctx.logger.Printf("sending snapshot %s", source)
		}
		if err := callWriteIoctl(f.Fd(), BTRFS_IOC_SEND, ctx.args); err != nil {
			ctx.errors <- fmt.Errorf("error sending snapshot: %w", err)
			return
		}
		ctx.errors <- nil
	}()
	select {
	case <-ctx.Done():
		return fmt.Errorf("context finished: %w", ctx.Err())
	case err := <-ctx.errors:
		if err == nil {
			ctx.wg.Wait()
		}
		return err
	}
}

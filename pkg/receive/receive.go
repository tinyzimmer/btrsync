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

// Package receive implements a receiver for btrfs send streams.
package receive

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/google/uuid"
	"github.com/tinyzimmer/btrsync/pkg/receive/receivers"
	"github.com/tinyzimmer/btrsync/pkg/receive/receivers/nop"
	"github.com/tinyzimmer/btrsync/pkg/sendstream"
)

var (
	// ErrInvalidSendCommand is returned when an invalid send command is encountered.
	ErrInvalidSendCommand = errors.New("invalid send command")
)

// ProcessSendStream will process a send stream and apply it to the receiver with the given options.
func ProcessSendStream(r io.Reader, opts ...Option) error {
	// Initialize a context
	ctx := &receiveCtx{
		Context:  context.Background(),
		log:      log.New(io.Discard, "", 0),
		receiver: nop.New(),
	}
	// Apply options
	for _, opt := range opts {
		if err := opt(ctx); err != nil {
			return err
		}
	}
	var cancel func()
	ctx.Context, cancel = context.WithCancel(ctx.Context)

	// Start an error counter and create a stream scanner
	var streamErrors int
	stream := sendstream.NewScanner(r, ctx.ignoreChecksums)

	// Scan the stream in a goroutine so we can block on either the context or the stream
	// itself. This allows us to stop processing the stream if the context is canceled.
	errCh := make(chan error, 1)
	go func() {
		defer cancel()
		if ctx.startOffset > 0 {
			ctx.log.Printf("Skipping to offset %d", ctx.startOffset)
		}
		for stream.Scan() {
			cmd, attrs := stream.Command()
			if ctx.verbosity >= 2 {
				ctx.log.Println("processing send cmd:", cmd.Cmd)
			}

			// Check if we are seeking
			if ctx.startOffset > ctx.currentOffset {
				ctx.currentOffset++
				if cmd.Cmd == sendstream.BTRFS_SEND_C_SUBVOL || cmd.Cmd == sendstream.BTRFS_SEND_C_SNAPSHOT {
					path := string(attrs[sendstream.BTRFS_SEND_A_PATH])
					ctransid := binary.LittleEndian.Uint64(attrs[sendstream.BTRFS_SEND_A_CTRANSID])
					uuid, err := uuid.FromBytes(attrs[sendstream.BTRFS_SEND_A_UUID])
					if err != nil {
						errCh <- fmt.Errorf("error parsing uuid: %s", err)
						return
					}
					ctx.log.Printf("Resuming subvol %s", path)
					ctx.currentSubvolInfo = &sendstream.ReceivingSubvolume{
						Path: path, UUID: uuid, Ctransid: ctransid,
					}
				}
				if ctx.verbosity >= 2 {
					ctx.log.Printf("skipping cmd at offset %d", ctx.currentOffset)
				}
				continue
			}

			// Run any preop functions
			if preOp, ok := ctx.receiver.(receivers.PreOpReceiver); ok {
				err := preOp.PreOp(ctx, cmd, attrs)
				if err != nil {
					ctx.currentOffset++
					if !errors.Is(err, receivers.ErrSkipCommand) {
						streamErrors++
						if streamErrors >= ctx.maxErrors {
							errCh <- fmt.Errorf("max errors reached (%d): last error: %w", streamErrors, err)
							return
						}
						ctx.log.Printf("Error processing pre-op: %s", err)
					}
					continue
				}
			}

			// Dispatch the command
			var err error
			if cmd.Cmd == sendstream.BTRFS_SEND_C_END {
				if ctx.honorEndCmd {
					if ctx.currentSubvolInfo != nil {
						if err := ctx.receiver.FinishSubvolume(ctx); err != nil {
							ctx.log.Printf("Error finishing subvolume: %s", err)
						}
					}
					return
				}
				err = ctx.receiver.FinishSubvolume(ctx)
				ctx.currentSubvolInfo = nil
			} else if f, ok := processFuncs[cmd.Cmd]; ok {
				err = f(ctx, attrs)
			} else {
				err = fmt.Errorf("%w: %d", ErrInvalidSendCommand, cmd.Cmd)
			}
			if err != nil && !errors.Is(err, receivers.ErrSkipCommand) {
				ctx.log.Println("error processing command:", err)
				streamErrors++
				if streamErrors >= ctx.maxErrors {
					errCh <- fmt.Errorf("max errors reached (%d): last error: %w", streamErrors, err)
					return
				}
			}

			// Run any post op functions
			if postOp, ok := ctx.receiver.(receivers.PostOpReceiver); ok {
				err := postOp.PostOp(ctx, cmd, attrs)
				if err != nil && !errors.Is(err, receivers.ErrSkipCommand) {
					streamErrors++
					if streamErrors >= ctx.maxErrors {
						errCh <- fmt.Errorf("max errors reached (%d): last error: %w", streamErrors, err)
						return
					}
					ctx.log.Printf("Error processing pre-op: %s", err)
				}
			}

			// Increment the offset
			ctx.currentOffset++
		}

		// Check for any stream errors
		if err := stream.Err(); err != nil {
			errCh <- stream.Err()
			return
		}

		if ctx.currentSubvolInfo != nil {
			if err := ctx.receiver.FinishSubvolume(ctx); err != nil {
				ctx.log.Printf("Error finishing subvolume: %s", err)
			}
		}
	}()
	<-ctx.Context.Done()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return ctx.Err()
	}
	ctx.LogVerbose(1, "context finished, checking for errors from stream")
	close(errCh)
	for err := range errCh {
		if err != nil {
			return err
		}
	}
	return nil
}

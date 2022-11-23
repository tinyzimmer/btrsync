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

package receive

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/tinyzimmer/btrsync/pkg/receive/receivers/nop"
	"github.com/tinyzimmer/btrsync/pkg/sendstream"
)

var (
	// ErrInvalidSendCommand is returned when an invalid send command is encountered.
	ErrInvalidSendCommand = errors.New("invalid send command")
)

// ProcessSendStream will process a send stream and apply it to the receiver with the given options.
func ProcessSendStream(r io.Reader, opts ...ReceiveOption) error {
	// Initialize a context
	ctx := &receiveCtx{
		Context:  context.Background(),
		log:      log.New(io.Discard, "", 0),
		receiver: nop.New(),
		errors:   make(chan error, 1),
	}
	// Apply options
	for _, opt := range opts {
		if err := opt(ctx); err != nil {
			return err
		}
	}

	// Start an error counter and create a stream scanner
	var errors int
	stream := sendstream.NewScanner(r, ctx.ignoreChecksums)

	var incrementer *sendstream.Scanner
	if ctx.incrementAgainst != nil {
		incrementer = sendstream.NewScanner(ctx.incrementAgainst, ctx.ignoreChecksums)
	}

	// Scan the stream in a goroutine so we can block on either the context or the stream
	// itself. This allows us to stop processing the stream if the context is canceled.
	go func() {
		defer func() {
			if ctx.currentSubvolInfo != nil {
				if err := ctx.receiver.FinishSubvolume(ctx); err != nil {
					ctx.log.Printf("Error finishing subvolume: %s", err)
				}
			}
		}()
		for stream.Scan() {
			cmd, attrs := stream.Command()
			if ctx.verbosity >= 2 {
				ctx.log.Println("processing send cmd:", cmd.Cmd)
			}
			if incrementer != nil && incrementer.Scan() {
				if cmd.Cmd == sendstream.BTRFS_SEND_C_SUBVOL || cmd.Cmd == sendstream.BTRFS_SEND_C_SNAPSHOT {
					// If we are incrementing, we need to make sure we are tracking the current subvolume in
					// the context.
					if ctx.currentSubvolInfo != nil {
						err := ctx.receiver.FinishSubvolume(ctx)
						if err != nil {
							errors++
							if errors >= ctx.maxErrors {
								ctx.errors <- fmt.Errorf("max errors reached (%d): last error: %w", errors, err)
								return
							}
							ctx.log.Printf("Error finishing subvolume: %s", err)
						}
						ctx.currentSubvolInfo = nil
					}
					curSubvol, err := attrs.SubvolInfo(cmd.Cmd)
					if err != nil {
						ctx.errors <- fmt.Errorf("failed to get subvol info from attrs: %w", err)
						return
					}
					ctx.currentSubvolInfo = curSubvol
				}
				rcmd, _ := incrementer.Command()
				if cmd.Cmd == rcmd.Cmd && cmd.Crc == rcmd.Crc {
					// If the commands are the same and the checksums match, we can skip it.
					if ctx.verbosity >= 2 {
						ctx.log.Println("skipping send cmd as checksums match:", cmd.Cmd)
					}
					continue
				}
			}
			if incrementer != nil {
				if err := incrementer.Err(); err != nil {
					ctx.errors <- fmt.Errorf("failed to scan incrementer: %w", err)
					return
				}
			}
			var err error
			if cmd.Cmd == sendstream.BTRFS_SEND_C_END {
				if ctx.honorEndCmd {
					ctx.errors <- nil
					return
				}
				err = ctx.receiver.FinishSubvolume(ctx)
				ctx.currentSubvolInfo = nil
			} else if f, ok := processFuncs[cmd.Cmd]; ok {
				err = f(ctx, attrs)
			} else {
				err = fmt.Errorf("%w: %d", ErrInvalidSendCommand, cmd.Cmd)
			}
			if err != nil {
				ctx.log.Println("error processing command:", err)
				errors++
				if errors >= ctx.maxErrors {
					ctx.errors <- fmt.Errorf("max errors reached (%d): last error: %w", errors, err)
					return
				}
			}
		}
		if err := stream.Err(); err != nil {
			ctx.errors <- stream.Err()
			return
		}
		ctx.errors <- nil
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("context finished: %w", ctx.Err())
	case err := <-ctx.errors:
		return err
	}
}

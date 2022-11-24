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
	"log"

	"github.com/tinyzimmer/btrsync/pkg/receive/receivers"
)

// Option is a function that can be passed to ProcessSendStream to configure
// the behavior of the receiver.
type Option func(*receiveCtx) error

// HonorEndCommand will cause the receiver to stop processing the stream when it
// encounters a BTRFS_SEND_C_END command.
func HonorEndCommand() Option {
	return func(args *receiveCtx) error {
		args.honorEndCmd = true
		return nil
	}
}

// WithLogger will set the logger for the receiver to use. Defaults to a logger
// that discards all output. Increasing the verbosity will cause the logger to
// print more information about the processing of the stream.
func WithLogger(logger *log.Logger, verbosity int) Option {
	return func(args *receiveCtx) error {
		args.log = logger
		args.verbosity = verbosity
		return nil
	}
}

// WithMaxErrors will set the maximum number of errors that can occur before the
// receiver stops processing the stream. Defaults to 1.
func WithMaxErrors(maxErrors int) Option {
	return func(args *receiveCtx) error {
		args.maxErrors = maxErrors
		return nil
	}
}

// WithContext will set the context for the receiver to use. Defaults to a
// background context.
func WithContext(ctx context.Context) Option {
	return func(args *receiveCtx) error {
		args.Context = ctx
		return nil
	}
}

// ForceDecompress will cause the receiver to decompress any compressed data
// it encounters in the stream. This is useful if the stream is compressed
// but the receiver does not support compression.
func ForceDecompress() Option {
	return func(args *receiveCtx) error {
		args.forceDecompress = true
		return nil
	}
}

// IgnoreChecksums will cause the receiver to ignore crc32 checksums in the stream.
// This has no effect on IncrementAgainst.
func IgnoreChecksums() Option {
	return func(args *receiveCtx) error {
		args.ignoreChecksums = true
		return nil
	}
}

// To will set the receiver to use for the stream. Defaults to a nop receiver.
func To(rcvr receivers.Receiver) Option {
	return func(args *receiveCtx) error {
		args.receiver = rcvr
		return nil
	}
}

// FromOffset will start processing the stream at the given command offset.
// This is useful if you want to resume a stream that was interrupted.
func FromOffset(offset uint64) Option {
	return func(args *receiveCtx) error {
		args.startOffset = offset
		return nil
	}
}

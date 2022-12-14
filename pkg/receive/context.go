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
	"github.com/tinyzimmer/btrsync/pkg/sendstream"
)

type receiveCtx struct {
	context.Context
	// Options
	log             *log.Logger
	verbosity       int
	maxErrors       int
	honorEndCmd     bool
	forceDecompress bool
	receiver        receivers.Receiver
	ignoreChecksums bool
	startOffset     uint64
	currentOffset   uint64
	// State
	currentSubvolInfo *sendstream.ReceivingSubvolume
}

func (r *receiveCtx) CurrentSubvolume() *sendstream.ReceivingSubvolume {
	return r.currentSubvolInfo
}

func (r *receiveCtx) ResolvePath(path string) string {
	return r.currentSubvolInfo.ResolvePath(path)
}

func (r *receiveCtx) LogVerbose(level int, format string, args ...interface{}) {
	if r.verbosity >= level {
		r.log.Printf(format, args...)
	}
}

func (r *receiveCtx) CurrentOffset() uint64 {
	return r.currentOffset
}

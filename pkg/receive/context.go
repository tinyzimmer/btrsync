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
	"io"
	"log"

	"github.com/tinyzimmer/btrsync/pkg/receive/receivers"
)

type receiveCtx struct {
	context.Context
	// Options
	log              *log.Logger
	verbosity        int
	maxErrors        int
	honorEndCmd      bool
	forceDecompress  bool
	receiver         receivers.Receiver
	ignoreChecksums  bool
	incrementAgainst io.Reader
	// State
	currentSubvolInfo *receivers.ReceivingSubvolume
	// Channel for returned error, if any
	errors chan (error)
}

func (r *receiveCtx) CurrentSubvolume() *receivers.ReceivingSubvolume {
	return r.currentSubvolInfo
}

func (r *receiveCtx) ResolvePath(path string) string {
	return r.currentSubvolInfo.ResolvePath(path)
}

func (r *receiveCtx) Log() *log.Logger {
	return r.log
}

func (r *receiveCtx) Verbosity() int {
	return r.verbosity
}

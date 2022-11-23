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

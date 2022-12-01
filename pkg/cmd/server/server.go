package server

import (
	"context"
	"fmt"

	"github.com/tinyzimmer/btrsync/pkg/cmd/config"
)

// ErrInvalidServerProtocol is returned when an invalid server protocol is specified.
var ErrInvalidServerProtocol = fmt.Errorf("invalid server protocol")

// Server is an interface for a btrsync server.
type Server interface {
	// Start the server
	Start() error
	// Stop the server
	Stop(ctx context.Context) error
}

// New returns a new server based on the configuration.
func New(conf *config.Config) (Server, error) {
	switch conf.Server.Protocol {
	case config.ServerProtoHTTP:
		return NewHTTPServer(conf), nil
	default:
		return nil, ErrInvalidServerProtocol
	}
}

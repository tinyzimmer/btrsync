package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/tinyzimmer/btrsync/pkg/cmd/config"
	"github.com/tinyzimmer/btrsync/pkg/receive"
	"github.com/tinyzimmer/btrsync/pkg/receive/receivers/local"
)

type HTTPServer struct {
	*http.Server
	logger *log.Logger
	config *config.Config
}

func NewHTTPServer(conf *config.Config) Server {
	logger := log.New(os.Stderr, "http: ", log.LstdFlags)
	s := &HTTPServer{Server: &http.Server{
		Addr:         net.JoinHostPort(conf.Server.ListenAddress, strconv.Itoa(conf.Server.ListenPort)),
		ReadTimeout:  300 * time.Second,
		WriteTimeout: 300 * time.Second,
		ErrorLog:     logger,
	}, logger: logger, config: conf}
	s.Server.Handler = s
	return s
}

func (s *HTTPServer) Start() error {
	if s.config.Server.TLSCertFile != "" && s.config.Server.TLSKeyFile != "" {
		s.logger.Printf("Starting HTTPS server on %s", s.Addr)
		return s.ListenAndServeTLS(s.config.Server.TLSCertFile, s.config.Server.TLSKeyFile)
	}
	s.logger.Printf("Starting HTTP server on %s", s.Addr)
	return s.ListenAndServe()
}

func (s *HTTPServer) Stop(ctx context.Context) error {
	s.logger.Printf("Stopping HTTP server")
	return s.Shutdown(ctx)
}

func (s *HTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.logger.Printf("%s %s", r.Method, r.URL.Path)
	if r.Method != http.MethodPost {
		s.logger.Printf("Invalid method %s", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		s.logger.Printf("Invalid path %s", r.URL.Path)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	destination := filepath.Join(s.config.Server.DataDirectory, path)
	s.logger.Printf("Receiving volume to %q", destination)
	defer r.Body.Close()

	err := receive.ProcessSendStream(r.Body,
		receive.WithLogger(s.logger, s.config.Verbosity),
		receive.WithContext(r.Context()),
		receive.HonorEndCommand(),
		receive.To(local.New(destination)),
	)
	if err != nil {
		msg := fmt.Sprintf("Error receiving volume: %v\n", err)
		s.logger.Printf(msg)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(msg))
		return
	}

	w.Write([]byte(fmt.Sprintf("Successfully received volume to %q\n", destination)))
}

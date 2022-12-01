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

package cmd

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/tinyzimmer/btrsync/pkg/cmd/config"
	"github.com/tinyzimmer/btrsync/pkg/cmd/server"
)

func NewServeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the btrsync server",
		RunE:  runServe,
	}

	cmd.Flags().Var(&conf.Server.Protocol, "protocol", "The protocol to use for the server")
	cmd.Flags().StringVar(&conf.Server.ListenAddress, "address", config.DefaultServerAddress, "The address to bind the server to")
	cmd.Flags().IntVar(&conf.Server.ListenPort, "port", config.DefaultServerPort, "The port to bind the server to")
	cmd.Flags().StringVar(&conf.Server.TLSCertFile, "tls-cert", "", "The path to the TLS certificate file")
	cmd.Flags().StringVar(&conf.Server.TLSKeyFile, "tls-key", "", "The path to the TLS key file")
	cmd.Flags().StringVar(&conf.Server.DataDirectory, "data-dir", "", "The path to the data directory")

	v.BindPFlag("server.protocol", cmd.Flags().Lookup("protocol"))
	v.BindPFlag("server.listen_address", cmd.Flags().Lookup("address"))
	v.BindPFlag("server.listen_port", cmd.Flags().Lookup("port"))
	v.BindPFlag("server.tls_cert_file", cmd.Flags().Lookup("tls-cert"))
	v.BindPFlag("server.tls_key_file", cmd.Flags().Lookup("tls-key"))
	v.BindPFlag("server.data_directory", cmd.Flags().Lookup("data-dir"))

	return cmd
}

func runServe(cmd *cobra.Command, args []string) error {
	srvr, err := server.New(&conf)
	if err != nil {
		return err
	}

	go func() {
		if err := srvr.Start(); err != nil && err != http.ErrServerClosed {
			logger.Fatal(err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	return srvr.Stop(cmd.Context())
}

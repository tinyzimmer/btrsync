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
	"io"
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/tinyzimmer/btrsync/pkg/receive"
	"github.com/tinyzimmer/btrsync/pkg/receive/receivers/local"
)

var (
	receivefile string
)

func NewReceiveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "receive [flags] <dest>",
		Short: "Receive a snapshot from a local or remote host",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runReceive,
	}
	cmd.Flags().StringVarP(&receivefile, "file", "f", "", "receive from encoded file")
	return cmd
}

func runReceive(cmd *cobra.Command, args []string) error {
	var src io.Reader = os.Stdin
	var err error
	if receivefile != "" {
		logLevel(1, "Receiving from file %s\n", receivefile)
		src, err = os.Open(receivefile)
		if err != nil {
			return err
		}
	} else {
		logLevel(1, "Receiving stream from stdin")
	}
	dest := args[0]
	logLevel(0, "Receiving to %q", dest)
	return receive.ProcessSendStream(src,
		receive.WithLogger(log.New(os.Stderr, "[receive]", log.LstdFlags|log.Lshortfile), conf.Verbosity),
		receive.HonorEndCommand(),
		receive.To(local.New(dest)),
	)
}

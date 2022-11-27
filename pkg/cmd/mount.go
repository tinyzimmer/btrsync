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
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	fusefs "github.com/hanwen/go-fuse/v2/fs"
	"github.com/spf13/cobra"

	"github.com/tinyzimmer/btrsync/pkg/receive"
	"github.com/tinyzimmer/btrsync/pkg/receive/receivers/memfs"
)

func NewMountCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "mount [file] [mountpoint]",
		Short: "Create and mount a FUSE filesystem of a sent snapshot",
		Args:  cobra.ExactArgs(2),
		RunE:  mount,
	}
}

func mount(cmd *cobra.Command, args []string) error {
	src, dest := args[0], args[1]
	if stat, err := os.Stat(dest); err != nil {
		return fmt.Errorf("cannot mount to %s: %w", dest, err)
	} else if !stat.IsDir() {
		return fmt.Errorf("cannot mount to %s: not a directory", dest)
	}
	snap, err := os.Open(src)
	if err != nil {
		return err
	}

	fs := memfs.New()

	logLevel(0, "Receiving btrfs stream to in-memory filesystem")
	err = receive.ProcessSendStream(
		snap,
		receive.HonorEndCommand(),
		receive.WithLogger(logger, conf.Verbosity),
		receive.To(fs),
	)
	if err != nil {
		logger.Fatal("Error processing send stream: ", err)
	}
	if err := snap.Close(); err != nil {
		logger.Fatal("Error closing send stream: ", err)
	}

	logLevel(0, "Mounting in-memory filesystem at %q", dest)
	timeout := time.Second
	server, err := fusefs.Mount(dest, fs, &fusefs.Options{
		AttrTimeout:  &timeout,
		EntryTimeout: &timeout,
	})
	if err != nil {
		return err
	}
	logLevel(0, "Serving FUSE filesystem")
	go server.Wait()
	ch := make(chan os.Signal, 1)

	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch

	logLevel(0, "Unmounting FUSE filesystem")
	return server.Unmount()
}

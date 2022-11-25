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
	"errors"
	"log"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"

	"github.com/tinyzimmer/btrsync/pkg/btrfs"
)

var (
	forcesend  bool
	sendfile   string
	compressed bool
)

func NewSendCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send [flags] <subvolumes>...",
		Short: "Send a snapshot to a local or remote host",
		Args:  cobra.MinimumNArgs(1),
		RunE:  send,
	}

	cmd.Flags().BoolVarP(&forcesend, "force", "f", false, "force source to be readonly if it already isn't")
	cmd.Flags().StringVarP(&sendfile, "output", "o", "", "send to encoded file")
	cmd.Flags().BoolVarP(&compressed, "compressed", "z", false, "send compressed data")

	return cmd
}

func send(cmd *cobra.Command, args []string) error {
	src := args[0]
	isReadonly, err := btrfs.IsSubvolumeReadOnly(src)
	if err != nil {
		return err
	}
	if !isReadonly {
		if !forcesend {
			return errors.New("source subvolume must be readonly")
		}
		logger.Println("Source subvolume is not readonly, setting readonly flag")
		if err := btrfs.SetSubvolumeReadOnly(src, true); err != nil {
			return err
		}

	}
	var dest *os.File
	if sendfile != "" {
		logLevel(0, "Sending to file %s", sendfile)
		dest, err = os.Create(sendfile)
	} else if len(args) == 1 {
		_, err = unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ)
		if err == nil {
			err = errors.New("stdout is a terminal, please specify an output file")
		} else {
			logLevel(0, "Sending stream to stdout")
			dest = os.Stdout
			err = nil
		}
	} else {
		err = errors.New("must specify an output file")
	}
	if err != nil {
		return err
	}
	defer dest.Close()
	var opts []btrfs.SendOption
	opts = append(opts,
		btrfs.SendToFile(dest),
		btrfs.SendWithLogger(log.New(os.Stderr, "[send]", log.LstdFlags|log.Lshortfile), conf.Verbosity),
	)
	if compressed {
		opts = append(opts, btrfs.SendCompressedData())
	}
	return btrfs.Send(src, opts...)
}

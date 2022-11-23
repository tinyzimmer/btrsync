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
		if config.Verbosity > 0 {
			logger.Printf("Receiving from file %s\n", receivefile)
		}
		src, err = os.Open(receivefile)
		if err != nil {
			return err
		}
	} else {
		if config.Verbosity > 0 {
			logger.Println("Receiving stream from stdin")
		}
	}
	dest := args[0]
	logger.Println("Receiving to", dest)
	return receive.ProcessSendStream(src,
		receive.WithLogger(log.New(os.Stderr, "[receive]", log.LstdFlags|log.Lshortfile), config.Verbosity),
		receive.HonorEndCommand(),
		receive.To(local.New(dest)),
	)
}

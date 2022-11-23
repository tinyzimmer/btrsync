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

	logger.Println("Receiving btrfs stream to in-memory filesystem")
	err = receive.ProcessSendStream(
		snap,
		receive.HonorEndCommand(),
		receive.WithLogger(logger, config.Verbosity),
		receive.To(fs),
	)
	if err != nil {
		logger.Fatal("Error processing send stream: ", err)
	}
	if err := snap.Close(); err != nil {
		logger.Fatal("Error closing send stream: ", err)
	}

	logger.Println("Mounting in-memory filesystem at", dest)
	timeout := time.Second
	server, err := fusefs.Mount(dest, fs, &fusefs.Options{
		AttrTimeout:  &timeout,
		EntryTimeout: &timeout,
	})
	if err != nil {
		return err
	}
	logger.Println("Serving FUSE filesystem")
	go server.Wait()
	ch := make(chan os.Signal, 1)

	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch

	logger.Println("Unmounting FUSE filesystem")
	return server.Unmount()
}

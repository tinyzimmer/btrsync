package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
)

var (
	envPrefix = "BTRSYNC"

	cfgFile string
	config  Config

	logger = log.New(os.Stderr, "", log.LstdFlags)
)

func Execute(version string) {
	if err := NewRootCommand(version).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
}

func NewRootCommand(version string) *cobra.Command {
	var rootCommand = &cobra.Command{
		Use:               "btrsync [flags] <source> <destination>",
		Short:             "A tool for syncing btrfs subvolumes and snapshots",
		SilenceErrors:     true,
		SilenceUsage:      true,
		Version:           version,
		PersistentPreRunE: initConfig,
	}

	rootCommand.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file")
	rootCommand.PersistentFlags().CountVarP(&config.Verbosity, "verbose", "v", "verbosity level (can be used multiple times)")

	rootCommand.AddCommand(NewRunCommand())
	rootCommand.AddCommand(NewSendCommand())
	rootCommand.AddCommand(NewReceiveCommand())
	rootCommand.AddCommand(NewPruneCommand())
	rootCommand.AddCommand(NewTreeCommand())
	rootCommand.AddCommand(NewMountCommand())

	return rootCommand
}

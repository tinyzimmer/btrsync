package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tinyzimmer/btrsync/cmd/btrsync/cmd/timemachine"
)

func NewTimeMachineCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "timemachine",
		Aliases: []string{"tm"},
		Short:   "Run the time machine tui",
		RunE: func(cmd *cobra.Command, args []string) error {
			return timemachine.Run(&conf)
		},
	}
}

package cmd

import "github.com/spf13/cobra"

func NewPruneCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "prune",
		Short: "Prune local and remote snapshots",
		RunE:  prune,
	}
}

func prune(cmd *cobra.Command, args []string) error {

	return nil
}

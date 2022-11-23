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

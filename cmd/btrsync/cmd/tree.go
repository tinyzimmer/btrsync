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

	"github.com/spf13/cobra"
	"github.com/xlab/treeprint"

	"github.com/tinyzimmer/btrsync/pkg/btrfs"
)

func NewTreeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tree [flags] <volume>",
		Short: "Print a tree of subvolumes and snapshots",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runTree,
	}
	return cmd
}

func runTree(cmd *cobra.Command, args []string) error {
	path := args[0]

	// Find the root device
	rootMount, err := btrfs.FindRootMount(path)
	if err != nil {
		return err
	}

	// Find the root ID of the subvolume we are descending from
	subvol, err := btrfs.SubvolumeSearch(btrfs.SearchWithRootMount(rootMount), btrfs.SearchWithPath(path))
	if err != nil {
		return err
	}

	rbtree, err := btrfs.BuildRBTree(path)
	if err != nil {
		return err
	}
	rbtree = rbtree.FilterFromRoot(subvol.RootID)

	// Start a treeprinter from the root device
	treeprint.IndentSize = 4
	tree := treeprint.NewWithRoot(rootMount)

	// Iterate the tree and add nodes to the treeprinter
	rbtree.InOrderIterate(func(info *btrfs.RootInfo, err error) error {
		if info.RefTree == 0 || info.RootID == btrfs.FSTreeObjectID {
			return nil
		}
		key := fmt.Sprintf(" %s", info.RootID.IntString())
		lookupKey := fmt.Sprintf(" %s", info.RefTree.IntString())
		name := info.Path
		node := tree.FindByMeta(lookupKey)
		if node == nil {
			tree.AddMetaNode(key, name)
		} else {
			node.AddMetaNode(key, name)
		}
		return nil
	})

	// Dump the results
	fmt.Println(tree.String())
	return nil
}

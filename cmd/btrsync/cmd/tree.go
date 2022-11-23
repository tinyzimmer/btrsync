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
	rbtree, err := btrfs.BuildRBTree(path)
	if err != nil {
		return err
	}

	// Find the root device
	rootMount, err := btrfs.FindRootMount(path)
	if err != nil {
		return err
	}

	// Start a treeprinter from the root device
	treeprint.IndentSize = 4
	tree := treeprint.NewWithRoot(rootMount)

	// Iterate the tree and add nodes to the treeprinter
	rbtree.InOrderIterate(func(info *btrfs.RootInfo, err error) error {
		if info.RefTree == 0 || info.RootID == btrfs.FSTreeObjectID {
			return nil
		}
		node := tree.FindByMeta(" " + info.RefTree.IntString())
		if node == nil {
			tree.AddMetaNode(" "+info.RootID.IntString(), info.Path)
		} else {
			node.AddMetaNode(" "+info.RootID.IntString(), info.Path)
		}
		return nil
	})

	// Dump the results
	fmt.Println(tree.String())
	return nil
}

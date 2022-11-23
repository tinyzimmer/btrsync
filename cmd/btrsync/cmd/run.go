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
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/tinyzimmer/btrsync/pkg/btrfs"
)

func NewRunCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Run a sync operation based on the configuration",
		RunE:  run,
	}
}

func run(cmd *cobra.Command, args []string) error {
	if err := ensureSnapshotSubvolumes(); err != nil {
		return err
	}
	logger.Println("Creating snapshots...")
	if err := createSnapshots(); err != nil {
		return err
	}
	return nil
}

func createSnapshots() error {
	for _, vol := range config.Volumes {
		for _, subvol := range vol.Subvolumes {
			snapDir := config.ResolveSnapshotPath(vol.GetName(), subvol.GetName())
			snapName := fmt.Sprintf("%s.%s",
				subvol.GetSnapshotName(),
				time.Now().Format("2006-01-02_15-04-05"))
			sourcePath := filepath.Join(vol.Path, subvol.Path)
			snapFullPath := filepath.Join(snapDir, snapName)
			if config.Verbosity >= 1 {
				logger.Printf("Creating read-only snapshot %q from %q\n", snapFullPath, sourcePath)
			}
			if err := btrfs.CreateSnapshot(
				sourcePath,
				btrfs.WithSnapshotPath(snapFullPath),
				btrfs.WithReadOnlySnapshot(),
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func ensureSnapshotSubvolumes() error {
	for _, vol := range config.Volumes {
	Subvolumes:
		for _, subvol := range vol.Subvolumes {
			snapDir := config.ResolveSnapshotPath(vol.GetName(), subvol.GetName())
			isSubvol, err := btrfs.IsSubvolume(snapDir)
			if err != nil {
				if !os.IsNotExist(err) {
					return err
				}
				logger.Printf("Creating snapshot subvolume %s\n", snapDir)
				if err := btrfs.CreateSubvolume(snapDir); err != nil {
					return err
				}
				continue Subvolumes
			}
			if !isSubvol {
				return fmt.Errorf("%s is not a btrfs subvolume", snapDir)
			}
			if config.Verbosity >= 2 {
				logger.Printf("Snapshot subvolume %s already exists\n", snapDir)
			}
		}
	}
	return nil
}

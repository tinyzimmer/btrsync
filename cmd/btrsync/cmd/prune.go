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
	"context"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tinyzimmer/btrsync/cmd/btrsync/cmd/snapmanager"
	"github.com/tinyzimmer/btrsync/cmd/btrsync/cmd/syncmanager"
)

func NewPruneCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "prune",
		Short: "Prune local and remote snapshots",
		RunE:  prune,
	}
}

func prune(cmd *cobra.Command, args []string) error {
	logger.Println("Running prune of local snapshots...")
	if err := pruneLocalSnapshots(); err != nil {
		return err
	}
	logger.Println("Running prune of mirrored snapshots...")
	if err := pruneMirrors(); err != nil {
		return err
	}
	logger.Println("Done.")
	return nil
}

func pruneLocalSnapshots() error {
	for _, vol := range conf.Volumes {
		volumeName := vol.GetName()
		if vol.Disabled {
			if conf.Verbosity >= 1 {
				logger.Printf("Skipping disabled volume %s: %s", volumeName, vol.Path)
			}
			continue
		}
		for _, subvol := range vol.Subvolumes {
			subvolName := subvol.GetName()
			if subvol.Disabled {
				if conf.Verbosity >= 1 {
					logger.Printf("Skipping disabled subvolume %s: %s", subvolName, subvol.Path)
				}
				continue
			}
			logger.Printf("Pruning snapshots for subvolume %s/%s...", vol.Path, subvol.Path)
			snapDir := conf.ResolveSnapshotPath(volumeName, subvolName)
			sourcePath := filepath.Join(vol.Path, subvol.Path)
			manager, err := snapmanager.New(&snapmanager.Config{
				FullSubvolumePath:         sourcePath,
				SnapshotName:              subvol.GetSnapshotName(),
				SnapshotDirectory:         snapDir,
				SnapshotInterval:          conf.ResolveSnapshotInterval(volumeName, subvolName),
				SnapshotMinimumRetention:  conf.ResolveSnapshotMinimumRetention(volumeName, subvolName),
				SnapshotRetention:         conf.ResolveSnapshotRetention(volumeName, subvolName),
				SnapshotRetentionInterval: conf.ResolveSnapshotRetentionInterval(volumeName, subvolName),
				TimeFormat:                conf.ResolveTimeFormat(volumeName, subvolName),
				Logger:                    logger,
				Verbosity:                 conf.Verbosity,
			})
			if err != nil {
				return err
			}
			if err := manager.PruneSnapshots(); err != nil {
				return err
			}
		}
	}
	return nil
}

func pruneMirrors() error {
	for _, vol := range conf.Volumes {
		volumeName := vol.GetName()
		if vol.Disabled {
			if conf.Verbosity >= 1 {
				logger.Printf("Skipping disabled volume %s: %s", volumeName, vol.Path)
			}
			continue
		}
		for _, subvol := range vol.Subvolumes {
			subvolName := subvol.GetName()
			if subvol.Disabled {
				if conf.Verbosity >= 1 {
					logger.Printf("Skipping disabled subvolume %s: %s", subvolName, subvol.Path)
				}
				continue
			}
			mirrors := conf.ResolveMirrors(volumeName, subvolName)
			if len(mirrors) == 0 {
				if conf.Verbosity >= 1 {
					logger.Printf("Skipping subvolume %s/%s: no mirrors configured", vol.Path, subvol.Path)
				}
				continue
			}
			logger.Printf("Pruning mirrors for subvolume %s/%s...", vol.Path, subvol.Path)
			snapDir := conf.ResolveSnapshotPath(volumeName, subvolName)
			sourcePath := filepath.Join(vol.Path, subvol.Path)
			for _, mirror := range mirrors {
				if mirror.Disabled {
					if conf.Verbosity >= 1 {
						logger.Printf("Skipping disabled mirror: %s", mirror.Path)
					}
					continue
				}
				manager, err := syncmanager.New(&syncmanager.Config{
					SubvolumeIdentifier: subvol.GetName(),
					FullSubvolumePath:   sourcePath,
					SnapshotName:        subvol.GetSnapshotName(),
					SnapshotDirectory:   snapDir,
					Logger:              logger,
					Verbosity:           conf.Verbosity,
					MirrorPath:          mirror.Path,
				})
				if err != nil {
					return err
				}
				if err := manager.Prune(context.Background()); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

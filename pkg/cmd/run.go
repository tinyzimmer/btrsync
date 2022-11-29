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
	"time"

	"github.com/spf13/cobra"

	"github.com/tinyzimmer/btrsync/pkg/cmd/queue"
	"github.com/tinyzimmer/btrsync/pkg/cmd/snapmanager"
	"github.com/tinyzimmer/btrsync/pkg/cmd/syncmanager"
)

var (
	runDaemon bool
)

func NewRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run a sync operation based on the configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			if runDaemon {
				return daemon(cmd, args)
			}
			return run(cmd, args)
		},
	}

	cmd.Flags().BoolVarP(&runDaemon, "daemon", "d", false, "Run the daemon process")
	cmd.Flags().Var(&conf.Daemon.ScanInterval, "scan-interval", "The interval to scan for work to do when running as a daemon")
	cmd.Flags().IntVar(&conf.Concurrency, "concurrency", 1, "The number of concurrent sync operations to run")
	v.BindPFlag("daemon.scan_interval", cmd.Flags().Lookup("scan-interval"))
	v.BindPFlag("concurrency", cmd.Flags().Lookup("concurrency"))
	return cmd
}

func run(cmd *cobra.Command, args []string) error {
	logLevel(0, "Running local snapshot operations...")
	if err := handleSnapshots(); err != nil {
		return err
	}
	logLevel(0, "Running sync operations...")
	if err := handleSync(); err != nil {
		return err
	}
	logLevel(0, "Finished sync operations.")
	return nil
}

func daemon(cmd *cobra.Command, args []string) error {
	logLevel(0, "Starting daemon process with %s scan interval...", conf.Daemon.ScanInterval)
	if err := run(cmd, args); err != nil {
		logLevel(0, "Error running sync: %s", err)
	}
	t := time.NewTicker(time.Duration(conf.Daemon.ScanInterval))
	for range t.C {
		if err := run(cmd, args); err != nil {
			logLevel(0, "Error running sync: %s", err)
			logLevel(0, "Will retry on next scan interval")
		}
	}
	return nil
}

func handleSnapshots() error {
	queue := queue.NewConcurrentQueue(queue.WithMaxConcurrency(conf.Concurrency), queue.WithLogger(logger, conf.Verbosity))
	for _, vol := range conf.Volumes {
		volumeName := vol.GetName()
		if vol.Disabled {
			logLevel(1, "Skipping disabled volume %s: %s", volumeName, vol.Path)
			continue
		}
		for _, s := range vol.Subvolumes {
			subvol := s
			queue.Push(func() error {
				subvolName := subvol.GetName()
				if subvol.Disabled {
					logLevel(1, "Skipping disabled subvolume %s: %s", subvolName, subvol.Path)
					return nil
				}
				logLevel(0, "Ensuring snapshots for subvolume %s/%s...", vol.Path, subvol.Path)
				snapDir := conf.ResolveSnapshotPath(volumeName, subvolName)
				sourcePath := filepath.Join(vol.Path, subvol.Path)
				manager, err := snapmanager.New(&snapmanager.Config{
					FullSubvolumePath:         sourcePath,
					SnapshotName:              subvol.GetSnapshotName(volumeName),
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
				if err := manager.EnsureMostRecentSnapshot(); err != nil {
					return err
				}
				if err := manager.PruneSnapshots(); err != nil {
					return err
				}
				return nil
			})
		}
	}
	return queue.Wait()
}

func handleSync() error {
	queue := queue.NewConcurrentQueue(queue.WithMaxConcurrency(conf.Concurrency), queue.WithLogger(logger, conf.Verbosity))
	for _, vol := range conf.Volumes {
		volumeName := vol.GetName()
		if vol.Disabled {
			logLevel(1, "Skipping disabled volume %s: %s", volumeName, vol.Path)
			continue
		}
		for _, subvol := range vol.Subvolumes {
			subvolName := subvol.GetName()
			if subvol.Disabled {
				logLevel(1, "Skipping disabled subvolume %s: %s", subvolName, subvol.Path)
				continue
			}
			mirrors := conf.ResolveMirrors(volumeName, subvolName)
			if len(mirrors) == 0 {
				logLevel(1, "Skipping subvolume %s/%s: no mirrors configured", vol.Path, subvol.Path)
				continue
			}
			logLevel(0, "Running sync for subvolume %s/%s...", vol.Path, subvol.Path)
			snapDir := conf.ResolveSnapshotPath(volumeName, subvolName)
			sourcePath := filepath.Join(vol.Path, subvol.Path)
			for _, m := range mirrors {
				mirror := m
				queue.Push(func() error {
					if mirror.Disabled {
						logLevel(1, "Skipping disabled mirror: %s", mirror.Path)
						return nil
					}
					manager, err := syncmanager.New(&syncmanager.Config{
						Logger:              logger,
						Verbosity:           conf.Verbosity,
						SubvolumeIdentifier: subvol.GetSnapshotName(volumeName),
						FullSubvolumePath:   sourcePath,
						SnapshotDirectory:   snapDir,
						SnapshotName:        subvol.GetSnapshotName(volumeName),
						MirrorPath:          mirror.Path,
						MirrorFormat:        mirror.Format,
						SSHUser:             conf.ResolveMirrorSSHUser(mirror.Name),
						SSHPassword:         conf.ResolveMirrorSSHPassword(mirror.Name),
						SSHKeyFile:          conf.ResolveMirrorSSHKeyFile(mirror.Name),
						SSHHostKey:          conf.ResolveMirrorSSHHostKey(mirror.Name),
					})
					if err != nil {
						return err
					}
					defer manager.Close()
					if err := manager.Sync(context.Background()); err != nil {
						return err
					}
					if err := manager.Prune(context.Background()); err != nil {
						return err
					}
					return nil
				})
			}
		}
	}
	return queue.Wait()
}

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
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/tinyzimmer/btrsync/pkg/cmd/config"
)

var (
	v         = viper.New()
	envPrefix = "BTRSYNC"
	cfgFile   string
	conf      = config.NewDefaultConfig()
	logger    = log.New(os.Stderr, "", log.LstdFlags)
)

func logLevel(level int, format string, args ...interface{}) {
	if conf.Verbosity >= level {
		logger.Printf(format, args...)
	}
}

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
	rootCommand.PersistentFlags().CountVarP(&conf.Verbosity, "verbose", "v", "verbosity level (can be used multiple times)")

	rootCommand.AddCommand(NewRunCommand())
	rootCommand.AddCommand(NewSendCommand())
	rootCommand.AddCommand(NewReceiveCommand())
	rootCommand.AddCommand(NewPruneCommand())
	rootCommand.AddCommand(NewTreeCommand())
	rootCommand.AddCommand(NewMountCommand())
	rootCommand.AddCommand(NewConfigCommand())

	return rootCommand
}
func initConfig(cmd *cobra.Command, args []string) error {
	v.BindPFlag("snapshots_dir", cmd.PersistentFlags().Lookup("snapshots-dir"))
	v.BindPFlag("verbosity", cmd.PersistentFlags().Lookup("verbose"))

	if cfgFile != "" {
		// Use config file from the flag.
		v.SetConfigFile(cfgFile)
	} else {
		cfgdir, err := os.UserConfigDir()
		cobra.CheckErr(err)
		v.AddConfigPath(".")                              // Current directory
		v.AddConfigPath(filepath.Join(cfgdir, "btrsync")) // User config directory
		v.AddConfigPath("/etc/btrsync")                   // System config directory
		v.SetConfigType("toml")
		v.SetConfigName("btrsync.toml")
	}

	if err := v.ReadInConfig(); err == nil {
		if err := v.Unmarshal(&conf, viper.DecodeHook(config.DurationHookFunc())); err != nil {
			return err
		}
		logLevel(1, "Using config file: %s", v.ConfigFileUsed())
	} else {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	v.SetEnvPrefix(envPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		if !f.Changed && v.IsSet(f.Name) {
			cmd.PersistentFlags().SetAnnotation(f.Name, cobra.BashCompOneRequiredFlag, []string{"false"})
			cmd.PersistentFlags().Set(f.Name, v.GetString(f.Name))
		}
	})
	for _, c := range cmd.Commands() {
		c.PersistentFlags().VisitAll(func(f *pflag.Flag) {
			if !f.Changed && v.IsSet(f.Name) {
				cmd.PersistentFlags().SetAnnotation(f.Name, cobra.BashCompOneRequiredFlag, []string{"false"})
				cmd.PersistentFlags().Set(f.Name, v.GetString(f.Name))
			}
		})
		c.Flags().VisitAll(func(f *pflag.Flag) {
			if !f.Changed && v.IsSet(f.Name) {
				cmd.PersistentFlags().SetAnnotation(f.Name, cobra.BashCompOneRequiredFlag, []string{"false"})
				cmd.PersistentFlags().Set(f.Name, v.GetString(f.Name))
			}
		})
	}

	if err := conf.Validate(); err != nil {
		return err
	}

	logLevel(3, "Rendered config: %+v", conf)
	return nil
}

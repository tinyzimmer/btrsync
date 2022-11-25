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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/tinyzimmer/btrsync/cmd/btrsync/cmd/config"
)

func NewConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Print the configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			out, err := json.MarshalIndent(conf, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(out))
			return nil
		},
	}
	return cmd
}

func initConfig(cmd *cobra.Command, args []string) error {
	v := viper.New()

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
		if conf.Verbosity >= 1 {
			logger.Println("Using config file:", v.ConfigFileUsed())
		}
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
			fmt.Println("Setting flag", f.Name, "to", v.Get(f.Name))
			cmd.PersistentFlags().SetAnnotation(f.Name, cobra.BashCompOneRequiredFlag, []string{"false"})
			cmd.PersistentFlags().Set(f.Name, v.GetString(f.Name))
		}
	})

	if err := conf.Validate(); err != nil {
		return err
	}

	if conf.Verbosity >= 3 {
		logger.Printf("Config: %+v\n", conf)
	}

	return nil
}

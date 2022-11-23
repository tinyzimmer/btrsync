package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Config struct {
	SnapshotsDir string   `mapstructure:"snapshots_dir"`
	Verbosity    int      `mapstructure:"verbosity"`
	Volumes      []Volume `mapstructure:"volumes"`
}

type Volume struct {
	Name         string      `mapstructure:"name"`
	Path         string      `mapstructure:"path"`
	SnapshotsDir string      `mapstructure:"snapshots_dir"`
	Subvolumes   []Subvolume `mapstructure:"subvolumes"`
}

type Subvolume struct {
	Name         string `mapstructure:"name"`
	Path         string `mapstructure:"path"`
	SnapshotsDir string `mapstructure:"snapshots_dir"`
	SnapshotName string `mapstructure:"snapshot_name"`
	Disabled     bool   `mapstructure:"disabled"`
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
		if err := v.Unmarshal(&config); err != nil {
			return err
		}
		if config.Verbosity >= 1 {
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

	if config.Verbosity >= 3 {
		logger.Printf("Config: %+v\n", config)
	}

	return nil
}

func (c Config) ResolveSnapshotPath(vol, subvol string) (path string) {
	v := c.GetVolume(vol)
	if v == nil {
		return
	}
	s := v.GetSubvolume(subvol)
	if s == nil {
		return
	}
	if s.SnapshotsDir != "" {
		path = filepath.Join(
			v.Path,
			s.Path,
			s.SnapshotsDir,
		)
	} else if v.SnapshotsDir != "" {
		path = filepath.Join(v.Path, v.SnapshotsDir)
	} else if c.SnapshotsDir != "" {
		path = filepath.Join(v.Path, c.SnapshotsDir)
	} else {
		path = filepath.Join(v.Path, "btrsync_snapshots")
	}
	return
}

func (c Config) GetVolume(name string) *Volume {
	for _, v := range c.Volumes {
		if v.GetName() == name {
			return &v
		}
	}
	return nil
}

func (v Volume) GetSubvolume(name string) *Subvolume {
	for _, s := range v.Subvolumes {
		if s.GetName() == name {
			return &s
		}
	}
	return nil
}

func (v Volume) GetName() string {
	if v.Name != "" {
		return v.Name
	}
	return filepath.Base(v.Path)
}

func (s Subvolume) GetName() string {
	if s.Name != "" {
		return s.Name
	}
	return filepath.Base(s.Path)
}

func (s Subvolume) GetSnapshotName() string {
	if s.SnapshotName != "" {
		return s.SnapshotName
	}
	return s.GetName()
}

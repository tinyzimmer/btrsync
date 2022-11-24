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

package config

import (
	"fmt"
	"path/filepath"
	"time"
)

type Duration time.Duration

func (d Duration) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%q", time.Duration(d).String())), nil
}

// Config is the root configuration object.
type Config struct {
	// Verbosity is the verbosity level.
	Verbosity int `mapstructure:"verbosity"`
	// SnapshotsDir is the directory where snapshots are stored. Defaults to "btrsync_snapshots"
	// on the root of each volume.
	SnapshotsDir string `mapstructure:"snapshots_dir"`
	// SnapshotInterval is the global interval between snapshots.
	SnapshotInterval Duration `mapstructure:"snapshot_interval"`
	// SnapshotMinimumRetention is the global minimum retention time for snapshots.
	SnapshotMinimumRetention Duration `mapstructure:"snapshot_min_retention"`
	// SnapshotRetention is the global retention time for snapshots.
	SnapshotRetention Duration `mapstructure:"snapshot_retention"`
	// SnapshotRetentionInterval is the global interval between snapshot retention runs.
	SnapshotRetentionInterval Duration `mapstructure:"snapshot_retention_interval"`
	// TimeFormat is the global time format for snapshots.
	TimeFormat string `mapstructure:"time_format"`
	// Volumes is a list of volumes to sync.
	Volumes []Volume `mapstructure:"volumes"`
	// Mirrors is a list of mirrors to sync snapshots to.
	Mirrors []Mirror `mapstructure:"mirrors"`
}

// Volume is the global configuration for a btrfs volume.
type Volume struct {
	// Name is a unique identifier for this volume. Defaults to the path.
	Name string `mapstructure:"name"`
	// Path is the mount path of the btrfs volume.
	Path string `mapstructure:"path"`
	// SnapshotsDir is the directory where snapshots are stored for this volume. If left
	// unset the global value is used.
	SnapshotsDir string `mapstructure:"snapshots_dir"`
	// SnapshotInterval is the interval between snapshots for this volume. If left unset
	// the global value is used.
	SnapshotInterval time.Duration `mapstructure:"snapshot_interval"`
	// SnapshotMinimumRetention is the minimum retention time for snapshots for this volume.
	// If left unset the global value is used.
	SnapshotMinimumRetention time.Duration `mapstructure:"snapshot_min_retention"`
	// SnapshotRetention is the retention time for snapshots for this volume. If left unset
	// the global value is used.
	SnapshotRetention time.Duration `mapstructure:"snapshot_retention"`
	// SnapshotRetentionInterval is the interval between snapshot retention runs for this
	// volume. If left unset the global value is used.
	SnapshotRetentionInterval time.Duration `mapstructure:"snapshot_retention_interval"`
	// TimeFormat is the time format for snapshots for this volume. If left unset the global
	// value is used.
	TimeFormat string `mapstructure:"time_format"`
	// Subvolumes is a list of subvolumes to manage.
	Subvolumes []Subvolume `mapstructure:"subvolumes"`
	// Mirrors is a list of mirror names to sync snapshots to.
	Mirrors []string `mapstructure:"mirrors"`
	// Disabled is a flag to disable managing this volume temporarily.
	Disabled bool `mapstructure:"disabled"`
}

// Subvolume is the configuration for a btrfs subvolume.
type Subvolume struct {
	// Name is a unique identifier for this subvolume. Defaults to the path.
	Name string `mapstructure:"name"`
	// Path is the path of the btrfs subvolume, relative to the volume mount.
	Path string `mapstructure:"path"`
	// SnapshotsDir is the directory where snapshots are stored for this subvolume. If left
	// unset either the volume or global value is used respectively.
	SnapshotsDir string `mapstructure:"snapshots_dir"`
	// SnapshotName is the name prefix to give snapshots of this subvolume. Defaults to the
	// subvolume name.
	SnapshotName string `mapstructure:"snapshot_name"`
	// SnapshotInterval is the interval between snapshots for this subvolume. If left unset
	// either the volume or global value is used respectively.
	SnapshotInterval time.Duration `mapstructure:"snapshot_interval"`
	// SnapshotMinimumRetention is the minimum retention time for snapshots for this subvolume.
	// If left unset either the volume or global value is used respectively.
	SnapshotMinimumRetention time.Duration `mapstructure:"snapshot_min_retention"`
	// SnapshotRetention is the retention time for snapshots for this subvolume. If left unset
	// either the volume or global value is used respectively.
	SnapshotRetention time.Duration `mapstructure:"snapshot_retention"`
	// SnapshotRetentionInterval is the interval between snapshot retention runs for this
	// subvolume. If left unset either the volume or global value is used respectively.
	SnapshotRetentionInterval time.Duration `mapstructure:"snapshot_retention_interval"`
	// TimeFormat is the time format for snapshots for this subvolume. If left unset either
	// the volume or global value is used respectively.
	TimeFormat string `mapstructure:"time_format"`
	// Mirrors is a list of mirror names to sync snapshots to. Automatically includes the
	// volume mirrors.
	Mirrors []string `mapstructure:"mirrors"`
	// ExcludeMirrors is a list of mirror names to exclude from syncing snapshots to.
	ExcludeMirrors []string `mapstructure:"exclude_mirrors"`
	// Disabled is a flag to disable managing this subvolume temporarily.
	Disabled bool `mapstructure:"disabled"`
}

// Mirror is the configuration for a btrfs snapshot mirror.
type Mirror struct {
	// Name is a unique identifier for this mirror.
	Name string `mapstructure:"name"`
	// Path is the location of the mirror. Each subvolume mirrored to this mirror will be
	// stored in a subdirectory of this path.
	Path string `mapstructure:"path"`
	// Disabled is a flag to disable managing this mirror temporarily.
	Disabled bool `mapstructure:"disabled"`
}

const (
	DefaultTimeFormat                = "2006-01-02_15-04-05"
	DefaultSnapshotInterval          = Duration(1 * time.Hour)      // Hourly snapshots
	DefaultSnapshotMinimumRetention  = Duration(1 * 24 * time.Hour) // Keep all snapshots at least a day
	DefaultSnapshotRetention         = Duration(7 * 24 * time.Hour) // Retain snapshots for 7 days
	DefaultSnapshotRetentionInterval = Duration(1 * 24 * time.Hour) // One snapshot retained per day
)

func NewDefaultConfig() Config {
	return Config{
		Verbosity:                 0,
		SnapshotInterval:          DefaultSnapshotInterval,
		SnapshotMinimumRetention:  DefaultSnapshotMinimumRetention,
		SnapshotRetention:         DefaultSnapshotRetention,
		SnapshotRetentionInterval: DefaultSnapshotRetentionInterval,
		TimeFormat:                DefaultTimeFormat,
	}
}

func (c Config) Validate() error {
	var volNames []string
	for _, volume := range c.Volumes {
		if !isUnique(volNames, volume.GetName()) {
			return fmt.Errorf("duplicate volume name: %s", volume.GetName())
		}
		if err := volume.Validate(); err != nil {
			return err
		}
		volNames = append(volNames, volume.GetName())
		var subvolNames []string
		for _, subvolume := range volume.Subvolumes {
			if !isUnique(subvolNames, subvolume.GetName()) {
				return fmt.Errorf("volume %s has duplicate subvolume name: %s", volume.GetName(), subvolume.GetName())
			}
			if err := c.ValidateSubvolume(volume, subvolume); err != nil {
				return err
			}
			subvolNames = append(subvolNames, subvolume.GetName())
		}
	}
	return nil
}

func isUnique(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return false
		}
	}
	return true
}

func (v Volume) Validate() error {
	if v.GetName() == "" {
		return fmt.Errorf("volume name or path is required")
	}
	if v.Path == "" {
		return fmt.Errorf("volume path is required")
	}
	return nil
}

func (c Config) ValidateSubvolume(vol Volume, subvol Subvolume) error {
	volName := vol.GetName()
	subvolName := subvol.GetName()
	if subvolName == "" {
		return fmt.Errorf("subvolume name or path is required")
	}
	if subvol.Path == "" {
		return fmt.Errorf("subvolume path is required")
	}
	snapshotDir := c.ResolveSnapshotPath(volName, subvolName)
	snapshotInterval := c.ResolveSnapshotInterval(volName, subvolName)
	snapshotMinRetention := c.ResolveSnapshotMinimumRetention(volName, subvolName)
	snapshotRetention := c.ResolveSnapshotRetention(volName, subvolName)
	snapshotRetentionInterval := c.ResolveSnapshotRetentionInterval(volName, subvolName)

	if snapshotDir == "" {
		return fmt.Errorf("snapshot directory required for subvolume %s:%s", volName, subvolName)
	}

	if snapshotInterval >= snapshotMinRetention {
		return fmt.Errorf("snapshot interval must be less than minimum retention for subvolume %s:%s", volName, subvolName)
	}

	if snapshotRetention <= snapshotMinRetention {
		return fmt.Errorf("snapshot retention must be greater than minimum retention for subvolume %s:%s", volName, subvolName)
	}

	if snapshotRetentionInterval >= snapshotRetention {
		return fmt.Errorf("snapshot retention interval must be less than snapshot retention for subvolume %s:%s", volName, subvolName)
	}

	return nil
}

func (c Config) ResolveTimeFormat(vol, subvol string) (format string) {
	v := c.GetVolume(vol)
	if v == nil {
		return
	}
	s := v.GetSubvolume(subvol)
	if s == nil {
		return
	}
	if s.TimeFormat != "" {
		format = s.TimeFormat
	} else if v.TimeFormat != "" {
		format = v.TimeFormat
	} else if c.TimeFormat != "" {
		format = c.TimeFormat
	} else {
		format = DefaultTimeFormat
	}
	return
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

func (c Config) ResolveSnapshotInterval(vol, subvol string) (interval time.Duration) {
	v := c.GetVolume(vol)
	if v == nil {
		return
	}
	s := v.GetSubvolume(subvol)
	if s == nil {
		return
	}
	if s.SnapshotInterval != 0 {
		interval = s.SnapshotInterval
	} else if v.SnapshotInterval != 0 {
		interval = v.SnapshotInterval
	} else if c.SnapshotInterval != 0 {
		interval = time.Duration(c.SnapshotInterval)
	} else {
		interval = time.Duration(DefaultSnapshotInterval)
	}
	return
}

func (c Config) ResolveSnapshotMinimumRetention(vol, subvol string) (retention time.Duration) {
	v := c.GetVolume(vol)
	if v == nil {
		return
	}
	s := v.GetSubvolume(subvol)
	if s == nil {
		return
	}
	if s.SnapshotMinimumRetention != 0 {
		retention = s.SnapshotMinimumRetention
	} else if v.SnapshotMinimumRetention != 0 {
		retention = v.SnapshotMinimumRetention
	} else if c.SnapshotMinimumRetention != 0 {
		retention = time.Duration(c.SnapshotMinimumRetention)
	} else {
		retention = time.Duration(DefaultSnapshotMinimumRetention)
	}
	return
}

func (c Config) ResolveSnapshotRetention(vol, subvol string) (retention time.Duration) {
	v := c.GetVolume(vol)
	if v == nil {
		return
	}
	s := v.GetSubvolume(subvol)
	if s == nil {
		return
	}
	if s.SnapshotRetention != 0 {
		retention = s.SnapshotRetention
	} else if v.SnapshotRetention != 0 {
		retention = v.SnapshotRetention
	} else if c.SnapshotRetention != 0 {
		retention = time.Duration(c.SnapshotRetention)
	} else {
		retention = time.Duration(DefaultSnapshotRetention)
	}
	return
}

func (c Config) ResolveSnapshotRetentionInterval(vol, subvol string) (interval time.Duration) {
	v := c.GetVolume(vol)
	if v == nil {
		return
	}
	s := v.GetSubvolume(subvol)
	if s == nil {
		return
	}
	if s.SnapshotRetentionInterval != 0 {
		interval = s.SnapshotRetentionInterval
	} else if v.SnapshotRetentionInterval != 0 {
		interval = v.SnapshotRetentionInterval
	} else if c.SnapshotRetentionInterval != 0 {
		interval = time.Duration(c.SnapshotRetentionInterval)
	} else {
		interval = time.Duration(DefaultSnapshotRetentionInterval)
	}
	return
}

func (c Config) ResolveMirrors(vol, subvol string) []Mirror {
	v := c.GetVolume(vol)
	if v == nil {
		return nil
	}
	s := v.GetSubvolume(subvol)
	if s == nil {
		return nil
	}
	var mirrors []Mirror
	for _, mirrorName := range append(v.Mirrors, s.Mirrors...) {
		mirror := c.GetMirror(mirrorName)
		if mirror == nil {
			continue
		}
		mirrors = append(mirrors, *mirror)
	}
	return s.FilterExcludedMirrors(mirrors)
}

func (c Config) GetVolume(name string) *Volume {
	for _, v := range c.Volumes {
		if v.GetName() == name {
			return &v
		}
	}
	return nil
}

func (c Config) GetMirror(name string) *Mirror {
	for _, m := range c.Mirrors {
		if m.Name == name {
			return &m
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

func (s Subvolume) FilterExcludedMirrors(mm []Mirror) []Mirror {
	var mirrors []Mirror
	for _, m := range mm {
		if !s.IsMirrorExcluded(m.Name) {
			mirrors = append(mirrors, m)
		}
	}
	return mirrors
}

func (s Subvolume) IsMirrorExcluded(name string) bool {
	for _, n := range s.ExcludeMirrors {
		if n == name {
			return true
		}
	}
	return false
}

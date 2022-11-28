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
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
)

// Config is the root configuration object.
type Config struct {
	// Verbosity is the verbosity level.
	Verbosity int `mapstructure:"verbosity" toml:"verbosity,omitempty"`
	// Concurrency is the number of concurrent operations to perform. Defaults to 1.
	Concurrency int `mapstructure:"concurrency" toml:"concurrency,omitempty"`
	// SnapshotsDir is the directory where snapshots are stored. Defaults to "btrsync_snapshots"
	// on the root of each volume.
	SnapshotsDir string `mapstructure:"snapshots_dir" toml:"snapshots_dir,omitempty"`
	// SnapshotInterval is the global interval between snapshots.
	SnapshotInterval Duration `mapstructure:"snapshot_interval" toml:"snapshot_interval,omitempty"`
	// SnapshotMinimumRetention is the global minimum retention time for snapshots.
	SnapshotMinimumRetention Duration `mapstructure:"snapshot_min_retention" toml:"snapshot_min_retention,omitempty"`
	// SnapshotRetention is the global retention time for snapshots.
	SnapshotRetention Duration `mapstructure:"snapshot_retention" toml:"snapshot_retention,omitempty"`
	// SnapshotRetentionInterval is the global interval for which snapshots will be retained in
	// the snapshot_retention.
	SnapshotRetentionInterval Duration `mapstructure:"snapshot_retention_interval" toml:"snapshot_retention_interval,omitempty"`
	// TimeFormat is the global time format for snapshots.
	TimeFormat string `mapstructure:"time_format" toml:"time_format,omitempty"`
	// SSHUser is the user to use for SSH connections to this mirror. If left unset, defaults
	// to the current user.
	SSHUser string `mapstructure:"ssh_user" toml:"ssh_user,omitempty"`
	// SSHPassword is the password to use for SSH connections to this mirror. If left unset,
	// or no identity key is provided, passwordless authentication is attempted.
	SSHPassword string `mapstructure:"ssh_password" toml:"ssh_password,omitempty"`
	// SSHKeyIdentityFile is the path to the SSH key identity file to use for SSH connections.
	SSHKeyIdentityFile string `mapstructure:"ssh_key_identity_file" toml:"ssh_key_identity_file,omitempty"`
	// SSHHostKey is the SSH host key to use for SSH connections. If left unset, the host key
	// is not verified.
	SSHHostKey string `mapstructure:"ssh_host_key" toml:"ssh_host_key,omitempty"`
	// Volumes is a list of volumes to sync.
	Volumes []Volume `mapstructure:"volumes" toml:"volumes,omitempty"`
	// Mirrors is a list of mirrors to sync snapshots to.
	Mirrors []Mirror `mapstructure:"mirrors" toml:"mirrors,omitempty"`
	// Daemon configuration
	Daemon DaemonConfig `mapstructure:"daemon" toml:"daemon,omitempty"`
}

// Volume is the global configuration for a btrfs volume.
type Volume struct {
	// Name is a unique identifier for this volume. Defaults to the path.
	Name string `mapstructure:"name" toml:"name,omitempty"`
	// Path is the mount path of the btrfs volume.
	Path string `mapstructure:"path" toml:"path,omitempty"`
	// SnapshotsDir is the directory where snapshots are stored for this volume. If left
	// unset the global value is used.
	SnapshotsDir string `mapstructure:"snapshots_dir" toml:"snapshots_dir,omitempty"`
	// SnapshotInterval is the interval between snapshots for this volume. If left unset
	// the global value is used.
	SnapshotInterval time.Duration `mapstructure:"snapshot_interval" toml:"snapshot_interval,omitempty"`
	// SnapshotMinimumRetention is the minimum retention time for snapshots for this volume.
	// If left unset the global value is used.
	SnapshotMinimumRetention time.Duration `mapstructure:"snapshot_min_retention" toml:"snapshot_min_retention,omitempty"`
	// SnapshotRetention is the retention time for snapshots for this volume. If left unset
	// the global value is used.
	SnapshotRetention time.Duration `mapstructure:"snapshot_retention" toml:"snapshot_retention,omitempty"`
	// SnapshotRetentionInterval is the interval for which snapshots will be retained in
	// the snapshot_retention. If left unset the global value is used.
	SnapshotRetentionInterval time.Duration `mapstructure:"snapshot_retention_interval" toml:"snapshot_retention_interval,omitempty"`
	// TimeFormat is the time format for snapshots for this volume. If left unset the global
	// value is used.
	TimeFormat string `mapstructure:"time_format" toml:"time_format,omitempty"`
	// Subvolumes is a list of subvolumes to manage.
	Subvolumes []Subvolume `mapstructure:"subvolumes" toml:"subvolumes,omitempty"`
	// Mirrors is a list of mirror names to sync snapshots to.
	Mirrors []string `mapstructure:"mirrors" toml:"mirrors,omitempty"`
	// Disabled is a flag to disable managing this volume temporarily.
	Disabled bool `mapstructure:"disabled" toml:"disabled,omitempty"`
}

// Subvolume is the configuration for a btrfs subvolume.
type Subvolume struct {
	// Name is a unique identifier for this subvolume. Defaults to the path.
	Name string `mapstructure:"name" toml:"name,omitempty"`
	// Path is the path of the btrfs subvolume, relative to the volume mount.
	Path string `mapstructure:"path" toml:"path,omitempty"`
	// SnapshotsDir is the directory where snapshots are stored for this subvolume. If left
	// unset either the volume or global value is used respectively.
	SnapshotsDir string `mapstructure:"snapshots_dir" toml:"snapshots_dir,omitempty"`
	// SnapshotName is the name prefix to give snapshots of this subvolume. Defaults to the
	// subvolume name.
	SnapshotName string `mapstructure:"snapshot_name" toml:"snapshot_name,omitempty"`
	// SnapshotInterval is the interval between snapshots for this subvolume. If left unset
	// either the volume or global value is used respectively.
	SnapshotInterval time.Duration `mapstructure:"snapshot_interval" toml:"snapshot_interval,omitempty"`
	// SnapshotMinimumRetention is the minimum retention time for snapshots for this subvolume.
	// If left unset either the volume or global value is used respectively.
	SnapshotMinimumRetention time.Duration `mapstructure:"snapshot_min_retention" toml:"snapshot_min_retention,omitempty"`
	// SnapshotRetention is the retention time for snapshots for this subvolume. If left unset
	// either the volume or global value is used respectively.
	SnapshotRetention time.Duration `mapstructure:"snapshot_retention" toml:"snapshot_retention,omitempty"`
	// SnapshotRetentionInterval is the interval for which snapshots will be retained in
	// the snapshot_retention. If left unset either the volume or global value is used respectively.
	SnapshotRetentionInterval time.Duration `mapstructure:"snapshot_retention_interval" toml:"snapshot_retention_interval,omitempty"`
	// TimeFormat is the time format for snapshots for this subvolume. If left unset either
	// the volume or global value is used respectively.
	TimeFormat string `mapstructure:"time_format" toml:"time_format,omitempty"`
	// Mirrors is a list of mirror names to sync snapshots to. Automatically includes the
	// volume mirrors.
	Mirrors []string `mapstructure:"mirrors" toml:"mirrors,omitempty"`
	// ExcludeMirrors is a list of mirror names to exclude from syncing snapshots to.
	ExcludeMirrors []string `mapstructure:"exclude_mirrors" toml:"exclude_mirrors,omitempty"`
	// Disabled is a flag to disable managing this subvolume temporarily.
	Disabled bool `mapstructure:"disabled" toml:"disabled,omitempty"`
}

// Mirror is the configuration for a btrfs snapshot mirror.
type Mirror struct {
	// Name is a unique identifier for this mirror.
	Name string `mapstructure:"name" toml:"name,omitempty"`
	// Path is the location of the mirror. Each subvolume mirrored to this mirror will be
	// stored in a subdirectory of this path.
	Path string `mapstructure:"path" toml:"path,omitempty"`
	// Format is the format to use for snapshots mirrored to this mirror. If left unset,
	// defaults to "subvolume".
	Format MirrorFormat `mapstructure:"format" toml:"format,omitempty"`
	// SSHUser is the user to use for SSH connections to this mirror. If left unset, the
	// global value is used.
	SSHUser string `mapstructure:"ssh_user" toml:"ssh_user,omitempty"`
	// SSHPassword is the password to use for SSH connections to this mirror. If left unset,
	// the global value is used.
	SSHPassword string `mapstructure:"ssh_password" toml:"ssh_password,omitempty"`
	// SSHKeyIdentityFile is the path to the SSH key identity file to use for mirroring
	// snapshots to this mirror. If left unset, defaults to the global value.
	SSHKeyIdentityFile string `mapstructure:"ssh_key_identity_file" toml:"ssh_key_identity_file,omitempty"`
	// SSHHostKey is the host key to use for SSH connections to this mirror. If left unset,
	// the global value is used.
	SSHHostKey string `mapstructure:"ssh_host_key" toml:"ssh_host_key,omitempty"`
	// Disabled is a flag to disable managing this mirror temporarily.
	Disabled bool `mapstructure:"disabled" toml:"disabled,omitempty"`
}

// DaemonConfig is the configuration for the daemon process
type DaemonConfig struct {
	// ScanInterval is the interval between scans for work to do.
	ScanInterval Duration `mapstructure:"scan_interval" toml:"scan_interval,omitempty"`
}

// MirrorFormat is the format of the mirror path.
type MirrorFormat string

const (
	// MirrorFormatSubvolume is the subvolume format. This is the default
	// format if no format is specified. Only works on Btrfs.
	MirrorFormatSubvolume MirrorFormat = "subvolume"
	// MirrorFormatDirectory is the directory format. This format is
	// compatible with all filesystems, however, it does not support
	// atomic snapshots. The most recent snapshot's contents will be stored
	// in the mirror path and retention settings will be ignored.
	MirrorFormatDirectory MirrorFormat = "directory"
	// // MirrorFormatZfs is the ZFS format. This format is compatible with
	// // ZFS filesystems. ZFS snapshots are used to create atomic snapshots
	// // of the subvolume and are stored in the mirror path.
	// MirrorFormatZfs MirrorFormat = "zfs"
	// MirrorFormatGzip is the gzip format. This format is compatible with all
	// filesystems. Snapshots are sent in stream format to the mirror path and
	// compressed with gzip.
	MirrorFormatGzip MirrorFormat = "gzip"
	// MirrorFormatLzw is the lzw format. This format is compatible with all
	// filesystems. Snapshots are sent in stream format to the mirror path and
	// compressed with lzw.
	MirrorFormatLzw MirrorFormat = "lzw"
	// MirrorFormatZlib is the zlib format. This format is compatible with all
	// filesystems. Snapshots are sent in stream format to the mirror path and
	// compressed with zlib.
	MirrorFormatZlib MirrorFormat = "zlib"
	// MirrorFormatZstd is the zstd format. This format is compatible with all
	// filesystems. Snapshots are sent in stream format to the mirror path and
	// compressed with zstd.
	MirrorFormatZstd MirrorFormat = "zstd"
)

func (m MirrorFormat) IsCompressed() bool {
	switch m {
	case MirrorFormatGzip, MirrorFormatLzw, MirrorFormatZlib, MirrorFormatZstd:
		return true
	}
	return false
}

type Duration time.Duration

func (d *Duration) Type() string {
	return "duration"
}

func (d Duration) String() string { return time.Duration(d).String() }

func (d *Duration) Set(s string) error {
	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(dur)
	return nil
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%q", time.Duration(d).String())), nil
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(dur)
	return nil
}

func (d Duration) MarshalText() ([]byte, error) {
	return []byte(time.Duration(d).String()), nil
}

func (d *Duration) UnmarshalText(b []byte) error {
	dur, err := time.ParseDuration(string(b))
	if err != nil {
		return err
	}
	*d = Duration(dur)
	return nil
}

func DurationHookFunc() mapstructure.DecodeHookFuncType {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{},
	) (interface{}, error) {
		// Check that the data is string
		if f.Kind() != reflect.String {
			return data, nil
		}

		// Check that the target type is our custom type
		if t != reflect.TypeOf(Duration(0)) {
			return data, nil
		}

		// Return the parsed value
		return time.ParseDuration(data.(string))
	}
}

const (
	DefaultSnapshotsDir              = "btrsync_snapshots"
	DefaultTimeFormat                = "2006-01-02_15-04-05"
	DefaultSnapshotInterval          = Duration(1 * time.Hour)      // Hourly snapshots
	DefaultSnapshotMinimumRetention  = Duration(1 * 24 * time.Hour) // Keep all snapshots at least a day
	DefaultSnapshotRetention         = Duration(7 * 24 * time.Hour) // Retain snapshots for 7 days
	DefaultSnapshotRetentionInterval = Duration(1 * 24 * time.Hour) // One snapshot retained per day
	DefaultDaemonScanInterval        = Duration(1 * time.Minute)    // Scan for operations every minute in daemon mode
)

func NewDefaultConfig() Config {
	return Config{
		Verbosity:                 0,
		SnapshotsDir:              DefaultSnapshotsDir,
		SnapshotInterval:          DefaultSnapshotInterval,
		SnapshotMinimumRetention:  DefaultSnapshotMinimumRetention,
		SnapshotRetention:         DefaultSnapshotRetention,
		SnapshotRetentionInterval: DefaultSnapshotRetentionInterval,
		TimeFormat:                DefaultTimeFormat,
		Daemon: DaemonConfig{
			ScanInterval: DefaultDaemonScanInterval,
		},
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

func (c Config) VolumeNameInUse(name string) bool {
	for _, volume := range c.Volumes {
		if volume.GetName() == name {
			return true
		}
	}
	return false
}

func (v Volume) SubvolumeNameInUse(name string) bool {
	for _, subvolume := range v.Subvolumes {
		if subvolume.GetName() == name {
			return true
		}
	}
	return false
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
	} else if strings.Contains(v.GetName(), ":") {
		return fmt.Errorf("volume name cannot contain a colon")
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
	} else if strings.Contains(subvolName, ":") {
		return fmt.Errorf("subvolume name cannot contain colon: %s", subvolName)
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
		path = filepath.Join(v.Path, DefaultSnapshotsDir)
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

func (c Config) ResolveMirrorSSHUser(name string) string {
	mirror := c.GetMirror(name)
	if mirror == nil {
		return ""
	}
	if !strings.HasPrefix(mirror.Path, "ssh://") {
		return ""
	}
	if mirror.SSHUser != "" {
		return mirror.SSHUser
	}
	if c.SSHUser != "" {
		return c.SSHUser
	}
	return ""
}

func (c Config) ResolveMirrorSSHPassword(name string) string {
	mirror := c.GetMirror(name)
	if mirror == nil {
		return ""
	}
	if !strings.HasPrefix(mirror.Path, "ssh://") {
		return ""
	}
	if mirror.SSHPassword != "" {
		return mirror.SSHPassword
	}
	if c.SSHPassword != "" {
		return c.SSHPassword
	}
	return ""
}

func (c Config) ResolveMirrorSSHKeyFile(name string) string {
	mirror := c.GetMirror(name)
	if mirror == nil {
		return ""
	}
	if !strings.HasPrefix(mirror.Path, "ssh://") {
		return ""
	}
	if mirror.SSHKeyIdentityFile != "" {
		return mirror.SSHKeyIdentityFile
	}
	if c.SSHKeyIdentityFile != "" {
		return c.SSHKeyIdentityFile
	}
	return ""
}

func (c Config) ResolveMirrorSSHHostKey(name string) string {
	mirror := c.GetMirror(name)
	if mirror == nil {
		return ""
	}
	if !strings.HasPrefix(mirror.Path, "ssh://") {
		return ""
	}
	if mirror.SSHHostKey != "" {
		return mirror.SSHHostKey
	}
	if c.SSHHostKey != "" {
		return c.SSHHostKey
	}
	return ""
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

func (s Subvolume) GetSnapshotName(volumeName string) string {
	if s.SnapshotName != "" {
		return s.SnapshotName
	}
	return fmt.Sprintf("%s.%s", volumeName, s.GetName())
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

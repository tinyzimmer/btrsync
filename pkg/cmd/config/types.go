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
	"reflect"
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
	// Server configuration
	Server ServerConfig `mapstructure:"server" toml:"server,omitempty"`
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
	// format <volume_name>.<subvolume_name>.
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

// DaemonConfig is the configuration for the daemon process.
type DaemonConfig struct {
	// ScanInterval is the interval between scans for work to do.
	ScanInterval Duration `mapstructure:"scan_interval" toml:"scan_interval,omitempty"`
}

// ServerConfig is the configuration for the server process.
type ServerConfig struct {
	// Protocol is the protocol to use for the server.
	Protocol ServerProto `mapstructure:"protocol" toml:"protocol,omitempty"`
	// ListenAddress is the address to listen on for the server.
	ListenAddress string `mapstructure:"listen_address" toml:"listen_address,omitempty"`
	// ListenPort is the port to listen on for the server.
	ListenPort int `mapstructure:"listen_port" toml:"listen_port,omitempty"`
	// TLSCertFile is the path to the TLS certificate file to use for the server.
	TLSCertFile string `mapstructure:"tls_cert_file" toml:"tls_cert_file,omitempty"`
	// TLSKeyFile is the path to the TLS key file to use for the server.
	TLSKeyFile string `mapstructure:"tls_key_file" toml:"tls_key_file,omitempty"`
	// DataDirectory is the directory to store data in for the server. Defaults to the
	// current working directory.
	DataDirectory string `mapstructure:"data_directory" toml:"data_directory,omitempty"`
}

// ServerProto is the protocol to use for the server.
type ServerProto string

const (
	// ServerProtoHTTP is the HTTP protocol.
	ServerProtoHTTP ServerProto = "http"
	// ServerProtoTCP is the TCP protocol.
	ServerProtoTCP ServerProto = "tcp"
	// ServerProtoUnix is the Unix protocol.
	ServerProtoUnix ServerProto = "unix"
	// ServerProtoUnixPacket is the Unix packet protocol.
	ServerProtoUnixPacket ServerProto = "unixpacket"
	// ServerProtoUDP is the UDP protocol.
	ServerProtoUDP ServerProto = "udp"
)

func (p *ServerProto) Type() string {
	return "string"
}

func (p ServerProto) String() string { return string(p) }

func (p *ServerProto) Set(s string) error {
	proto := ServerProto(s)
	p = &proto
	switch proto {
	case ServerProtoHTTP, ServerProtoTCP, ServerProtoUnix, ServerProtoUnixPacket, ServerProtoUDP:
		return nil
	default:
		return fmt.Errorf("invalid protocol: %s", s)
	}
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

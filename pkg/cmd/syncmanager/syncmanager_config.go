package syncmanager

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"os/user"

	"github.com/tinyzimmer/btrsync/pkg/cmd/config"
	"golang.org/x/crypto/ssh"
)

type Config struct {
	Logger              *log.Logger
	Verbosity           int
	SubvolumeIdentifier string
	FullSubvolumePath   string
	SnapshotDirectory   string
	SnapshotName        string
	MirrorPath          string
	MirrorFormat        config.MirrorFormat
	SSHUser             string
	SSHPassword         string
	SSHKeyFile          string
	SSHHostKey          string
}

func (c *Config) LogVerbose(level int, format string, args ...interface{}) {
	if c.Verbosity >= level {
		c.Logger.Printf(format, args...)
	}
}

func (c *Config) MirrorURL() (*url.URL, error) {
	u, err := url.Parse(c.MirrorPath)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "" {
		u.Scheme = "file"
	}
	return u, nil
}

func (c *Config) SSHConfig() (*ssh.ClientConfig, error) {
	mirrorURL, err := c.MirrorURL()
	if err != nil {
		return nil, err
	}
	usr := c.SSHUser
	if mirrorURL.User != nil {
		usr = mirrorURL.User.Username()
	}
	if usr == "" {
		cur, err := user.Current()
		if err != nil {
			return nil, err
		}
		usr = cur.Username
	}
	cfg := ssh.ClientConfig{User: usr}
	if c.SSHPassword != "" {
		cfg.Auth = append(cfg.Auth, ssh.Password(c.SSHPassword))
	}
	if c.SSHKeyFile != "" {
		data, err := os.ReadFile(c.SSHKeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read ssh key file: %s", err)
		}
		signer, err := ssh.ParsePrivateKey(data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse ssh key file: %s", err)
		}
		cfg.Auth = append(cfg.Auth, ssh.PublicKeys(signer))
	}
	if c.SSHHostKey != "" {
		key, err := ssh.ParsePublicKey([]byte(c.SSHHostKey))
		if err != nil {
			return nil, fmt.Errorf("failed to parse ssh host key: %s", err)
		}
		cfg.HostKeyCallback = ssh.FixedHostKey(key)
	} else {
		cfg.HostKeyCallback = ssh.InsecureIgnoreHostKey()
	}
	return &cfg, nil
}

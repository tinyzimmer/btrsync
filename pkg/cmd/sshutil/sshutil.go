package sshutil

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"

	"golang.org/x/crypto/ssh"
)

func Dial(ctx context.Context, url *url.URL, config *ssh.ClientConfig) (*ssh.Client, error) {
	var addr string
	if url.Port() != "" {
		addr = fmt.Sprintf("%s:%s", url.Hostname(), url.Port())
	} else {
		addr = fmt.Sprintf("%s:22", url.Hostname())
	}
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	c, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		return nil, err
	}
	return ssh.NewClient(c, chans, reqs), nil
}

func CommandExists(ctx context.Context, client *ssh.Client, cmd string) (bool, error) {
	sess, err := client.NewSession()
	if err != nil {
		return false, err
	}
	defer sess.Close()
	out, err := sess.CombinedOutput(fmt.Sprintf("command -v %q", cmd))
	if err != nil {
		if strings.Contains(err.Error(), "status 1") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check %s: %s: %w", cmd, string(out), err)
	}
	return len(out) > 0, nil
}

func IsFileNotExist(err error) bool {
	return strings.Contains(err.Error(), "No such file or directory")
}

func FileOrDirectoryExists(ctx context.Context, client *ssh.Client, path string) (exists bool, err error) {
	sess, err := client.NewSession()
	if err != nil {
		return false, err
	}
	defer sess.Close()
	out, err := sess.CombinedOutput(fmt.Sprintf("test -e %q && echo -n true || echo -n false", path))
	if err != nil {
		return false, fmt.Errorf("failed to check %s: %s: %w", path, string(out), err)
	}
	return string(out) == "true", nil
}

func ReadFile(ctx context.Context, client *ssh.Client, path string) ([]byte, error) {
	sess, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	defer sess.Close()
	out, err := sess.CombinedOutput(fmt.Sprintf("cat %q", path))
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %s: %w", path, string(out), err)
	}
	return out, nil
}

func ReadDir(ctx context.Context, client *ssh.Client, path string) ([]string, error) {
	sess, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	defer sess.Close()
	out, err := sess.CombinedOutput(fmt.Sprintf("ls -1 %q", path))
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %s: %w", path, string(out), err)
	}
	return strings.Fields(string(out)), nil
}

func RemoveFile(ctx context.Context, client *ssh.Client, path string) error {
	sess, err := client.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()
	out, err := sess.CombinedOutput(fmt.Sprintf("rm -f %q", path))
	if err != nil {
		return fmt.Errorf("failed to remove %s: %s: %w", path, string(out), err)
	}
	return nil
}

func MkdirAll(ctx context.Context, client *ssh.Client, path string) error {
	sess, err := client.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()
	out, err := sess.CombinedOutput(fmt.Sprintf("mkdir -p %q", path))
	if err != nil {
		return fmt.Errorf("failed to create %s: %s: %w", path, string(out), err)
	}
	return nil
}

func WriteFile(ctx context.Context, client *ssh.Client, path string, rdr io.Reader) error {
	sess, err := client.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()
	ip, err := sess.StdinPipe()
	if err != nil {
		return err
	}
	errs := make(chan error, 1)
	go func() {
		defer ip.Close()
		_, err := io.Copy(ip, rdr)
		errs <- err
	}()
	out, err := sess.CombinedOutput(fmt.Sprintf("dd of=%q", path))
	if err != nil {
		return fmt.Errorf("failed to write %s: %s: %w", path, string(out), err)
	}
	return <-errs
}

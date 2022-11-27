package syncmanager

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/tinyzimmer/btrsync/pkg/btrfs"
	"github.com/tinyzimmer/btrsync/pkg/cmd/snaputil"
	"github.com/tinyzimmer/btrsync/pkg/cmd/sshutil"
	"golang.org/x/crypto/ssh"
)

type sshSubvolumeManager struct {
	config     *Config
	sourceInfo *btrfs.RootInfo
	mirrorURL  *url.URL
	sshClient  *ssh.Client
}

func NewSSHSubvolumeManager(cfg *Config, subvolInfo *btrfs.RootInfo) (Manager, error) {
	mirrorURL, err := cfg.MirrorURL()
	if err != nil {
		return nil, err
	}
	cfg.LogVerbose(0, "Initiating SSH directory sync manager for %q with mirror URL: %s\n",
		cfg.FullSubvolumePath, mirrorURL.String())

	sshcfg, err := cfg.SSHConfig()
	if err != nil {
		return nil, err
	}
	var addr string
	if mirrorURL.Port() != "" {
		addr = fmt.Sprintf("%s:%s", mirrorURL.Hostname(), mirrorURL.Port())
	} else {
		addr = fmt.Sprintf("%s:22", mirrorURL.Hostname())
	}
	cfg.LogVerbose(1, "Connecting to remote host using tcp: %s\n", addr)
	sshClient, err := ssh.Dial("tcp", addr, sshcfg)
	if err != nil {
		return nil, fmt.Errorf("failed to dial ssh server: %s", err)
	}
	return &sshSubvolumeManager{
		config:     cfg,
		sourceInfo: subvolInfo,
		mirrorURL:  mirrorURL,
		sshClient:  sshClient,
	}, nil
}

func (sm *sshSubvolumeManager) Sync(ctx context.Context) error {
	exists, err := sshutil.CommandExists(ctx, sm.sshClient, "btrsync")
	if err != nil {
		return err
	}
	var syncFunc func(context.Context, *btrfs.RootInfo, *btrfs.RootInfo) error
	if exists {
		sm.config.LogVerbose(0, "Remote host has btrsync installed, using btrfsync for sync\n")
		syncFunc = sm.syncBtrsync
	} else {
		sm.config.LogVerbose(0, "Remote host does not have btrsync installed, using btrfs send/receive for sync\n")
		syncFunc = sm.syncBtrfs
	}
	// Make sure the top directory exists on the path
	parentdir := filepath.Dir(sm.getRemoteSnapshotPath(sm.sourceInfo))
	if err := sshutil.MkdirAll(ctx, sm.sshClient, parentdir); err != nil {
		return err
	}
	snapshots := snaputil.MapParents(sm.sourceInfo.Snapshots)
	for _, snap := range snapshots {
		if err := syncFunc(ctx, snap.Parent, snap.Snapshot); err != nil {
			return err
		}
	}
	return nil
}

func (sm *sshSubvolumeManager) Prune(ctx context.Context) error {
	sm.config.LogVerbose(0, "Pruning expired snapshots from mirror: %s\n", sm.config.MirrorPath)
	remoteSnapshots, err := sm.listRemoteSnapshots(ctx)
	if err != nil {
		return err
	}
	sm.config.LogVerbose(3, "Found %d remote snapshots\n", len(remoteSnapshots))
	sm.config.LogVerbose(4, "Remote snapshots: %v\n", remoteSnapshots)
	// Check remote against what we have locally
	for snapshotName, uuid := range remoteSnapshots {
		if !snaputil.SnapshotUUIDExists(sm.sourceInfo.Snapshots, uuid) {
			sess, err := sm.sshClient.NewSession()
			if err != nil {
				return err
			}
			defer sess.Close()
			sm.config.LogVerbose(0, "Pruning remote snapshot %q\n", snapshotName)
			fullpath := filepath.Join(sm.mirrorURL.Path, sm.config.SubvolumeIdentifier, snapshotName)
			cmd := fmt.Sprintf("btrfs subvolume delete %s", fullpath)
			sm.config.LogVerbose(1, "Running command: %s\n", cmd)
			out, err := sess.CombinedOutput(cmd)
			if err != nil {
				return fmt.Errorf("failed to delete remote snapshot: %s: %w", string(out), err)
			}
		}
	}
	return nil
}

func (sm *sshSubvolumeManager) Close() error {
	return sm.sshClient.Close()
}

func (sm *sshSubvolumeManager) syncBtrfs(ctx context.Context, parent, snap *btrfs.RootInfo) error {
	synced, err := sm.isRemoteSnapshotSynced(ctx, snap)
	if err != nil {
		return fmt.Errorf("failed to check if remote snapshot is synced: %s", err)
	}
	if synced {
		sm.config.LogVerbose(1, "Remote snapshot %q is already synced, skipping\n", snap.Path)
		return nil
	}

	// Double check if the directory exists and remove if so (this should be cleaned up)
	exists, err := sshutil.FileOrDirectoryExists(ctx, sm.sshClient, sm.getRemoteSnapshotPath(snap))
	if err != nil {
		return err
	}
	if exists {
		sess, err := sm.sshClient.NewSession()
		if err != nil {
			return fmt.Errorf("failed to create ssh session: %s", err)
		}
		out, err := sess.CombinedOutput(fmt.Sprintf("btrfs subvol del %s", sm.getRemoteSnapshotPath(snap)))
		if err != nil {
			return fmt.Errorf("failed to delete remote snapshot: %s: %s", err, out)
		}
	}
	sm.config.LogVerbose(0, "Syncing snapshot %q to remote host\n", snap.Path)

	sess, err := sm.sshClient.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()
	sessStdin, err := sess.StdinPipe()
	if err != nil {
		return err
	}

	if sm.config.Verbosity >= 3 {
		stderr, err := sess.StderrPipe()
		if err != nil {
			return err
		}
		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				sm.config.LogVerbose(3, "SSH REMOTE: %s\n", scanner.Text())
			}
		}()
	}

	// Start the send operation

	pipeOpt, pipe, err := btrfs.SendToPipe()
	if err != nil {
		return fmt.Errorf("error creating send pipe: %w", err)
	}

	var wg sync.WaitGroup
	errors := make(chan error, 2)

	wg.Add(1)
	go func() {
		defer wg.Done()
		sendOpts := []btrfs.SendOption{
			pipeOpt,
			btrfs.SendWithLogger(sm.config.Logger, sm.config.Verbosity),
			btrfs.SendCompressedData(),
		}
		if parent != nil {
			sendOpts = append(sendOpts, btrfs.SendWithParentRoot(sm.getLocalSnapshotPath(parent)))
		}
		if err := btrfs.Send(sm.getLocalSnapshotPath(snap), sendOpts...); err != nil {
			err = fmt.Errorf("error sending snapshot: %w", err)
			errors <- err
		}
	}()

	// Copy the send data to the stdin pipe

	wg.Add(1)
	go func() {
		defer wg.Done()
		sm.config.LogVerbose(4, "Copying send data to remote host\n")
		_, err := io.Copy(sessStdin, pipe)
		if err != nil {
			err = fmt.Errorf("error copying send data to remote: %w", err)
			errors <- err
		}
		sm.config.LogVerbose(4, "Finished copying send data to remote host")
	}()

	// Start a receive on the remote end
	cmd := "btrfs receive"
	if sm.config.Verbosity >= 3 {
		cmd = "btrfs receive -v"
	}
	cmd = fmt.Sprintf("%s -e %s", cmd, filepath.Dir(sm.getRemoteSnapshotPath(snap)))
	err = sess.Run(cmd)
	if err != nil {
		err = fmt.Errorf("error running btrfs receive: %w", err)
		return err
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			return err
		}
	}
	sm.config.LogVerbose(0, "Finished syncing snapshot %q to remote host", snap.Path)
	return nil
}

func (sm *sshSubvolumeManager) syncBtrsync(ctx context.Context, parent, snap *btrfs.RootInfo) error {
	sm.config.LogVerbose(0, "Btrsync is not yet supported for SSH subvolume manager, falling back to btrfs\n")
	return sm.syncBtrfs(ctx, parent, snap)
}

func (sm *sshSubvolumeManager) getLocalSnapshotPath(snap *btrfs.RootInfo) string {
	return filepath.Join(sm.config.SnapshotDirectory, snap.Path)
}

func (sm *sshSubvolumeManager) getRemoteSnapshotPath(snap *btrfs.RootInfo) string {
	return filepath.Join(sm.mirrorURL.Path, sm.config.SubvolumeIdentifier, snap.Path)
}

func (sm *sshSubvolumeManager) isRemoteSnapshotSynced(ctx context.Context, snap *btrfs.RootInfo) (bool, error) {
	sm.config.LogVerbose(2, "Checking if remote snapshot %q is synced\n", snap.Path)
	exists, err := sshutil.FileOrDirectoryExists(ctx, sm.sshClient, sm.getRemoteSnapshotPath(snap))
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}
	sm.config.LogVerbose(3, "Remote snapshot %q exists, checking if received UUID matches local\n", snap.Path)
	receivedUUID, err := sm.getRemoteReceivedUUID(ctx, snap)
	if err != nil {
		return false, err
	}
	sm.config.LogVerbose(3, "Remote snapshot %q has received UUID %q\n", snap.Path, receivedUUID)
	return receivedUUID == snap.UUID, nil
}

func (sm *sshSubvolumeManager) getRemoteReceivedUUID(ctx context.Context, snap *btrfs.RootInfo) (uuid.UUID, error) {
	sess, err := sm.sshClient.NewSession()
	if err != nil {
		return uuid.Nil, err
	}
	defer sess.Close()
	cmd := fmt.Sprintf("btrfs subvolume show %s | grep 'Received UUID' | awk '{print $3}'", sm.getRemoteSnapshotPath(snap))
	sm.config.LogVerbose(4, "Running command %q on remote host\n", cmd)
	out, err := sess.CombinedOutput(cmd)
	if err != nil {
		return uuid.Nil, fmt.Errorf("error running command on remote host: %s: %w", string(out), err)
	}
	data := strings.TrimSpace(string(out))
	if string(data) == "-" {
		return uuid.Nil, nil
	}
	return uuid.Parse(string(data))
}

func (sm *sshSubvolumeManager) listRemoteSnapshots(ctx context.Context) (map[string]uuid.UUID, error) {
	parentdir := filepath.Dir(sm.getRemoteSnapshotPath(sm.sourceInfo))
	sess, err := sm.sshClient.NewSession()
	if err != nil {
		return nil, err
	}
	cmd := fmt.Sprintf("btrfs subvol list -osR %q", parentdir)
	sm.config.LogVerbose(4, "Running command %q on remote host\n", cmd)
	out, err := sess.CombinedOutput(cmd)
	if err != nil {
		return nil, fmt.Errorf("error listing remote snapshots: %s: %w", string(out), err)
	}
	snapshots := make(map[string]uuid.UUID)
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 16 {
			return nil, fmt.Errorf("unexpected output from btrfs subvol list: %s", line)
		}
		name := filepath.Base(parts[15])
		uustr := parts[13]
		uuid, err := uuid.Parse(uustr)
		if err != nil {
			return nil, fmt.Errorf("error parsing UUID %q: %w", uustr, err)
		}
		snapshots[name] = uuid
	}
	return snapshots, nil
}

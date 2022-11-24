package receive

import (
	"fmt"
	"io"
	"sync"

	"github.com/tinyzimmer/btrsync/pkg/btrfs"
	"github.com/tinyzimmer/btrsync/pkg/sendstream"
)

// FindPathDiffOffset is the same as FindDiffOffset but will initiate the stream
// from the given paths. This is useful for ensuring that the source paths are
// read-only before attempting to find the diff offset and that the recommended
// flags are passed to the send stream.
func FindPathDiffOffset(pathA, pathB string, pathAParent string, pathBParent string) (offset uint64, err error) {
	// Make sure subvolumes are read-only
	aIsReadOnly, err := btrfs.IsSubvolumeReadOnly(pathA)
	if err != nil {
		return 0, err
	}
	bIsReadOnly, err := btrfs.IsSubvolumeReadOnly(pathB)
	if err != nil {
		return 0, err
	}
	if !aIsReadOnly {
		if err := btrfs.SetSubvolumeReadOnly(pathA, true); err != nil {
			return 0, err
		}
		defer btrfs.SetSubvolumeReadOnly(pathA, false)
	}
	if !bIsReadOnly {
		if err := btrfs.SetSubvolumeReadOnly(pathB, true); err != nil {
			return 0, err
		}
		defer btrfs.SetSubvolumeReadOnly(pathB, false)
	}

	// Create pipes for the send streams
	aStreamOpt, aStream, err := btrfs.SendToPipe()
	if err != nil {
		return 0, err
	}
	bStreamOpt, bStream, err := btrfs.SendToPipe()
	if err != nil {
		return 0, err
	}

	// Send the streams
	var wg sync.WaitGroup
	errCh := make(chan error, 2)
	wg.Add(2)

	go func() {
		defer wg.Done()
		opts := []btrfs.SendOption{aStreamOpt, btrfs.SendWithoutData()}
		if pathAParent != "" {
			opts = append(opts, btrfs.SendWithParentRoot(pathAParent))
		}
		errCh <- btrfs.Send(pathA, opts...)
	}()

	go func() {
		defer wg.Done()
		opts := []btrfs.SendOption{bStreamOpt, btrfs.SendWithoutData()}
		if pathBParent != "" {
			opts = append(opts, btrfs.SendWithParentRoot(pathBParent))
		}
		errCh <- btrfs.Send(pathB, opts...)
	}()

	// Find the offset between the streams
	offset, err = FindDiffOffset(aStream, bStream)
	if err != nil {
		return 0, err
	}

	// Wait and report any errors
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			return 0, err
		}
	}
	return offset, nil
}

// FindDiffOffset will find the offset in the stream where the diff between the
// two streams begins. This is useful for determining where to resume a stream
// that has been interrupted. It is recommended to pass streams sent with the
// SendWithoutData flag to this function.
func FindDiffOffset(streamA, streamB io.Reader) (offset uint64, err error) {
	scanA := sendstream.NewScanner(streamA, false)
	scanB := sendstream.NewScanner(streamB, false)
	for scanA.Scan() {
		if !scanB.Scan() {
			if err = scanB.Err(); err != nil {
				return
			}
			return
		}
		aCmd, _ := scanA.Command()
		bCmd, _ := scanB.Command()
		fmt.Println(aCmd, bCmd)
		// Subvol and snapshot commands will not be exact since UUIDs will be different
		if aCmd.Cmd == bCmd.Cmd && (aCmd.Cmd == sendstream.BTRFS_SEND_C_SUBVOL || aCmd.Cmd == sendstream.BTRFS_SEND_C_SNAPSHOT) {
			offset++
			continue
		}
		if aCmd.Cmd != bCmd.Cmd || aCmd.Crc != bCmd.Crc || aCmd.Len != bCmd.Len {
			return
		}
		offset++
	}
	if err = scanA.Err(); err != nil {
		return
	}
	return
}

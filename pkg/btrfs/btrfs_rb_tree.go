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

package btrfs

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// RBRoot represents the root of the Btrfs tree.
type RBRoot struct {
	RBNode *RBNode
}

// RBNode represents a node in the Btrfs tree.
type RBNode struct {
	RBLeft  *RBNode
	RBRight *RBNode
	Info    *RootInfo
}

// RootInfo represents the information about a Btrfs root.
// RBNode contains a reference to the node in the tree.
type RootInfo struct {
	RBNode *RBNode

	RootID             ObjectID
	RootOffset         uint64
	Flags              uint64
	RefTree            ObjectID
	DirID              uint64
	TopID              uint64
	Generation         uint64
	OriginalGeneration uint64
	CreationTime       time.Time
	SendTime           time.Time
	ReceiveTime        time.Time
	UUID               uuid.UUID
	ParentUUID         uuid.UUID
	ReceivedUUID       uuid.UUID
	Path               string
	Name               string

	// Only populated by resolving the path while building a tree.
	FullPath string
	Deleted  bool

	// Only populated during SubvolumeSearch with SearchWithSnapshots
	Snapshots []*RootInfo

	// The underlying item and reference that built this info
	Item *BtrfsRootItem
	Ref  *BtrfsRootRef
}

func newRBRoot() *RBRoot { return &RBRoot{} }

func (r *RBRoot) InsertRoot(info *RootInfo) {
	if r.RBNode == nil {
		r.RBNode = info.RBNode
		return
	}
	r.insertRoot(r.RBNode, info)
}

func (r *RBRoot) insertRoot(node *RBNode, info *RootInfo) {
	if info.RootID < node.Info.RootID {
		if node.RBLeft == nil {
			node.RBLeft = info.RBNode
			return
		}
		r.insertRoot(node.RBLeft, info)
	}
	if node.RBRight == nil {
		node.RBRight = info.RBNode
		return
	}
	r.insertRoot(node.RBRight, info)
}

func (r *RBRoot) LookupRoot(rootID ObjectID) *RootInfo {
	if r.RBNode == nil {
		return nil
	}
	return r.lookupRoot(r.RBNode, rootID)
}

func (r *RBRoot) lookupRoot(node *RBNode, rootID ObjectID) *RootInfo {
	if node.Info.RootID == rootID {
		return node.Info
	}
	if rootID < node.Info.RootID {
		if node.RBLeft == nil {
			return nil
		}
		return r.lookupRoot(node.RBLeft, rootID)
	}
	if node.RBRight == nil {
		return nil
	}
	return r.lookupRoot(node.RBRight, rootID)
}

func (r *RBRoot) UpdateRoot(info *RootInfo) bool {
	toUpdate := r.LookupRoot(info.RootID)
	if toUpdate == nil {
		return false
	}
	if info.RootOffset != 0 {
		toUpdate.RootOffset = info.RootOffset
	}
	if info.Flags != 0 {
		toUpdate.Flags = info.Flags
	}
	if info.RefTree != 0 {
		toUpdate.RefTree = info.RefTree
	}
	if info.DirID != 0 {
		toUpdate.DirID = info.DirID
	}
	if info.TopID != 0 {
		toUpdate.TopID = info.TopID
	}
	if info.Generation != 0 {
		toUpdate.Generation = info.Generation
	}
	if info.OriginalGeneration != 0 {
		toUpdate.OriginalGeneration = info.OriginalGeneration
	}
	if !info.CreationTime.IsZero() {
		toUpdate.CreationTime = info.CreationTime
	}
	if !info.SendTime.IsZero() {
		toUpdate.SendTime = info.SendTime
	}
	if !info.ReceiveTime.IsZero() {
		toUpdate.ReceiveTime = info.ReceiveTime
	}
	if info.UUID != uuid.Nil {
		toUpdate.UUID = info.UUID
	}
	if info.ParentUUID != uuid.Nil {
		toUpdate.ParentUUID = info.ParentUUID
	}
	if info.ReceivedUUID != uuid.Nil {
		toUpdate.ReceivedUUID = info.ReceivedUUID
	}
	if info.Path != "" {
		toUpdate.Path = info.Path
	}
	if info.Name != "" {
		toUpdate.Name = info.Name
	}
	if info.FullPath != "" {
		toUpdate.FullPath = info.FullPath
	}
	if info.Deleted {
		toUpdate.Deleted = info.Deleted
	}
	if info.Item != nil {
		toUpdate.Item = info.Item
	}
	if info.Ref != nil {
		toUpdate.Ref = info.Ref
	}
	return true
}

// RBTreeIterFunc is the function signature for the RBTreeIterFunc.
// Lasterr is the last error returned by the function. If the function returns
// an ErrStopTreeIteration error, the iteration will stop and the error will be
// returned by RBTree.Iterate.
type RBTreeIterFunc func(info *RootInfo, lastErr error) error

var ErrStopTreeIteration = fmt.Errorf("stop tree iteration")

func (r *RBRoot) PreOrderIterate(f RBTreeIterFunc) error {
	if r.RBNode == nil {
		return nil
	}
	return r.preOrderIterate(r.RBNode, f, nil)
}

func (r *RBRoot) preOrderIterate(node *RBNode, f RBTreeIterFunc, lastErr error) error {
	lastErr = f(node.Info, lastErr)
	if lastErr != nil && errors.Is(lastErr, ErrStopTreeIteration) {
		return lastErr
	}
	if node.RBLeft != nil {
		lastErr = r.preOrderIterate(node.RBLeft, f, lastErr)
		if lastErr != nil && errors.Is(lastErr, ErrStopTreeIteration) {
			return lastErr
		}
	}
	if node.RBRight != nil {
		lastErr = r.preOrderIterate(node.RBRight, f, lastErr)
		if lastErr != nil && errors.Is(lastErr, ErrStopTreeIteration) {
			return lastErr
		}
	}
	return lastErr
}

func (r *RBRoot) PostOrderIterate(f RBTreeIterFunc) error {
	if r.RBNode == nil {
		return nil
	}
	return r.postOrderIterate(r.RBNode, f, nil)
}

func (r *RBRoot) postOrderIterate(node *RBNode, f RBTreeIterFunc, lastErr error) error {
	if node.RBLeft != nil {
		lastErr = r.postOrderIterate(node.RBLeft, f, lastErr)
		if lastErr != nil && errors.Is(lastErr, ErrStopTreeIteration) {
			return lastErr
		}
	}
	if node.RBRight != nil {
		lastErr = r.postOrderIterate(node.RBRight, f, lastErr)
		if lastErr != nil && errors.Is(lastErr, ErrStopTreeIteration) {
			return lastErr
		}
	}
	return f(node.Info, lastErr)
}

func (r *RBRoot) InOrderIterate(f RBTreeIterFunc) error {
	if r.RBNode == nil {
		return nil
	}
	return r.inOrderIterate(r.RBNode, f, nil)
}

func (r *RBRoot) inOrderIterate(node *RBNode, f RBTreeIterFunc, lastErr error) error {
	if node.RBLeft != nil {
		lastErr = r.inOrderIterate(node.RBLeft, f, lastErr)
		if lastErr != nil && errors.Is(lastErr, ErrStopTreeIteration) {
			return lastErr
		}
	}
	lastErr = f(node.Info, lastErr)
	if lastErr != nil && errors.Is(lastErr, ErrStopTreeIteration) {
		return lastErr
	}
	if node.RBRight != nil {
		lastErr = r.inOrderIterate(node.RBRight, f, lastErr)
		if lastErr != nil && errors.Is(lastErr, ErrStopTreeIteration) {
			return lastErr
		}
	}
	return lastErr
}

func (r *RBRoot) resolveFullPaths(rootFd uintptr, topID uint64) error {
	return r.PreOrderIterate(func(info *RootInfo, lastErr error) error {
		if lastErr != nil {
			return lastErr
		}
		info.Deleted = info.RefTree == 0
		if info.Path != "" || info.RefTree == 0 {
			return nil
		}
		// Lookup path relative to the parent subvolume
		var path string
		path, err := lookupInoPath(rootFd, info)
		if err != nil {
			return err
		}
		info.Path = path
		// Resolve full path up to root mount
		fullpath := path
		next := info.RefTree
		for uint64(next) != topID && next != FSTreeObjectID {
			found := r.LookupRoot(next)
			if found == nil {
				break
			}
			fullpath = found.Name + "/" + fullpath
			next = found.RefTree
		}
		info.FullPath = fullpath
		return nil
	})
}

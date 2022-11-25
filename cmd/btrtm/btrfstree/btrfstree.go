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

package btrfstree

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/tinyzimmer/btrsync/cmd/btrsync/cmd/config"
	"github.com/tinyzimmer/btrsync/pkg/btrfs"
)

var logger = log.New(os.Stderr, "", log.LstdFlags)

type BtrfsTree struct {
	*widget.Tree

	config    *config.Config
	snapshots map[string]*btrfs.RootInfo
}

func New(conf *config.Config) (*BtrfsTree, error) {
	t := &BtrfsTree{
		config:    conf,
		snapshots: make(map[string]*btrfs.RootInfo),
	}
	t.Tree = widget.NewTree(t.ChildUIDs, t.IsBranch, t.CreateNode, t.UpdateNode)
	return t, nil
}

func (b *BtrfsTree) ChildUIDs(uid widget.TreeNodeID) []widget.TreeNodeID {
	if uid == "" {
		uids := make([]widget.TreeNodeID, 0, len(b.config.Volumes))
		for _, vol := range b.config.Volumes {
			uids = append(uids, widget.TreeNodeID(vol.GetName()))
		}
		return uids
	}
	volume := b.config.GetVolume(string(uid))
	if volume != nil {
		uids := make([]widget.TreeNodeID, 0, len(volume.Subvolumes))
		for _, subvol := range volume.Subvolumes {
			id := fmt.Sprintf("%s:%s", volume.GetName(), subvol.GetName())
			uids = append(uids, widget.TreeNodeID(id))
		}
		return uids
	}
	// Will be a string of <volname:subvolname> from the isVolume branch,
	// we need to start iterating snapshots.
	spl := strings.Split(string(uid), ":")
	if len(spl) != 2 {
		return nil
	}
	volname, subvolname := spl[0], spl[1]
	volume = b.config.GetVolume(volname)
	if volume == nil {
		logger.Printf("Could not lookup volume name %s", volname)
		return nil
	}
	subvol := volume.GetSubvolume(subvolname)
	if subvol == nil {
		logger.Printf("Could not lookup subvolume name %s from volume %s", subvolname, volname)
		return nil
	}
	info, err := btrfs.SubvolumeSearch(
		btrfs.SearchWithRootMount(volume.Path),
		btrfs.SearchWithPath(filepath.Join(volume.Path, subvol.Path)),
		btrfs.SearchWithSnapshots(),
	)
	if err != nil {
		logger.Printf("Error looking up subvolume snapshots: %s", err)
		return nil
	}
	uids := make([]widget.TreeNodeID, 0)
	// We'll start using the format <volname:subvolname:snapuuid>
	for _, snap := range info.Snapshots {
		if snap.Deleted || snap.FullPath == "" {
			continue
		}
		s := snap
		b.snapshots[snap.UUID.String()] = s
		id := fmt.Sprintf("%s:%s:%s", volume.GetName(), subvol.GetName(), snap.UUID)
		uids = append(uids, id)
	}
	return uids
}

func (b *BtrfsTree) CreateNode(isBranch bool) fyne.CanvasObject {
	if isBranch {
		return widget.NewLabel("")
	}
	return container.NewHBox(
		widget.NewLabel(""),
		layout.NewSpacer(),
		widget.NewButtonWithIcon("Browse", theme.FolderIcon(), func() {}),
		widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {}),
	)
}

func (b *BtrfsTree) UpdateNode(uid widget.TreeNodeID, isBranch bool, node fyne.CanvasObject) {
	if !isBranch {
		spl := strings.Split(string(uid), ":")
		volname, subvolname, uid := spl[0], spl[1], spl[2]
		info := b.snapshots[uid]
		container := node.(*fyne.Container)
		for _, item := range container.Objects {
			snapshotDir := b.config.ResolveSnapshotPath(volname, subvolname)
			snapshotPath := filepath.Join(snapshotDir, info.Path)
			// Handle the label
			if label, ok := item.(*widget.Label); ok {
				label.SetText(info.CreationTime.Format(time.RFC1123))
			}
			// Handle the browse button
			if button, ok := item.(*widget.Button); ok && button.Text == "Browse" {
				button.OnTapped = func() {
					logger.Println("Setting file dialog to", snapshotPath)
					window := fyne.CurrentApp().Driver().AllWindows()[0]
					cb := func(fyne.URIReadCloser, error) {
						// call back to restore file (open a new dialog for where to save)
					}
					f := dialog.NewFileOpen(cb, window)
					f.Resize(fyne.NewSize(700, 500))
					f.SetConfirmText("Restore")
					f.SetFilter(&fileFilter{snapshotPath})
					lister, err := storage.ListerForURI(storage.NewFileURI(snapshotPath))
					if err == nil {
						f.SetLocation(lister)
					}
					f.Show()
				}
			}
			// Handle the delete button
			if button, ok := item.(*widget.Button); ok && button.Text == "Delete" {
				button.OnTapped = func() {
					if err := btrfs.DeleteSubvolume(snapshotPath); err != nil {
						logger.Println("Error deleting snapshot:", err)
					}
					b.Tree.Refresh()
				}
			}
		}
		return
	}
	spl := strings.Split(string(uid), ":")
	volume := b.config.GetVolume(spl[0])
	var text string
	if len(spl) == 1 {
		text = volume.Path
	}
	if len(spl) == 2 {
		subvol := volume.GetSubvolume(spl[1])
		text = subvol.Path
	}
	node.(*widget.Label).SetText(string(text))
}

func (b *BtrfsTree) IsBranch(uid widget.TreeNodeID) bool {
	if uid == "" {
		return true
	}
	spl := strings.Split(string(uid), ":")
	return len(spl) < 3
}

type fileFilter struct {
	rootPath string
}

func (f *fileFilter) Matches(uri fyne.URI) bool {
	return strings.HasPrefix(uri.Path(), f.rootPath)
}

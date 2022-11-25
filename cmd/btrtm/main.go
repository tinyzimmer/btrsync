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

// Btrtm is an app written with the Fyne toolkit for browsing and managing btrfs snapshots.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/spf13/viper"

	"github.com/tinyzimmer/btrsync/cmd/btrsync/cmd/config"
	"github.com/tinyzimmer/btrsync/cmd/btrtm/btrfstree"
)

func main() {
	app, err := New()
	if err != nil {
		panic(err)
	}
	app.ShowAndRun()
}

func New() (fyne.Window, error) {
	a := app.New()
	w := a.NewWindow("Btrfs Time Machine")
	w.Resize(fyne.NewSize(800, 600))

	conf, err := initConfig()
	if err != nil {
		w.SetContent(container.NewMax(container.NewCenter(container.NewVBox(
			widget.NewIcon(theme.WarningIcon()),
			widget.NewLabel(fmt.Sprintf("There was an error loading the configuration: %s", err)),
		))))
		return w, nil
	}

	tree, err := btrfstree.New(conf)
	if err != nil {
		return nil, err
	}
	label := canvas.NewText("Managed Volumes", theme.ForegroundColor())
	label.TextStyle = fyne.TextStyle{Bold: true}
	w.SetContent(container.New(layout.NewPaddedLayout(), container.NewBorder(
		label,                       // Top
		nil,                         // Bottom
		nil,                         // Left
		nil,                         // Right
		container.NewMax(tree.Tree), // Middle
	)))

	return w, nil
}

func initConfig() (*config.Config, error) {
	v := viper.New()

	cfgdir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	v.AddConfigPath(".")                              // Current directory
	v.AddConfigPath(filepath.Join(cfgdir, "btrsync")) // User config directory
	v.AddConfigPath("/etc/btrsync")                   // System config directory
	v.SetConfigType("toml")
	v.SetConfigName("btrsync.toml")

	var conf config.Config
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}
	if err := v.Unmarshal(&conf); err != nil {
		return nil, err
	}

	if err := conf.Validate(); err != nil {
		return nil, err
	}

	return &conf, err
}

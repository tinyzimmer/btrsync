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

package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"

	"github.com/tinyzimmer/btrsync/cmd/btrsync/cmd/config"
	"github.com/tinyzimmer/btrsync/pkg/btrfs"
)

var (
	generateMounts       []string
	generateIncludeExprs []string
	generateExcludeExprs []string
)

func NewConfigCommand() *cobra.Command {
	root := &cobra.Command{
		Use:   "config",
		Short: "Work with btrsync configuration files",
	}

	test := &cobra.Command{
		Use:   "test",
		Short: "Test a configuration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "Configuration file is valid: %q\n", v.ConfigFileUsed())
			return nil
		},
	}

	show := &cobra.Command{
		Use:   "show",
		Short: "Print the configuration",
		RunE:  showConfig,
	}

	generate := &cobra.Command{
		Use:     "generate",
		Short:   "Generate a new configuration file",
		Aliases: []string{"gen"},
		RunE:    generateConfig,
	}

	generate.Flags().StringArrayVarP(&generateMounts, "mount", "m", []string{}, "Mount points to include in the configuration (defaults to all detected btrfs mounts)")
	generate.Flags().StringArrayVarP(&generateIncludeExprs, "include", "i", []string{},
		"Include expressions to apply to subvolume paths while generating the configuration (default match all)")
	generate.Flags().StringArrayVarP(&generateExcludeExprs, "exclude", "e", []string{},
		"Exclude expressions to apply to subvolume paths while generating the configuration (default exclude none)")

	root.AddCommand(test)
	root.AddCommand(show)
	root.AddCommand(generate)

	return root
}

func showConfig(cmd *cobra.Command, args []string) error {
	out, err := json.MarshalIndent(conf, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

func generateConfig(cmd *cobra.Command, args []string) error {
	var includeExprs, excludeExprs []*regexp.Regexp

	for _, expr := range generateIncludeExprs {
		re, err := regexp.Compile(expr)
		if err != nil {
			return fmt.Errorf("failed to compile include expression: %s", err)
		}
		includeExprs = append(includeExprs, re)
	}

	for _, expr := range generateExcludeExprs {
		re, err := regexp.Compile(expr)
		if err != nil {
			return fmt.Errorf("failed to compile exclude expression: %s", err)
		}
		excludeExprs = append(excludeExprs, re)
	}

	var err error
	conf := config.NewDefaultConfig()
	if len(generateMounts) == 0 {
		generateMounts, err = btrfs.ListBtrfsMounts()
		if err != nil {
			return err
		}
	}
	// Populate volumes
	for _, mount := range generateMounts {
		var name string
		if mount == "/" {
			name = "root"
		} else {
			name = filepath.Base(mount)
		}
		if conf.VolumeNameInUse(name) {
			name = fmt.Sprintf("%s-%d", name, time.Now().Unix())
		}
		volume := config.Volume{
			Name:       name,
			Path:       mount,
			Subvolumes: make([]config.Subvolume, 0),
		}
		// Populate subvolumes starting with one for the root of the mount
		tree, err := btrfs.BuildRBTree(mount)
		if err != nil {
			return err
		}
		rootName := fmt.Sprintf("%s-root", name)
		subvols := []config.Subvolume{
			{
				Name: fmt.Sprintf("%s-root", name),
			},
		}
		// Make sure we are including the root
		if len(includeExprs) > 0 {
			for _, expr := range includeExprs {
				if expr.MatchString(rootName) {
					break
				}
				subvols = nil
			}
		}
		// Make sure we are not excluding the root
		for _, expr := range excludeExprs {
			if expr.MatchString(rootName) {
				subvols = nil
				break
			}
		}
		// Populate the rest of the subvolumes
		tree.InOrderIterate(func(info *btrfs.RootInfo, _ error) error {
			if info.Deleted || info.Path == "" {
				return nil
			}
			if info.ParentUUID == uuid.Nil {
				var fullpath = info.Path
				parent := tree.LookupRoot(info.RefTree)
				for parent != nil {
					fullpath = filepath.Join(parent.Path, fullpath)
					parent = tree.LookupRoot(parent.RefTree)
				}

				// Check if we should include this subvolume
				if len(includeExprs) > 0 {
					var matched bool
					for _, expr := range includeExprs {
						if expr.MatchString(fullpath) {
							matched = true
							break
						}
					}
					if !matched {
						return nil
					}
				}

				// Check if we should exclude this subvolume
				if len(excludeExprs) > 0 {
					for _, expr := range excludeExprs {
						if expr.MatchString(fullpath) {
							return nil
						}
					}
				}

				name := filepath.Base(fullpath)
				if volume.SubvolumeNameInUse(name) {
					name = fmt.Sprintf("%s-%d", name, time.Now().Unix())
				}
				subvols = append(subvols, config.Subvolume{
					Name: name,
					Path: fullpath,
				})
			}
			return nil
		})
		volume.Subvolumes = subvols
		if len(volume.Subvolumes) > 0 {
			conf.Volumes = append(conf.Volumes, volume)
		}
	}

	var buf bytes.Buffer
	err = toml.NewEncoder(&buf).SetIndentSymbol("    ").Encode(conf)
	if err != nil {
		return err
	}

	var indentedBuf strings.Builder
	scanner := bufio.NewScanner(&buf)
	var indent int
	for scanner.Scan() {
		text := scanner.Text()
		if text == "" {
			indent = 0
		}
		if strings.HasPrefix(text, "[[volumes.subvolumes") {
			indent++
		}
		indentedBuf.WriteString(strings.Repeat("    ", indent))
		indentedBuf.WriteString(text + "\n")
	}

	fmt.Println(indentedBuf.String())
	return nil
}

package main

import "github.com/tinyzimmer/btrsync/cmd/btrsync/cmd"

var version string

func main() {
	cmd.Execute(version)
}

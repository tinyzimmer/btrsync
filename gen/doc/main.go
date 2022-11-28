package main

import (
	"log"

	"github.com/spf13/cobra/doc"
	"github.com/tinyzimmer/btrsync/pkg/cmd"
)

func main() {
	err := doc.GenMarkdownTree(cmd.NewRootCommand(""), "docs/")
	if err != nil {
		log.Fatal(err)
	}
}

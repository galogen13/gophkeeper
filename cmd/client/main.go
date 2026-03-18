package main

import (
	"fmt"
	"os"

	"github.com/galogen13/gophkeeper/internal/buildinfo"
	"github.com/galogen13/gophkeeper/internal/client/commands"
)

var (
	buildVersion = buildinfo.BuildInfoNotAvaluable
	buildDate    = buildinfo.BuildInfoNotAvaluable
)

func main() {
	if err := commands.NewRootCmd(buildVersion, buildDate).Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

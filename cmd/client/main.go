package main

import (
	"fmt"
	"os"

	"github.com/galogen13/gophkeeper/internal/client/commands"
)

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
)

func main() {
	if err := commands.NewRootCmd(buildVersion, buildDate).Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

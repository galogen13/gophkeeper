package main

import (
	"log"

	"github.com/galogen13/gophkeeper/internal/buildinfo"
	"github.com/galogen13/gophkeeper/internal/client/commands"
)

var (
	buildVersion = buildinfo.BuildInfoNotAvaluable
	buildDate    = buildinfo.BuildInfoNotAvaluable
)

// func main() {
// 	if err := commands.NewRootCmd(buildVersion, buildDate).Execute(); err != nil {
// 		log.Fatal(err)
// 	}
// }

func main() {
	if err := commands.RunInteractiveMode(buildVersion, buildDate); err != nil {
		log.Fatal(err)
	}
}

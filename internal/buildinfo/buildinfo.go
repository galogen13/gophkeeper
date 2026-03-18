package buildinfo

import (
	"fmt"
)

const BuildInfoNotAvaluable = "N/A"

func PrintBuildInfo(buildVersion, buildDate string) {

	buildInfo := `
Build version: %s
Build date: %s
`
	fmt.Printf(buildInfo, buildVersion, buildDate)

}

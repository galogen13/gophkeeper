package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewVersionCmd(version, date string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("GophKeeper Client\n")
			fmt.Printf("Version: %s\n", version)
			fmt.Printf("Build date: %s\n", date)
		},
	}
}

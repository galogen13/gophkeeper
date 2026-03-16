package commands

import (
	"github.com/spf13/cobra"
)

func NewRootCmd(version, date string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "gophkeeper",
		Short: "GophKeeper - secure password manager",
		Long:  `GophKeeper is a client-server system for securely storing passwords and private data.`,
	}

	// Добавляем подкоманды
	rootCmd.AddCommand(
		// NewAuthCmd(),
		// NewSecretsCmd(),
		// NewSyncCmd(),
		NewVersionCmd(version, date),
	)

	return rootCmd
}

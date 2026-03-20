package commands

import (
	"github.com/spf13/cobra"
)

func NewSecretsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secrets",
		Short: "Manage secrets",
		Long: `Manage your secrets: passwords, bank cards, text notes, and binary files.
You can create, list, view, update, and delete secrets.`,
	}

	// Добавляем подкоманды
	cmd.AddCommand(
		NewSecretsListCmd(),
		NewSecretsCreateCmd(),
		NewSecretsGetCmd(),
		NewSecretsUpdateCmd(),
		NewSecretsDeleteCmd(),
	)

	return cmd
}

package commands

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func NewSecretsListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all secrets",
		RunE: func(cmd *cobra.Command, args []string) error {
			return listSecrets()
		},
	}

	// Добавляем флаги
	cmd.Flags().BoolP("all", "a", false, "Include deleted secrets")
	cmd.Flags().StringP("type", "t", "", "Filter by type (credentials, text, binary, bank_card)")

	return cmd
}

func listSecrets() error {
	ctx := context.Background()

	// Проверяем аутентификацию
	auth, err := store.GetAuth(ctx)
	if err != nil {
		return fmt.Errorf("failed to get auth: %w", err)
	}
	if auth == nil {
		return fmt.Errorf("not logged in. Please run 'gophkeeper auth login' first")
	}

	// Получаем все секреты из локального хранилища
	secrets, err := store.ListActiveSecrets(ctx)
	if err != nil {
		return fmt.Errorf("failed to list secrets: %w", err)
	}

	if len(secrets) == 0 {
		fmt.Println("No secrets found. Create one with 'gophkeeper secrets create'")
		return nil
	}

	// Выводим в виде таблицы
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTITLE\tTYPE\tCREATED")
	fmt.Fprintln(w, "--\t-----\t----\t-------")

	for _, s := range secrets {
		created := s.CreatedAt.Format("2006-01-02 15:04")
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			truncate(s.ID, 36),
			truncate(s.Title, 20),
			s.Type.String(),
			created,
		)
	}
	w.Flush()

	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

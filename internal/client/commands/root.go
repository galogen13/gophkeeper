package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/galogen13/gophkeeper/internal/client"
	"github.com/galogen13/gophkeeper/internal/client/grpc"
	"github.com/galogen13/gophkeeper/internal/client/storage/sqlite"
)

var (
	cfgFile string

	grpcClient *grpc.Client
	store      client.Storage
)

func NewRootCmd(version, date string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "gophkeeper",
		Short: "GophKeeper - secure password manager",
		Long: `GophKeeper is a client-server system for securely storing 
passwords, bank cards, and other private data.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initializeClient()
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			return closeClient()
		},
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "../../configs/client.yaml", "config file (default is $HOME/.gophkeeper/config.yaml)")

	// Подкоманды
	rootCmd.AddCommand(NewAuthCmd())
	rootCmd.AddCommand(NewSecretsCmd())
	rootCmd.AddCommand(NewSyncCmd())
	rootCmd.AddCommand(NewVersionCmd(version, date))

	return rootCmd
}

func initializeClient() error {
	if err := initConfig(); err != nil {
		return err
	}

	// Создаём gRPC клиент
	client, err := grpc.NewClient(grpc.Config{
		ServerAddress: viper.GetString("server.address"),
		EnableTLS:     viper.GetBool("server.tls_enabled"),
		Timeout:       viper.GetDuration("server.timeout"),
	})
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	grpcClient = client

	// Создаём локальное хранилище
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	dbPath := filepath.Join(home, ".gophkeeper", "data.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0700); err != nil {
		return err
	}

	sqliteStore, err := sqlite.NewStorage(sqlite.Config{
		Path: dbPath,
	})
	if err != nil {
		return fmt.Errorf("failed to create storage: %w", err)
	}

	store = sqliteStore

	// Загружаем токен, если есть
	if err := loadToken(); err != nil {
		return err
	}

	return nil
}

func loadToken() error {
	ctx := context.Background()
	auth, err := store.GetAuth(ctx)
	if err != nil {
		return err
	}
	if auth != nil && !auth.IsExpired() {
		grpcClient.SetToken(auth.AccessToken)
	}
	return nil
}

func closeClient() error {
	if grpcClient != nil {
		grpcClient.Close()
	}
	if store != nil {
		store.Close()
	}
	return nil
}

func initConfig() error {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		configDir := filepath.Join(home, ".gophkeeper")
		viper.AddConfigPath(configDir)
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")

		// Создаём конфиг по умолчанию, если его нет
		if err := createDefaultConfig(configDir); err != nil {
			return err
		}
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	return nil
}

func createDefaultConfig(configDir string) error {
	defaultConfig := `server:
  address: "localhost:8080"
  tls_enabled: false
  timeout: 10

log:
  level: "info"
`
	configPath := filepath.Join(configDir, "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return os.WriteFile(configPath, []byte(defaultConfig), 0600)
	}
	return nil
}

package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/galogen13/gophkeeper/internal/client"
	"github.com/galogen13/gophkeeper/internal/client/grpc"
	"github.com/galogen13/gophkeeper/internal/client/storage"
	"github.com/spf13/cobra"
)

func NewSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Synchronize data with server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return syncData()
		},
	}

	return cmd
}

func syncData() error {
	ctx := context.Background()

	// Проверяем аутентификацию
	auth, err := store.GetAuth(ctx)
	if err != nil {
		return fmt.Errorf("failed to get auth: %w", err)
	}
	if auth == nil {
		return fmt.Errorf("not logged in")
	}

	fmt.Println("Synchronizing with server...")

	// Получаем время последней синхронизации
	lastSync, _ := store.GetLastSyncTime(ctx)

	// Создаём контекст с токеном
	authCtx := grpcClient.WithAuth(ctx)

	syncInfo := &client.SyncInfo{LastSync: lastSync}

	// Запрашиваем изменения
	resp, err := grpcClient.GetKeeperClient().SyncSecrets(authCtx, grpc.SyncInfoToProtoSyncRequest(syncInfo))
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	// Сохраняем новые/обновлённые секреты
	secrets := make([]*client.Secret, 0, len(resp.GetSecrets()))
	for _, pbSecret := range resp.GetSecrets() {
		secret := grpc.SecretFromProtoSecret(pbSecret)
		secrets = append(secrets, secret)
	}
	if err := store.SaveSecrets(ctx, secrets); err != nil {
		return fmt.Errorf("failed to save secrets during synchronization: %w", err)
	}

	// Помечаем удалённые
	for _, id := range resp.GetDeletedIds() {
		if err := store.DeleteSecret(ctx, id); err != nil {
			// Игнорируем ошибки, если секрет уже удалён локально
			if err != storage.ErrNotFound {
				return fmt.Errorf("failed to delete secret %s: %w", id, err)
			}
		}
	}

	// Обновляем время синхронизации
	if err := store.SetLastSyncTime(ctx, time.Now()); err != nil {
		return fmt.Errorf("failed to save sync time: %w", err)
	}

	fmt.Printf("Sync completed: %d secrets, %d deleted\n", len(resp.GetSecrets()), len(resp.GetDeletedIds()))

	return nil
}

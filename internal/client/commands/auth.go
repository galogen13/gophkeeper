package commands

import (
	"context"
	"fmt"
	"time"

	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/galogen13/gophkeeper/internal/client"
	"github.com/galogen13/gophkeeper/internal/client/grpc"
)

func NewAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication commands",
	}

	cmd.AddCommand(
		NewRegisterCmd(),
		NewLoginCmd(),
		NewLogoutCmd(),
		NewStatusCmd(),
	)

	return cmd
}

func NewRegisterCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "register",
		Short: "Register a new account",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Print("Enter email: ")
			var email string
			fmt.Scanln(&email)

			fmt.Print("Enter password: ")
			password, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			registerInfo := &client.RegisterLoginInfo{
				Email:    email,
				Password: string(password),
			}

			resp, err := grpcClient.GetAuthClient().
				Register(ctx, grpc.RegisterInfoToProtoRegisterRequest(registerInfo))
			if err != nil {
				return fmt.Errorf("registration failed: %w", err)
			}

			fmt.Println("Registration successful!")

			// Сервер не возвращает user_id при регистрации, но он нам и не особо нужен
			auth := grpc.AuthFromProtoAuth(resp, "")

			if err := store.SaveAuth(ctx, auth); err != nil {
				return fmt.Errorf("failed to save auth: %w", err)
			}

			grpcClient.SetToken(auth.AccessToken)

			return nil
		},
	}
}

func NewLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Login to existing account",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Print("Enter email: ")
			var email string
			fmt.Scanln(&email)

			fmt.Print("Enter password: ")
			password, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			loginInfo := &client.RegisterLoginInfo{
				Email:    email,
				Password: string(password),
			}

			resp, err := grpcClient.GetAuthClient().Login(ctx, grpc.RegisterInfoToProtoLoginRequest(loginInfo))
			if err != nil {
				return fmt.Errorf("login failed: %w", err)
			}

			fmt.Println("Login successful!")

			auth := grpc.AuthFromProtoAuth(resp, "")

			if err := store.SaveAuth(ctx, auth); err != nil {
				return fmt.Errorf("failed to save auth: %w", err)
			}

			grpcClient.SetToken(auth.AccessToken)

			return nil
		},
	}
}

func NewLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Logout and clear local data",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			if err := store.ClearAuth(ctx); err != nil {
				return err
			}

			grpcClient.SetToken("")
			fmt.Println("Logged out")

			return nil
		},
	}
}

func NewStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show authentication status",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			auth, err := store.GetAuth(ctx)
			if err != nil {
				return err
			}

			if auth == nil {
				fmt.Println("Not logged in")
				return nil
			}

			if auth.IsExpired() {
				fmt.Println("Logged in but token expired (run 'gophkeeper auth login' to refresh)")
			} else {
				fmt.Println("Logged in")
				expiresIn := time.Until(auth.SavedAt.Add(time.Duration(auth.ExpiresIn) * time.Second))
				fmt.Printf("Token expires in: %v\n", expiresIn.Round(time.Second))
			}

			return nil
		},
	}
}

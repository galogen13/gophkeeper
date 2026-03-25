package grpc

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/galogen13/gophkeeper/internal/pkg/proto"
)

type Client struct {
	conn         *grpc.ClientConn
	authClient   proto.AuthServiceClient
	keeperClient proto.KeeperServiceClient
	token        string
}

type Config struct {
	ServerAddress string
	EnableTLS     bool
	Timeout       time.Duration
}

func NewClient(cfg Config) (*Client, error) {
	var opts []grpc.DialOption

	// Настройка TLS
	if cfg.EnableTLS {
		creds := credentials.NewClientTLSFromCert(nil, "")
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Таймаут по умолчанию
	opts = append(opts, grpc.WithConnectParams(grpc.ConnectParams{
		MinConnectTimeout: cfg.Timeout * time.Second,
	}))

	conn, err := grpc.NewClient(cfg.ServerAddress, opts...)
	if err != nil {
		return nil, fmt.Errorf("gRPC client init failure: %w", err)
	}

	return &Client{
		conn:         conn,
		authClient:   proto.NewAuthServiceClient(conn),
		keeperClient: proto.NewKeeperServiceClient(conn),
	}, nil
}

// SetToken устанавливает токен для последующих запросов
func (c *Client) SetToken(token string) {
	c.token = token
}

// GetAuthClient возвращает клиент аутентификации
func (c *Client) GetAuthClient() proto.AuthServiceClient {
	return c.authClient
}

// GetKeeperClient возвращает клиент для работы с секретами
func (c *Client) GetKeeperClient() proto.KeeperServiceClient {
	return c.keeperClient
}

// Close закрывает соединение
func (c *Client) Close() error {
	return c.conn.Close()
}

// WithAuth добавляет токен в контекст для аутентифицированных запросов
func (c *Client) WithAuth(ctx context.Context) context.Context {
	if c.token == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+c.token)
}

// WithTimeout создаёт контекст с таймаутом
func (c *Client) WithTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, timeout)
}

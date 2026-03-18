package config

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Crypto   CryptoConfig   `mapstructure:"crypto"`
	Log      LogConfig      `mapstructure:"log"`
}

type ServerConfig struct {
	Address string `mapstructure:"address"`
	Host    string `mapstructure:"host"`
	Port    int    `mapstructure:"port"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

func (c DatabaseConfig) ConnString() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

type JWTConfig struct {
	Secret          string        `mapstructure:"secret"`
	AccessTokenTTL  time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"`
}

type CryptoConfig struct {
	Argon2 struct {
		Time    uint32 `mapstructure:"time"`
		Memory  uint32 `mapstructure:"memory"`
		Threads uint8  `mapstructure:"threads"`
		KeyLen  uint32 `mapstructure:"key_len"`
	} `mapstructure:"argon2"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
}

func Load(path string) (*Config, error) {
	viper.SetConfigFile(path)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	cfg.Server.Address = net.JoinHostPort(cfg.Server.Host, strconv.Itoa(cfg.Server.Port))

	return &cfg, nil
}

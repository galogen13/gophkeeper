package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
)

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
)

func main() {
	// Вывод версии при запуске
	log.Printf("Starting GophKeeper Server")
	log.Printf("Version: %s", buildVersion)
	log.Printf("Build date: %s", buildDate)

	// Парсинг флагов
	configPath := flag.String("config", "configs/server.yaml", "path to config file")
	flag.Parse()

	// TODO: Здесь будет инициализация приложения
	log.Printf("Config path: %s", *configPath)
	log.Printf("Server starting...")

	connStr := "postgres://gophkeeper:gophkeeper@localhost:5435/gophkeeper?sslmode=disable&client_encoding=UTF8&lc_messages=C"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Database ping failed:", err)
	}

	log.Println("Successfully connected to database!")

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	<-ctx.Done()
	log.Println("Shutting down server...")
	time.Sleep(2 * time.Second)
	log.Println("Server stopped")
}

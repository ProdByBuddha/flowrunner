package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/tcmartin/flowrunner/pkg/storage"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database migrations and readiness checks",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Postgres migration
		if err := migratePostgres(); err != nil {
			return err
		}
		// Redis readiness
		if err := checkRedis(); err != nil {
			return err
		}
		fmt.Println("Migrations and checks completed successfully")
		return nil
	},
}

func init() {
	// attach to root in main.go via rootCmd.AddCommand in main()
	// We cannot modify rootCmd here, so expose a function used by main to attach
}

func attachMigrate(root *cobra.Command) {
	root.AddCommand(migrateCmd)
}

func migratePostgres() error {
	host := getenvDefault("FLOWRUNNER_POSTGRES_HOST", getenvDefault("POSTGRES_HOST", "localhost"))
	user := getenvDefault("FLOWRUNNER_POSTGRES_USER", getenvDefault("POSTGRES_USER", "flowrunner"))
	password := getenvDefault("FLOWRUNNER_POSTGRES_PASSWORD", getenvDefault("POSTGRES_PASSWORD", "flowrunner"))
	dbName := getenvDefault("FLOWRUNNER_POSTGRES_DATABASE", getenvDefault("POSTGRES_DB", "flowrunner_db"))
	portStr := getenvDefault("FLOWRUNNER_POSTGRES_PORT", "5432")
	sslMode := getenvDefault("FLOWRUNNER_POSTGRES_SSL_MODE", "disable")
	port, _ := strconv.Atoi(portStr)

	cfg := storage.PostgreSQLProviderConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		Database: dbName,
		SSLMode:  sslMode,
	}
	provider, err := storage.NewPostgreSQLProvider(cfg)
	if err != nil {
		return fmt.Errorf("postgres connect failed: %w", err)
	}
	defer provider.Close()
	if err := provider.Initialize(); err != nil {
		return fmt.Errorf("postgres initialize failed: %w", err)
	}
	fmt.Printf("Postgres migrated (host=%s db=%s)\n", host, dbName)
	return nil
}

func checkRedis() error {
	host := getenvDefault("REDIS_HOST", "localhost")
	port := getenvDefault("REDIS_PORT", "6379")
	addr := net.JoinHostPort(host, port)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("redis connect failed: %w", err)
	}
	defer conn.Close()
	// Send PING in RESP: *1\r\n$4\r\nPING\r\n
	if _, err := conn.Write([]byte("*1\r\n$4\r\nPING\r\n")); err != nil {
		return fmt.Errorf("redis ping write failed: %w", err)
	}
	buf := make([]byte, 16)
	n, err := conn.Read(buf)
	if err != nil {
		return fmt.Errorf("redis ping read failed: %w", err)
	}
	resp := string(buf[:n])
	if len(resp) == 0 || resp[0] != '+' { // expect +PONG\r\n
		return fmt.Errorf("redis unexpected response: %q", resp)
	}
	fmt.Printf("Redis ready (host=%s port=%s)\n", host, port)
	return nil
}

func getenvDefault(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

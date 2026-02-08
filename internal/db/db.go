package db

import (
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // Register the Postgres driver
)

// DB wraps the sqlx.DB connection
type DB struct {
	*sqlx.DB
}

// NewConnection creates a new database connection
func NewConnection() (*DB, error) {
	// We get connection details from environment variables (12-Factor App methodology)
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		getEnv("DB_HOST", "localhost"),
		getEnv("DB_PORT", "5432"),
		getEnv("DB_USER", "user"),
		getEnv("DB_PASSWORD", "password"),
		getEnv("DB_NAME", "codedrop"),
	)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to db: %w", err)
	}

	// Connection Pool Settings
	// SetMaxOpenConns: Max number of open connections to the database.
	// SetMaxIdleConns: Max number of connections in the idle connection pool.
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	log.Println("Successfully connected to Postgres!")
	return &DB{db}, nil
}

// Migrate applies the schema to the database
func (d *DB) Migrate() error {
	schema, err := os.ReadFile("internal/db/schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	_, err = d.Exec(string(schema))
	if err != nil {
		return fmt.Errorf("failed to apply schema: %w", err)
	}

	log.Println("Database schema applied successfully!")
	return nil
}

// Helper to get env vars with a fallback
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
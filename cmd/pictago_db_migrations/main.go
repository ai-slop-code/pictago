package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"

	"github.com/ai-slop-code/pictago/assets"
	"github.com/ai-slop-code/pictago/cmd/pictago_db_migrations/migrations"
	"github.com/ai-slop-code/pictago/internal/config"
	"github.com/ai-slop-code/pictago/internal/log"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

func loadServerConfig(configFileName string) error {
	cfgBytes, err := assets.CfgFs.ReadFile(configFileName)
	if err != nil {
		return err
	}
	return config.LoadConfiguration(cfgBytes)
}

var (
	logger = log.NewDefaultLogger()
	_      = migrations.Fs
)

func main() {
	if err := loadServerConfig(assets.ConfigFileName); err != nil {
		panic(err)
	}

	ctx := context.Background()
	fmt.Println("Starting database migration process")
	connStr := config.GetConfig().Database.ConnectionString()

	dbConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		logger.Error("Failed to parse database config", err)
	}

	pool, err := pgxpool.NewWithConfig(
		ctx,
		dbConfig,
	)
	if err != nil {
		logger.Error("Failed to create database connection pool", err)
	}
	defer pool.Close()

	fmt.Println("Database connection established successfully")

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		logger.Error("Unable to connect to database", err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		logger.Error("Could not create migration instance", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", config.Cfg.Database.MigrationsDir),
		"postgres",
		driver,
	)
	if err != nil {
		logger.Error("Could not finish database migrations", "operations", "revert")
	}

	cmd := os.Args[len(os.Args)-1]
	if cmd == "up" {
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			logger.Error("Migration up failed: %v", err)
		}
		fmt.Println("Migration up completed successfully")
	}

	if cmd == "down" {
		if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			logger.Error("Migration down failed", err)
		}
		fmt.Println("Migration down completed successfully")
	}
}

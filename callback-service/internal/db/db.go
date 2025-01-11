package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"callback-service/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

func GetConnStr(cfg config.Database) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, cfg.SSLMode)
}

func RunMigrations(connStr, migrationsDir string) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := goose.Up(db, migrationsDir); err != nil {
		log.Fatal(err)
	}
}

func GetPool(connStr string) (*pgxpool.Pool, error) {
	dbpool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		return nil, err
	}
	return dbpool, nil
}

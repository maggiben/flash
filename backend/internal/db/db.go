package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return pgxpool.NewWithConfig(ctx, cfg)
}

func RunMigrations(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", e.Name(), err)
		}
		if _, err := pool.Exec(ctx, string(body)); err != nil {
			return fmt.Errorf("exec migration %s: %w", e.Name(), err)
		}
	}
	return nil
}

func RunSeed(ctx context.Context, pool *pgxpool.Pool, path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read seed: %w", err)
	}
	if _, err := pool.Exec(ctx, string(body)); err != nil {
		return fmt.Errorf("exec seed: %w", err)
	}
	return nil
}

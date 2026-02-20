package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	pool *pgxpool.Pool
}

func Open(ctx context.Context, databaseURL string) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	cfg.MaxConnLifetime = 5 * time.Minute
	cfg.MaxConnIdleTime = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return &DB{pool: pool}, nil
}

func (d *DB) Close() {
	d.pool.Close()
}

func (d *DB) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return d.pool.Ping(ctx)
}

func (d *DB) Exec(ctx context.Context, sql string, args ...any) error {
	_, err := d.pool.Exec(ctx, sql, args...)
	return err
}

func (d *DB) QueryRow(ctx context.Context, sql string, args ...any) Row {
	return d.pool.QueryRow(ctx, sql, args...)
}

func (d *DB) Query(ctx context.Context, sql string, args ...any) (Rows, error) {
	return d.pool.Query(ctx, sql, args...)
}

type Row interface {
	Scan(dest ...any) error
}

type Rows interface {
	Close()
	Err() error
	Next() bool
	Scan(dest ...any) error
}

var ErrNotFound = errors.New("not found")

func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound) || (err != nil && err.Error() == "no rows in result set")
}

func WrapNotFound(err error) error {
	if err == nil {
		return nil
	}
	if err.Error() == "no rows in result set" {
		return ErrNotFound
	}
	return fmt.Errorf("db: %w", err)
}

package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/example/resy-scheduler/internal/domain/user"
	"github.com/example/resy-scheduler/internal/internaltypes"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepo struct{ pool *pgxpool.Pool }

func NewUserRepo(pool *pgxpool.Pool) *UserRepo { return &UserRepo{pool: pool} }

func (r *UserRepo) Create(ctx context.Context, u user.User) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO users (id, username, password_hash, created_at) VALUES ($1,$2,$3,$4)`,
		u.ID, u.Username, u.PasswordHash, u.CreatedAt,
	)
	return err
}

func (r *UserRepo) GetByUsername(ctx context.Context, username string) (user.User, error) {
	row := r.pool.QueryRow(ctx, `SELECT id, username, password_hash, created_at FROM users WHERE username=$1`, username)
	var u user.User
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return user.User{}, internaltypes.ErrNotFound
		}
		return user.User{}, err
	}
	return u, nil
}

func (r *UserRepo) EnsureCredentialsRow(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO credentials (user_id) VALUES ($1)
		ON CONFLICT (user_id) DO NOTHING
	`, userID)
	return err
}

func (r *UserRepo) GetCredentials(ctx context.Context, userID string) (user.Credentials, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT user_id, resy_api_key, resy_auth_token, opentable_token, opentable_pq_hash, created_at, updated_at
		FROM credentials WHERE user_id=$1
	`, userID)
	var c user.Credentials
	if err := row.Scan(&c.UserID, &c.ResyAPIKey, &c.ResyAuthToken, &c.OpenTableToken, &c.OpenTablePQHash, &c.CreatedAt, &c.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return user.Credentials{}, internaltypes.ErrNotFound
		}
		return user.Credentials{}, err
	}
	return c, nil
}

func (r *UserRepo) UpdateCredentials(ctx context.Context, c user.Credentials) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE credentials
		SET resy_api_key=$2, resy_auth_token=$3, opentable_token=$4, opentable_pq_hash=$5, updated_at=$6
		WHERE user_id=$1
	`, c.UserID, c.ResyAPIKey, c.ResyAuthToken, c.OpenTableToken, c.OpenTablePQHash, time.Now().UTC())
	return err
}

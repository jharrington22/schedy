package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/example/resy-scheduler/internal/domain/user"
	"github.com/example/resy-scheduler/internal/infrastructure/postgres"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	Users *postgres.UserRepo
}

func (a AuthService) VerifyPassword(ctx context.Context, username, password string) (user.User, error) {
	u, err := a.Users.GetByUsername(ctx, username)
	if err != nil {
		return user.User{}, err
	}
	if err := bcrypt.CompareHashAndPassword(u.PasswordHash, []byte(password)); err != nil {
		return user.User{}, fmt.Errorf("invalid credentials")
	}
	_ = a.Users.EnsureCredentialsRow(ctx, u.ID)
	return u, nil
}

func HashPassword(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
}

func NewUser(username, password string) (user.User, error) {
	h, err := HashPassword(password)
	if err != nil { return user.User{}, err }
	return user.User{
		ID: fmt.Sprintf("u_%d", time.Now().UnixNano()),
		Username: username,
		PasswordHash: h,
		CreatedAt: time.Now().UTC(),
	}, nil
}

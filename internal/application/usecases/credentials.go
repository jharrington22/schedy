package usecases

import (
	"context"

	"github.com/example/resy-scheduler/internal/domain/user"
	"github.com/example/resy-scheduler/internal/infrastructure/crypto"
	"github.com/example/resy-scheduler/internal/infrastructure/postgres"
)

type CredentialsService struct {
	Users *postgres.UserRepo
	AEAD  *crypto.AEAD
}

func (s CredentialsService) Get(ctx context.Context, userID string) (user.Credentials, error) {
	c, err := s.Users.GetCredentials(ctx, userID)
	if err != nil {
		return user.Credentials{}, err
	}
	// decrypt fields
	if c.ResyAPIKey != "" { if v, err := s.AEAD.DecryptString(c.ResyAPIKey); err == nil { c.ResyAPIKey = v } }
	if c.ResyAuthToken != "" { if v, err := s.AEAD.DecryptString(c.ResyAuthToken); err == nil { c.ResyAuthToken = v } }
	if c.OpenTableToken != "" { if v, err := s.AEAD.DecryptString(c.OpenTableToken); err == nil { c.OpenTableToken = v } }
	// pq hash is not sensitive; store plaintext
	return c, nil
}

func (s CredentialsService) Update(ctx context.Context, c user.Credentials) error {
	// encrypt sensitive values
	var err error
	if c.ResyAPIKey != "" { c.ResyAPIKey, err = s.AEAD.EncryptToString(c.ResyAPIKey); if err != nil { return err } }
	if c.ResyAuthToken != "" { c.ResyAuthToken, err = s.AEAD.EncryptToString(c.ResyAuthToken); if err != nil { return err } }
	if c.OpenTableToken != "" { c.OpenTableToken, err = s.AEAD.EncryptToString(c.OpenTableToken); if err != nil { return err } }
	return s.Users.UpdateCredentials(ctx, c)
}

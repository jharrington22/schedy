package user

import "time"

type User struct {
	ID           string
	Username     string
	PasswordHash []byte
	CreatedAt    time.Time
}

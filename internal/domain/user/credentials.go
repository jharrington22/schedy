package user

import "time"

type Credentials struct {
	UserID string

	ResyAPIKey     string
	ResyAuthToken  string
	OpenTableToken string

	OpenTablePQHash string // optional override

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (c Credentials) HasResy() bool {
	return c.ResyAPIKey != "" && c.ResyAuthToken != ""
}
func (c Credentials) HasOpenTable() bool {
	return c.OpenTableToken != ""
}

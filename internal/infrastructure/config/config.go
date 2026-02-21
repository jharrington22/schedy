package config

import (
	"os"
	"strings"
)

type Config struct {
	ResyAPIKey    string
	ResyAuthToken string

	OpenTableToken string
	OpenTablePersistedQuerySHA256 string

	// OpenTable default contact (can be overridden per request/job in the future)
	FirstName string
	LastName  string
	Email     string
	Phone     string
}

func FromEnv() Config {
	return Config{
		ResyAPIKey:    strings.TrimSpace(os.Getenv("RESY_API_KEY")),
		ResyAuthToken: strings.TrimSpace(os.Getenv("RESY_AUTH_TOKEN")),
		OpenTableToken: strings.TrimSpace(os.Getenv("OPENTABLE_TOKEN")),
		OpenTablePersistedQuerySHA256: strings.TrimSpace(os.Getenv("OPENTABLE_PERSISTED_QUERY_SHA256")),
		FirstName: strings.TrimSpace(os.Getenv("BOOKING_FIRST_NAME")),
		LastName:  strings.TrimSpace(os.Getenv("BOOKING_LAST_NAME")),
		Email:     strings.TrimSpace(os.Getenv("BOOKING_EMAIL")),
		Phone:     strings.TrimSpace(os.Getenv("BOOKING_PHONE")),
	}
}

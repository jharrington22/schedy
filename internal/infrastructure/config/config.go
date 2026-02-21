package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	HTTPAddr    string
	DatabaseURL string

	SessionHashKey  []byte // base64
	SessionBlockKey []byte // base64

	CredEncKey []byte // 32 bytes for AES-256-GCM, base64

	DevMode bool
}

func FromEnv() (Config, error) {
	cfg := Config{
		HTTPAddr:    envDefault("HTTP_ADDR", ":8080"),
		DatabaseURL: strings.TrimSpace(os.Getenv("DATABASE_URL")),
		DevMode:     strings.TrimSpace(os.Getenv("DEV_MODE")) == "1",
	}
	if cfg.DatabaseURL == "" {
		return cfg, fmt.Errorf("DATABASE_URL is required")
	}
	var err error
	cfg.SessionHashKey, err = mustB64("SESSION_HASH_KEY")
	if err != nil {
		return cfg, err
	}
	cfg.SessionBlockKey, err = mustB64("SESSION_BLOCK_KEY")
	if err != nil {
		return cfg, err
	}
	cfg.CredEncKey, err = mustB64("CRED_ENC_KEY")
	if err != nil {
		return cfg, err
	}
	if len(cfg.CredEncKey) != 32 {
		return cfg, fmt.Errorf("CRED_ENC_KEY must decode to 32 bytes (got %d)", len(cfg.CredEncKey))
	}
	return cfg, nil
}

func envDefault(k, d string) string {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return d
	}
	return v
}

func mustB64(k string) ([]byte, error) {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return nil, fmt.Errorf("%s is required (base64)", k)
	}
	if b, err := base64.StdEncoding.DecodeString(v); err == nil {
		return b, nil
	}
	b, err := base64.RawStdEncoding.DecodeString(v)
	if err != nil {
		return nil, err
	}
	return b, nil
}

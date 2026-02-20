package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	ListenAddr     string
	BaseURL        string
	DatabaseURL    string
	CookieHashKey  []byte
	CookieBlockKey []byte

	// scheduler
	PollInterval time.Duration
	ResyBin      string
}

func FromEnv() (Config, error) {
	cfg := Config{
		ListenAddr:  getenv("LISTEN_ADDR", ":8080"),
		BaseURL:     getenv("BASE_URL", "http://localhost:8080"),
		DatabaseURL: getenv("DATABASE_URL", "postgres://resy:resy@localhost:5432/resy?sslmode=disable"),
		ResyBin:     getenv("RESY_BIN", "resy"),
	}

	pollSec, err := strconv.Atoi(getenv("SCHED_POLL_SECONDS", "2"))
	if err != nil || pollSec < 1 {
		return Config{}, fmt.Errorf("invalid SCHED_POLL_SECONDS")
	}
	cfg.PollInterval = time.Duration(pollSec) * time.Second

	hashKey := os.Getenv("COOKIE_HASH_KEY")
	blockKey := os.Getenv("COOKIE_BLOCK_KEY")
	if hashKey == "" || blockKey == "" {
		return Config{}, fmt.Errorf("COOKIE_HASH_KEY and COOKIE_BLOCK_KEY are required (32 and 32/16/24/32 bytes base64)")
	}
	var derr error
	cfg.CookieHashKey, derr = decodeB64(hashKey)
	if derr != nil {
		return Config{}, fmt.Errorf("COOKIE_HASH_KEY: %w", derr)
	}
	cfg.CookieBlockKey, derr = decodeB64(blockKey)
	if derr != nil {
		return Config{}, fmt.Errorf("COOKIE_BLOCK_KEY: %w", derr)
	}

	return cfg, nil
}

func decodeB64(s string) ([]byte, error) {
	b, err := os.ReadFile(s)
	if err == nil {
		// allow pointing to file path for k8s secret mounts
		s = string(b)
	}
	// trim whitespace/newlines
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r' || s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	dec, err := decodeStdB64(s)
	if err != nil {
		return nil, err
	}
	return dec, nil
}

func decodeStdB64(s string) ([]byte, error) {
	// avoid importing encoding/base64 everywhere
	return base64Decode(s)
}

func getenv(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}

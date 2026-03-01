package config

import (
	"fmt"
	"os"
)

type Config struct {
	Addr  string
	Token string
}

func Load() (*Config, error) {
	addr := os.Getenv("VAULT_ADDR")
	if addr == "" {
		return nil, fmt.Errorf("VAULT_ADDR environment variable is required")
	}

	token := os.Getenv("VAULT_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("VAULT_TOKEN environment variable is required")
	}

	return &Config{
		Addr:  addr,
		Token: token,
	}, nil
}

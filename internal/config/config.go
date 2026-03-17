package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Addr  string
	Token string
	Theme string
}

type fileConfig struct {
	VaultAddr  string `toml:"vault_addr"`
	VaultToken string `toml:"vault_token"`
	Theme      string `toml:"theme"`
}

func loadFromFile() (*fileConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(home, ".config", "hv-tui", "config.toml")

	var fc fileConfig
	if _, err := toml.DecodeFile(path, &fc); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("error reading config file %s: %w", path, err)
	}

	return &fc, nil
}

func Load() (*Config, error) {
	fc, err := loadFromFile()
	if err != nil {
		return nil, err
	}

	cfg := &Config{}

	if fc != nil {
		cfg.Addr = fc.VaultAddr
		cfg.Token = fc.VaultToken
		cfg.Theme = fc.Theme
	}

	if v := os.Getenv("VAULT_ADDR"); v != "" {
		cfg.Addr = v
	}
	if v := os.Getenv("VAULT_TOKEN"); v != "" {
		cfg.Token = v
	}

	if cfg.Addr == "" {
		return nil, fmt.Errorf("VAULT_ADDR is required (set in ~/.config/hv-tui/config.toml or VAULT_ADDR env var)")
	}
	if cfg.Token == "" {
		return nil, fmt.Errorf("VAULT_TOKEN is required (set in ~/.config/hv-tui/config.toml or VAULT_TOKEN env var)")
	}

	return cfg, nil
}

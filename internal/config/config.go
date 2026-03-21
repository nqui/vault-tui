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
	path, err := configPath()
	if err != nil {
		return nil, err
	}

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

	return cfg, nil
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "hv-tui", "config.toml"), nil
}

func Save(cfg *Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	fc := fileConfig{
		VaultAddr:  cfg.Addr,
		VaultToken: cfg.Token,
		Theme:      cfg.Theme,
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}
	defer f.Close()

	if err := toml.NewEncoder(f).Encode(fc); err != nil {
		return fmt.Errorf("encoding config file: %w", err)
	}

	return nil
}

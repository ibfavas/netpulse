package config

import (
	"errors"
	"os"
	"os/user"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Targets struct {
		Backbones []string `toml:"backbones"`
		DNS       []struct {
			Name string `toml:"name"`
			Addr string `toml:"addr"`
		} `toml:"dns"`
	} `toml:"targets"`
	Theme struct {
		Good   string `toml:"good"`
		Warn   string `toml:"warn"`
		Bad    string `toml:"bad"`
		Title  string `toml:"title"`
		Border string `toml:"border"`
	} `toml:"theme"`
	Daemon struct {
		LogFile        string `toml:"log_file"`
		Interval       int    `toml:"interval"`
		AlertThreshold int    `toml:"alert_threshold_ms"`
	} `toml:"daemon"`
}

func DefaultConfig() *Config {
	cfg := &Config{}
	cfg.Targets.Backbones = []string{"1.1.1.1", "8.8.8.8"}
	cfg.Targets.DNS = []struct {
		Name string `toml:"name"`
		Addr string `toml:"addr"`
	}{
		{"System", "system"},
		{"Google", "8.8.8.8"},
		{"Cloudflare", "1.1.1.1"},
	}
	cfg.Theme.Good = "42"
	cfg.Theme.Warn = "220"
	cfg.Theme.Bad = "196"
	cfg.Theme.Title = "62"
	cfg.Theme.Border = "240"
	cfg.Daemon.LogFile = "~/.local/share/netpulse/metrics.log"
	cfg.Daemon.Interval = 5
	cfg.Daemon.AlertThreshold = 100
	return cfg
}

func GetConfigPath() string {
	sudoUser := os.Getenv("SUDO_USER")
	if sudoUser != "" && sudoUser != "root" {
		u, err := user.Lookup(sudoUser)
		if err == nil {
			return filepath.Join(u.HomeDir, ".config", "netpulse", "config.toml")
		}
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "netpulse", "config.toml")
}

func LoadConfig() *Config {
	cfg := DefaultConfig()

	configPath := GetConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			os.MkdirAll(filepath.Dir(configPath), 0755)
			b, _ := toml.Marshal(cfg)
			os.WriteFile(configPath, b, 0644)
		}
		return cfg
	}

	toml.Unmarshal(data, cfg)
	return cfg
}

func SaveConfig(cfg *Config) error {
	configPath := GetConfigPath()
	b, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, b, 0644)
}

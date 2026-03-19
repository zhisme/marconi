package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Build-time defaults set via -ldflags:
//
//	go build -ldflags "-X github.com/zhisme/marconi/config.DefaultBotToken=123:ABC -X github.com/zhisme/marconi/config.DefaultAPIID=12345 -X github.com/zhisme/marconi/config.DefaultAPIHash=abc123"
var (
	DefaultBotToken string
	DefaultAPIID    string
	DefaultAPIHash  string
)

type Config struct {
	BotToken   string `yaml:"bot_token"`
	ChannelID  string `yaml:"channel_id"`
	DelayHours int    `yaml:"delay_hours"`
	APIID      int    `yaml:"api_id"`
	APIHash    string `yaml:"api_hash"`
}

type CLIFlags struct {
	BotToken  string
	ChannelID string
}

func LoadFromPath(path string, flags CLIFlags) (Config, error) {
	var cfg Config

	data, err := os.ReadFile(path)
	if err == nil {
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return Config{}, fmt.Errorf("invalid config file: %w", err)
		}
	}

	// CLI flags override file values
	if flags.BotToken != "" {
		cfg.BotToken = flags.BotToken
	}
	if flags.ChannelID != "" {
		cfg.ChannelID = flags.ChannelID
	}

	// Fall back to embedded defaults
	if cfg.BotToken == "" {
		cfg.BotToken = DefaultBotToken
	}
	if cfg.APIID == 0 && DefaultAPIID != "" {
		if id, err := strconv.Atoi(DefaultAPIID); err == nil {
			cfg.APIID = id
		}
	}
	if cfg.APIHash == "" {
		cfg.APIHash = DefaultAPIHash
	}

	// Default delay
	if cfg.DelayHours == 0 {
		cfg.DelayHours = 24
	}

	// Validate required fields
	if cfg.BotToken == "" {
		return Config{}, fmt.Errorf("missing bot_token. Set via config, --token flag, or build with embedded token")
	}
	if cfg.ChannelID == "" {
		return Config{}, fmt.Errorf("missing channel_id. Run: marconi init")
	}
	if cfg.APIID == 0 {
		return Config{}, fmt.Errorf("missing api_id. Set via config or build with -ldflags")
	}
	if cfg.APIHash == "" {
		return Config{}, fmt.Errorf("missing api_hash. Set via config or build with -ldflags")
	}

	return cfg, nil
}

// SessionPath returns the path to the MTProto session file.
func SessionPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "marconi", "session.json")
}

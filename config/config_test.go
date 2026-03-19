package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_FromFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yml")
	content := []byte("bot_token: \"123:ABC\"\nchannel_id: \"@testchan\"\ndelay_hours: 12\napi_id: 12345\napi_hash: \"abc123\"\n")
	if err := os.WriteFile(configPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromPath(configPath, CLIFlags{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.BotToken != "123:ABC" {
		t.Errorf("BotToken = %q, want %q", cfg.BotToken, "123:ABC")
	}
	if cfg.ChannelID != "@testchan" {
		t.Errorf("ChannelID = %q, want %q", cfg.ChannelID, "@testchan")
	}
	if cfg.DelayHours != 12 {
		t.Errorf("DelayHours = %d, want %d", cfg.DelayHours, 12)
	}
	if cfg.APIID != 12345 {
		t.Errorf("APIID = %d, want %d", cfg.APIID, 12345)
	}
	if cfg.APIHash != "abc123" {
		t.Errorf("APIHash = %q, want %q", cfg.APIHash, "abc123")
	}
}

func TestLoad_CLIFlagsOverrideFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yml")
	content := []byte("bot_token: \"file-token\"\nchannel_id: \"@filechan\"\ndelay_hours: 24\napi_id: 12345\napi_hash: \"abc123\"\n")
	if err := os.WriteFile(configPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	flags := CLIFlags{
		ChannelID: "@clichan",
	}

	cfg, err := LoadFromPath(configPath, flags)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ChannelID != "@clichan" {
		t.Errorf("ChannelID = %q, want %q (CLI should override file)", cfg.ChannelID, "@clichan")
	}
	if cfg.DelayHours != 24 {
		t.Errorf("DelayHours = %d, want %d (should keep file value)", cfg.DelayHours, 24)
	}
}

func TestLoad_DefaultTokenFromBuild(t *testing.T) {
	original := DefaultBotToken
	DefaultBotToken = "embedded-token"
	defer func() { DefaultBotToken = original }()

	origAPIID := DefaultAPIID
	origAPIHash := DefaultAPIHash
	DefaultAPIID = "99999"
	DefaultAPIHash = "embeddedHash"
	defer func() { DefaultAPIID = origAPIID; DefaultAPIHash = origAPIHash }()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yml")
	content := []byte("channel_id: \"@testchan\"\n")
	if err := os.WriteFile(configPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromPath(configPath, CLIFlags{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.BotToken != "embedded-token" {
		t.Errorf("BotToken = %q, want %q (should use embedded default)", cfg.BotToken, "embedded-token")
	}
	if cfg.APIID != 99999 {
		t.Errorf("APIID = %d, want %d (should use embedded default)", cfg.APIID, 99999)
	}
	if cfg.APIHash != "embeddedHash" {
		t.Errorf("APIHash = %q, want %q (should use embedded default)", cfg.APIHash, "embeddedHash")
	}
}

func TestLoad_ChannelOnly_WithEmbeddedToken(t *testing.T) {
	original := DefaultBotToken
	DefaultBotToken = "embedded-token"
	defer func() { DefaultBotToken = original }()

	origAPIID := DefaultAPIID
	origAPIHash := DefaultAPIHash
	DefaultAPIID = "12345"
	DefaultAPIHash = "hash123"
	defer func() { DefaultAPIID = origAPIID; DefaultAPIHash = origAPIHash }()

	cfg, err := LoadFromPath("/nonexistent/config.yml", CLIFlags{ChannelID: "@clichan"})
	if err != nil {
		t.Fatalf("should succeed with embedded token + CLI channel: %v", err)
	}
	if cfg.BotToken != "embedded-token" {
		t.Errorf("BotToken = %q, want %q", cfg.BotToken, "embedded-token")
	}
	if cfg.ChannelID != "@clichan" {
		t.Errorf("ChannelID = %q, want %q", cfg.ChannelID, "@clichan")
	}
}

func TestLoad_MissingChannelID(t *testing.T) {
	original := DefaultBotToken
	DefaultBotToken = "embedded-token"
	defer func() { DefaultBotToken = original }()

	origAPIID := DefaultAPIID
	origAPIHash := DefaultAPIHash
	DefaultAPIID = "12345"
	DefaultAPIHash = "hash123"
	defer func() { DefaultAPIID = origAPIID; DefaultAPIHash = origAPIHash }()

	_, err := LoadFromPath("/nonexistent/config.yml", CLIFlags{})
	if err == nil {
		t.Fatal("expected error when no channel_id, got nil")
	}
}

func TestLoad_MissingEverything(t *testing.T) {
	original := DefaultBotToken
	DefaultBotToken = ""
	defer func() { DefaultBotToken = original }()

	origAPIID := DefaultAPIID
	origAPIHash := DefaultAPIHash
	DefaultAPIID = ""
	DefaultAPIHash = ""
	defer func() { DefaultAPIID = origAPIID; DefaultAPIHash = origAPIHash }()

	_, err := LoadFromPath("/nonexistent/config.yml", CLIFlags{})
	if err == nil {
		t.Fatal("expected error when no config, no flags, no embedded token, got nil")
	}
}

func TestLoad_MissingAPIID(t *testing.T) {
	original := DefaultBotToken
	DefaultBotToken = "token"
	defer func() { DefaultBotToken = original }()

	origAPIID := DefaultAPIID
	origAPIHash := DefaultAPIHash
	DefaultAPIID = ""
	DefaultAPIHash = ""
	defer func() { DefaultAPIID = origAPIID; DefaultAPIHash = origAPIHash }()

	_, err := LoadFromPath("/nonexistent/config.yml", CLIFlags{ChannelID: "@ch"})
	if err == nil {
		t.Fatal("expected error when no api_id, got nil")
	}
}

func TestLoad_MissingAPIHash(t *testing.T) {
	original := DefaultBotToken
	DefaultBotToken = "token"
	defer func() { DefaultBotToken = original }()

	origAPIID := DefaultAPIID
	origAPIHash := DefaultAPIHash
	DefaultAPIID = "12345"
	DefaultAPIHash = ""
	defer func() { DefaultAPIID = origAPIID; DefaultAPIHash = origAPIHash }()

	_, err := LoadFromPath("/nonexistent/config.yml", CLIFlags{ChannelID: "@ch"})
	if err == nil {
		t.Fatal("expected error when no api_hash, got nil")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(configPath, []byte("{{not yaml"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadFromPath(configPath, CLIFlags{})
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestLoad_DefaultDelayHours(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yml")
	content := []byte("bot_token: \"123:ABC\"\nchannel_id: \"@testchan\"\napi_id: 12345\napi_hash: \"abc\"\n")
	if err := os.WriteFile(configPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromPath(configPath, CLIFlags{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DelayHours != 24 {
		t.Errorf("DelayHours = %d, want %d (default)", cfg.DelayHours, 24)
	}
}

func TestSessionPath(t *testing.T) {
	path := SessionPath()
	if path == "" {
		t.Fatal("SessionPath() returned empty string")
	}
	if filepath.Base(path) != "session.json" {
		t.Errorf("SessionPath() = %q, expected session.json filename", path)
	}
}

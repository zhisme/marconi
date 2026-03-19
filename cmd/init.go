package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/zhisme/marconi/config"
)

func RunInit(in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)

	fmt.Fprint(out, "Channel ID (e.g. @mychannel or -100123456): ")
	scanner.Scan()
	channelID := strings.TrimSpace(scanner.Text())
	if channelID == "" {
		return fmt.Errorf("channel ID is required")
	}

	fmt.Fprint(out, "Delay hours [24]: ")
	scanner.Scan()
	delayStr := strings.TrimSpace(scanner.Text())
	if delayStr == "" {
		delayStr = "24"
	}

	var apiIDStr, apiHash string

	if config.DefaultAPIID == "" {
		fmt.Fprint(out, "API ID (from https://my.telegram.org): ")
		scanner.Scan()
		apiIDStr = strings.TrimSpace(scanner.Text())
		if apiIDStr == "" {
			return fmt.Errorf("API ID is required")
		}
	}

	if config.DefaultAPIHash == "" {
		fmt.Fprint(out, "API Hash (from https://my.telegram.org): ")
		scanner.Scan()
		apiHash = strings.TrimSpace(scanner.Text())
		if apiHash == "" {
			return fmt.Errorf("API Hash is required")
		}
	}

	configDir := filepath.Join(homeDir(), ".config", "marconi")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("cannot create config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "config.yml")
	content := fmt.Sprintf("channel_id: %q\ndelay_hours: %s\n", channelID, delayStr)
	if apiIDStr != "" {
		content += fmt.Sprintf("api_id: %s\n", apiIDStr)
	}
	if apiHash != "" {
		content += fmt.Sprintf("api_hash: %q\n", apiHash)
	}

	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("cannot write config: %w", err)
	}

	fmt.Fprintf(out, "Config saved to %s\n", configPath)
	fmt.Fprintln(out, "Now add @MarconiPostBot as admin to your channel with 'Post Messages' permission.")
	return nil
}

func homeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return os.Getenv("HOME")
	}
	return home
}

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zhisme/marconi/cmd"
	"github.com/zhisme/marconi/config"
	"github.com/zhisme/marconi/telegram"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "send":
		if err := runSend(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "preview":
		if err := runPreview(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "init":
		if err := cmd.RunInit(os.Stdin, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "--help", "-h", "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func runSend(args []string) error {
	fs := flag.NewFlagSet("send", flag.ExitOnError)
	imagePath := fs.String("i", "", "image file to attach")
	now := fs.Bool("now", false, "send immediately (skip scheduling)")
	channel := fs.String("channel", "", "channel ID (overrides config)")
	fs.Parse(args)

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: marconi send <file.md> [-i image] [--now] [--channel ID]")
	}
	mdFile := fs.Arg(0)

	cfg, err := loadConfig(config.CLIFlags{ChannelID: *channel})
	if err != nil {
		return err
	}

	ctx := context.Background()
	client, err := telegram.NewClient(ctx, cfg.APIID, cfg.APIHash, cfg.BotToken, cfg.ChannelID, config.SessionPath())
	if err != nil {
		return err
	}
	defer client.Close()

	return cmd.RunSend(ctx, client, cfg.DelayHours, mdFile, *imagePath, *now)
}

func runPreview(args []string) error {
	fs := flag.NewFlagSet("preview", flag.ExitOnError)
	fs.Parse(args)

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: marconi preview <file.md>")
	}

	return cmd.RunPreview(fs.Arg(0), os.Stdout)
}

func loadConfig(flags config.CLIFlags) (config.Config, error) {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".config", "marconi", "config.yml")
	return config.LoadFromPath(configPath, flags)
}

func printUsage() {
	fmt.Println(`marconi — send markdown posts to Telegram channels via MTProto

Usage:
  marconi send <file.md> [-i image] [--now] [--channel ID]
  marconi preview <file.md>
  marconi init

Commands:
  send      Convert markdown and send to Telegram channel (scheduled 24h by default)
  preview   Convert markdown and print MarkdownV2 to stdout (dry run)
  init      Set your channel ID and API credentials (one-time setup)

Build with embedded credentials:
  go build -ldflags "-X github.com/zhisme/marconi/config.DefaultBotToken=TOKEN -X github.com/zhisme/marconi/config.DefaultAPIID=12345 -X github.com/zhisme/marconi/config.DefaultAPIHash=HASH"`)
}

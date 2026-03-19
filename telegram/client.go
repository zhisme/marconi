package telegram

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gotd/contrib/bg"
	"github.com/gotd/td/session"
	tdclient "github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/uploader"
	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"
)

// Sender defines the interface for sending messages via Telegram.
type Sender interface {
	SendMessage(ctx context.Context, text string, entities []tg.MessageEntityClass, scheduleDate int) (int, error)
	SendPhoto(ctx context.Context, caption string, captionEntities []tg.MessageEntityClass, imagePath string, scheduleDate int) (int, error)
	Close()
}

// Client implements Sender using MTProto via gotd/td.
type Client struct {
	api  *tg.Client
	stop bg.StopFunc
	peer tg.InputPeerClass
	up   *uploader.Uploader
}

// NewClient creates a new MTProto client, authenticates the bot, and resolves the target channel.
func NewClient(ctx context.Context, apiID int, apiHash, botToken, channelID, sessionPath string) (*Client, error) {
	if err := os.MkdirAll(filepath.Dir(sessionPath), 0700); err != nil {
		return nil, fmt.Errorf("cannot create session directory: %w", err)
	}

	client := tdclient.NewClient(apiID, apiHash, tdclient.Options{
		SessionStorage: &session.FileStorage{Path: sessionPath},
	})

	stop, err := bg.Connect(client)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Telegram: %w", err)
	}

	// Bot auth (skipped if session is still valid)
	status, err := client.Auth().Status(ctx)
	if err != nil {
		stop()
		return nil, fmt.Errorf("auth check failed: %w", err)
	}
	if !status.Authorized {
		if _, err := client.Auth().Bot(ctx, botToken); err != nil {
			stop()
			return nil, classifyError(err)
		}
	}

	api := client.API()

	peer, err := resolveChannel(ctx, api, channelID)
	if err != nil {
		stop()
		return nil, err
	}

	return &Client{
		api:  api,
		stop: stop,
		peer: peer,
		up:   uploader.NewUploader(api),
	}, nil
}

func (c *Client) SendMessage(ctx context.Context, text string, entities []tg.MessageEntityClass, scheduleDate int) (int, error) {
	req := &tg.MessagesSendMessageRequest{
		Peer:     c.peer,
		Message:  text,
		RandomID: randomID(),
	}
	if len(entities) > 0 {
		req.SetEntities(entities)
	}
	if scheduleDate > 0 {
		req.SetScheduleDate(scheduleDate)
	}

	updates, err := c.api.MessagesSendMessage(ctx, req)
	if err != nil {
		return 0, classifyError(err)
	}
	return extractMessageID(updates), nil
}

func (c *Client) SendPhoto(ctx context.Context, caption string, captionEntities []tg.MessageEntityClass, imagePath string, scheduleDate int) (int, error) {
	f, err := c.up.FromPath(ctx, imagePath)
	if err != nil {
		return 0, fmt.Errorf("failed to upload photo: %w", err)
	}

	req := &tg.MessagesSendMediaRequest{
		Peer: c.peer,
		Media: &tg.InputMediaUploadedPhoto{
			File: f,
		},
		Message:  caption,
		RandomID: randomID(),
	}
	if len(captionEntities) > 0 {
		req.SetEntities(captionEntities)
	}
	if scheduleDate > 0 {
		req.SetScheduleDate(scheduleDate)
	}

	updates, err := c.api.MessagesSendMedia(ctx, req)
	if err != nil {
		return 0, classifyError(err)
	}
	return extractMessageID(updates), nil
}

func (c *Client) Close() {
	c.stop()
}

func resolveChannel(ctx context.Context, api *tg.Client, channelID string) (tg.InputPeerClass, error) {
	if strings.HasPrefix(channelID, "@") {
		username := strings.TrimPrefix(channelID, "@")
		res, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{Username: username})
		if err != nil {
			return nil, classifyError(err)
		}
		for _, ch := range res.Chats {
			if channel, ok := ch.(*tg.Channel); ok {
				return channel.AsInputPeer(), nil
			}
		}
		return nil, fmt.Errorf("@%s is not a channel", username)
	}

	// Numeric channel ID (e.g., -1001234567890)
	idStr := channelID
	if strings.HasPrefix(idStr, "-100") {
		idStr = strings.TrimPrefix(idStr, "-100")
	} else if strings.HasPrefix(idStr, "-") {
		idStr = strings.TrimPrefix(idStr, "-")
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid channel ID: %s", channelID)
	}

	return &tg.InputPeerChannel{ChannelID: id}, nil
}

func extractMessageID(updates tg.UpdatesClass) int {
	switch u := updates.(type) {
	case *tg.UpdateShortSentMessage:
		return u.ID
	case *tg.Updates:
		for _, update := range u.Updates {
			if m, ok := update.(*tg.UpdateNewChannelMessage); ok {
				if msg, ok := m.Message.(*tg.Message); ok {
					return msg.ID
				}
			}
			if m, ok := update.(*tg.UpdateNewMessage); ok {
				if msg, ok := m.Message.(*tg.Message); ok {
					return msg.ID
				}
			}
		}
	}
	return 0
}

func classifyError(err error) error {
	if err == nil {
		return nil
	}

	rpcErr, ok := tgerr.As(err)
	if !ok {
		return fmt.Errorf("Telegram error: %w", err)
	}

	switch rpcErr.Type {
	case "USERNAME_NOT_OCCUPIED", "CHAT_NOT_FOUND", "CHANNEL_INVALID", "PEER_ID_INVALID":
		return fmt.Errorf("channel not found: check that the channel ID is correct (use @username for public channels)")
	case "CHAT_WRITE_FORBIDDEN", "CHAT_ADMIN_REQUIRED":
		return fmt.Errorf("bot lacks permission: add the bot as an admin to the channel with 'Post Messages' permission")
	case "AUTH_KEY_UNREGISTERED", "SESSION_REVOKED", "AUTH_KEY_INVALID":
		return fmt.Errorf("invalid bot token or session expired: check your bot token or delete session file and retry")
	default:
		return fmt.Errorf("Telegram API error (%d %s): %s", rpcErr.Code, rpcErr.Type, rpcErr.Message)
	}
}

func randomID() int64 {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return int64(binary.LittleEndian.Uint64(b[:]))
}

// Ensure *Client implements Sender at compile time.
var _ Sender = (*Client)(nil)


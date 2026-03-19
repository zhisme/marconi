package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gotd/td/tg"
	"github.com/zhisme/marconi/converter"
	"github.com/zhisme/marconi/validator"
)

// Sender defines the interface for sending messages to Telegram.
type Sender interface {
	SendMessage(ctx context.Context, text string, entities []tg.MessageEntityClass, scheduleDate int) (int, error)
	SendPhoto(ctx context.Context, caption string, captionEntities []tg.MessageEntityClass, imagePath string, scheduleDate int) (int, error)
}

func RunSend(ctx context.Context, sender Sender, delayHours int, mdFile, imgFile string, now bool) error {
	source, err := os.ReadFile(mdFile)
	if err != nil {
		return fmt.Errorf("file not found: %s", mdFile)
	}

	plainText, entities, err := converter.ConvertToEntities(source)
	if err != nil {
		return fmt.Errorf("failed to convert markdown: %w", err)
	}

	hasImage := imgFile != ""
	if err := validator.Validate(plainText, hasImage); err != nil {
		return err
	}

	var scheduleDate int
	if !now && delayHours > 0 {
		scheduleDate = int(time.Now().Unix()) + delayHours*3600
	}

	var msgID int
	if hasImage {
		msgID, err = sender.SendPhoto(ctx, plainText, entities, imgFile, scheduleDate)
	} else {
		msgID, err = sender.SendMessage(ctx, plainText, entities, scheduleDate)
	}
	if err != nil {
		return err
	}

	if now {
		fmt.Printf("Sent! (message_id: %d)\n", msgID)
	} else {
		fmt.Printf("Scheduled for %s (message_id: %d)\n",
			time.Unix(int64(scheduleDate), 0).Format(time.RFC3339), msgID)
	}
	return nil
}

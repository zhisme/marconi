package telegram

import (
	"testing"

	"github.com/gotd/td/tg"
)

func TestExtractMessageID_UpdateShortSentMessage(t *testing.T) {
	updates := &tg.UpdateShortSentMessage{ID: 42}
	if got := extractMessageID(updates); got != 42 {
		t.Errorf("extractMessageID = %d, want 42", got)
	}
}

func TestExtractMessageID_Updates_ChannelMessage(t *testing.T) {
	updates := &tg.Updates{
		Updates: []tg.UpdateClass{
			&tg.UpdateNewChannelMessage{
				Message: &tg.Message{ID: 99},
			},
		},
	}
	if got := extractMessageID(updates); got != 99 {
		t.Errorf("extractMessageID = %d, want 99", got)
	}
}

func TestExtractMessageID_Updates_NewMessage(t *testing.T) {
	updates := &tg.Updates{
		Updates: []tg.UpdateClass{
			&tg.UpdateNewMessage{
				Message: &tg.Message{ID: 55},
			},
		},
	}
	if got := extractMessageID(updates); got != 55 {
		t.Errorf("extractMessageID = %d, want 55", got)
	}
}

func TestExtractMessageID_Unknown_ReturnsZero(t *testing.T) {
	updates := &tg.UpdatesTooLong{}
	if got := extractMessageID(updates); got != 0 {
		t.Errorf("extractMessageID = %d, want 0 for unknown type", got)
	}
}

func TestClassifyError_NilReturnsNil(t *testing.T) {
	if err := classifyError(nil); err != nil {
		t.Errorf("classifyError(nil) = %v, want nil", err)
	}
}

func TestResolveChannel_ParsesNumericID(t *testing.T) {
	// We can't test @username resolution without a real API,
	// but we can test numeric ID parsing.
	peer, err := resolveChannel(nil, nil, "-1001234567890")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ch, ok := peer.(*tg.InputPeerChannel)
	if !ok {
		t.Fatalf("expected *tg.InputPeerChannel, got %T", peer)
	}
	if ch.ChannelID != 1234567890 {
		t.Errorf("ChannelID = %d, want 1234567890", ch.ChannelID)
	}
}

func TestResolveChannel_InvalidID(t *testing.T) {
	_, err := resolveChannel(nil, nil, "not-a-number")
	if err == nil {
		t.Fatal("expected error for invalid channel ID")
	}
}

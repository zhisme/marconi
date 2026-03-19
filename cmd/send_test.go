package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/gotd/td/tg"
)

// mockSender implements the Sender interface for testing.
type mockSender struct {
	msgID       int
	err         error
	text        string
	entities    []tg.MessageEntityClass
	schedule    int
	caption     string
	imagePath   string
	messageSent bool
	photoSent   bool
}

func (m *mockSender) SendMessage(_ context.Context, text string, entities []tg.MessageEntityClass, scheduleDate int) (int, error) {
	m.messageSent = true
	m.text = text
	m.entities = entities
	m.schedule = scheduleDate
	return m.msgID, m.err
}

func (m *mockSender) SendPhoto(_ context.Context, caption string, entities []tg.MessageEntityClass, imagePath string, scheduleDate int) (int, error) {
	m.photoSent = true
	m.caption = caption
	m.entities = entities
	m.imagePath = imagePath
	m.schedule = scheduleDate
	return m.msgID, m.err
}

func TestSend_TextOnly_Scheduled(t *testing.T) {
	mdFile := writeTempMD(t, "Hello **world**!")
	mock := &mockSender{msgID: 1}

	err := RunSend(context.Background(), mock, 24, mdFile, "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.messageSent {
		t.Error("expected SendMessage to be called")
	}
	if mock.photoSent {
		t.Error("SendPhoto should not be called for text-only")
	}
	if mock.schedule == 0 {
		t.Error("expected schedule_date to be set for scheduled send")
	}
	// Text should be plain (entities-based, no MarkdownV2 escaping)
	if mock.text != "Hello world!" {
		t.Errorf("text = %q, want %q", mock.text, "Hello world!")
	}
	// Should have a bold entity
	if len(mock.entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(mock.entities))
	}
	if _, ok := mock.entities[0].(*tg.MessageEntityBold); !ok {
		t.Errorf("expected bold entity, got %T", mock.entities[0])
	}
}

func TestSend_WithImage_Scheduled(t *testing.T) {
	mdFile := writeTempMD(t, "Photo caption")
	imgFile := writeTempFile(t, "test.jpg", []byte("fake-image"))
	mock := &mockSender{msgID: 2}

	err := RunSend(context.Background(), mock, 24, mdFile, imgFile, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.photoSent {
		t.Error("expected SendPhoto to be called")
	}
	if mock.messageSent {
		t.Error("SendMessage should not be called when image is present")
	}
	if mock.imagePath != imgFile {
		t.Errorf("imagePath = %q, want %q", mock.imagePath, imgFile)
	}
}

func TestSend_Now_NoScheduleDate(t *testing.T) {
	mdFile := writeTempMD(t, "Immediate post")
	mock := &mockSender{msgID: 3}

	err := RunSend(context.Background(), mock, 24, mdFile, "", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.schedule != 0 {
		t.Errorf("schedule = %d, want 0 for immediate send", mock.schedule)
	}
}

func TestSend_MissingFile(t *testing.T) {
	mock := &mockSender{msgID: 1}
	err := RunSend(context.Background(), mock, 24, "/nonexistent/post.md", "", false)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if mock.messageSent || mock.photoSent {
		t.Error("no API calls should be made for missing file")
	}
}

func TestSend_TextTooLong_NoAPICalled(t *testing.T) {
	longText := ""
	for i := 0; i < 5000; i++ {
		longText += "a"
	}
	mdFile := writeTempMD(t, longText)
	mock := &mockSender{msgID: 99}

	err := RunSend(context.Background(), mock, 24, mdFile, "", false)
	if err == nil {
		t.Fatal("expected validation error for long text")
	}
	if mock.messageSent {
		t.Error("API should NOT be called when validation fails")
	}
}

func TestSend_CaptionTooLong_NoAPICalled(t *testing.T) {
	longText := ""
	for i := 0; i < 1200; i++ {
		longText += "a"
	}
	mdFile := writeTempMD(t, longText)
	imgFile := writeTempFile(t, "img.jpg", []byte("fake"))
	mock := &mockSender{msgID: 99}

	err := RunSend(context.Background(), mock, 24, mdFile, imgFile, false)
	if err == nil {
		t.Fatal("expected validation error for long caption")
	}
	if mock.photoSent {
		t.Error("API should NOT be called when validation fails")
	}
}

func TestSend_SenderError_Propagated(t *testing.T) {
	mdFile := writeTempMD(t, "Hello")
	mock := &mockSender{err: fmt.Errorf("bot lacks permission")}

	err := RunSend(context.Background(), mock, 24, mdFile, "", false)
	if err == nil {
		t.Fatal("expected error from sender to propagate")
	}
	if err.Error() != "bot lacks permission" {
		t.Errorf("error = %q, want %q", err.Error(), "bot lacks permission")
	}
}

// --- Helpers ---

func writeTempMD(t *testing.T, content string) string {
	t.Helper()
	return writeTempFile(t, "post.md", []byte(content))
}

func writeTempFile(t *testing.T, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

package validator

import (
	"strings"
	"testing"
)

func TestValidate_TextMessageAtLimit(t *testing.T) {
	text := strings.Repeat("a", 4096)
	if err := Validate(text, false); err != nil {
		t.Errorf("text at exactly 4096 should pass, got: %v", err)
	}
}

func TestValidate_TextMessageOverLimit(t *testing.T) {
	text := strings.Repeat("a", 4097)
	err := Validate(text, false)
	if err == nil {
		t.Fatal("text at 4097 should fail, got nil")
	}
	// Error should mention the length
	if !strings.Contains(err.Error(), "4097") {
		t.Errorf("error should mention actual length, got: %v", err)
	}
}

func TestValidate_CaptionAtLimit(t *testing.T) {
	text := strings.Repeat("a", 1024)
	if err := Validate(text, true); err != nil {
		t.Errorf("caption at exactly 1024 should pass, got: %v", err)
	}
}

func TestValidate_CaptionOverLimit(t *testing.T) {
	text := strings.Repeat("a", 1025)
	err := Validate(text, true)
	if err == nil {
		t.Fatal("caption at 1025 should fail, got nil")
	}
	if !strings.Contains(err.Error(), "1025") {
		t.Errorf("error should mention actual length, got: %v", err)
	}
}

func TestValidate_EmptyText(t *testing.T) {
	if err := Validate("", false); err != nil {
		t.Errorf("empty text should pass, got: %v", err)
	}
}

func TestValidate_EmptyCaption(t *testing.T) {
	if err := Validate("", true); err != nil {
		t.Errorf("empty caption should pass, got: %v", err)
	}
}

package cmd

import (
	"bytes"
	"testing"
)

func TestPreview_OutputsConvertedMarkdown(t *testing.T) {
	mdFile := writeTempMD(t, "Hello **bold** and *italic*!")

	var buf bytes.Buffer
	err := RunPreview(mdFile, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Fatal("expected output, got empty string")
	}
	// Should contain Telegram bold markers
	if !containsString(output, "*bold*") {
		t.Errorf("expected bold markers in output:\n%s", output)
	}
	// Should contain Telegram italic markers
	if !containsString(output, "_italic_") {
		t.Errorf("expected italic markers in output:\n%s", output)
	}
}

func TestPreview_MissingFile(t *testing.T) {
	var buf bytes.Buffer
	err := RunPreview("/nonexistent/post.md", &buf)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestPreview_NoHTTPCalls(t *testing.T) {
	// Preview should never make HTTP calls.
	// If it did, it would need config with bot token, but we don't provide one.
	// This test ensures preview only needs the markdown file.
	mdFile := writeTempMD(t, "Just a preview test.")

	var buf bytes.Buffer
	err := RunPreview(mdFile, &buf)
	if err != nil {
		t.Fatalf("preview should work without any config: %v", err)
	}
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

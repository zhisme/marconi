package converter

import (
	"os"
	"testing"

	"github.com/gotd/td/tg"
)

func TestConvertToEntities_EmptyInput(t *testing.T) {
	text, entities, err := ConvertToEntities([]byte(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "" {
		t.Errorf("expected empty text, got %q", text)
	}
	if len(entities) != 0 {
		t.Errorf("expected no entities, got %d", len(entities))
	}
}

func TestConvertToEntities_PlainText(t *testing.T) {
	text, entities, err := ConvertToEntities([]byte("Hello world"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "Hello world" {
		t.Errorf("text = %q, want %q", text, "Hello world")
	}
	if len(entities) != 0 {
		t.Errorf("plain text should produce no entities, got %d", len(entities))
	}
}

func TestConvertToEntities_Bold(t *testing.T) {
	text, entities, err := ConvertToEntities([]byte("**bold text**"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "bold text" {
		t.Errorf("text = %q, want %q", text, "bold text")
	}
	if len(entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(entities))
	}
	bold, ok := entities[0].(*tg.MessageEntityBold)
	if !ok {
		t.Fatalf("expected MessageEntityBold, got %T", entities[0])
	}
	if bold.Offset != 0 || bold.Length != 9 {
		t.Errorf("bold entity = {Offset: %d, Length: %d}, want {0, 9}", bold.Offset, bold.Length)
	}
}

func TestConvertToEntities_Italic(t *testing.T) {
	text, entities, err := ConvertToEntities([]byte("*italic*"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "italic" {
		t.Errorf("text = %q, want %q", text, "italic")
	}
	if len(entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(entities))
	}
	italic, ok := entities[0].(*tg.MessageEntityItalic)
	if !ok {
		t.Fatalf("expected MessageEntityItalic, got %T", entities[0])
	}
	if italic.Offset != 0 || italic.Length != 6 {
		t.Errorf("italic entity = {Offset: %d, Length: %d}, want {0, 6}", italic.Offset, italic.Length)
	}
}

func TestConvertToEntities_Strikethrough(t *testing.T) {
	text, entities, err := ConvertToEntities([]byte("~~deleted~~"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "deleted" {
		t.Errorf("text = %q, want %q", text, "deleted")
	}
	if len(entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(entities))
	}
	strike, ok := entities[0].(*tg.MessageEntityStrike)
	if !ok {
		t.Fatalf("expected MessageEntityStrike, got %T", entities[0])
	}
	if strike.Offset != 0 || strike.Length != 7 {
		t.Errorf("strike entity = {Offset: %d, Length: %d}, want {0, 7}", strike.Offset, strike.Length)
	}
}

func TestConvertToEntities_InlineCode(t *testing.T) {
	text, entities, err := ConvertToEntities([]byte("use `fmt.Println` here"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "use fmt.Println here" {
		t.Errorf("text = %q, want %q", text, "use fmt.Println here")
	}
	if len(entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(entities))
	}
	code, ok := entities[0].(*tg.MessageEntityCode)
	if !ok {
		t.Fatalf("expected MessageEntityCode, got %T", entities[0])
	}
	if code.Offset != 4 || code.Length != 11 {
		t.Errorf("code entity = {Offset: %d, Length: %d}, want {4, 11}", code.Offset, code.Length)
	}
}

func TestConvertToEntities_FencedCodeBlock(t *testing.T) {
	source := "```go\nfunc main() {}\n```"
	text, entities, err := ConvertToEntities([]byte(source))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "func main() {}" {
		t.Errorf("text = %q, want %q", text, "func main() {}")
	}
	if len(entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(entities))
	}
	pre, ok := entities[0].(*tg.MessageEntityPre)
	if !ok {
		t.Fatalf("expected MessageEntityPre, got %T", entities[0])
	}
	if pre.Language != "go" {
		t.Errorf("pre language = %q, want %q", pre.Language, "go")
	}
	if pre.Offset != 0 || pre.Length != 14 {
		t.Errorf("pre entity = {Offset: %d, Length: %d}, want {0, 14}", pre.Offset, pre.Length)
	}
}

func TestConvertToEntities_Link(t *testing.T) {
	text, entities, err := ConvertToEntities([]byte("[click here](https://example.com)"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "click here" {
		t.Errorf("text = %q, want %q", text, "click here")
	}
	if len(entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(entities))
	}
	link, ok := entities[0].(*tg.MessageEntityTextURL)
	if !ok {
		t.Fatalf("expected MessageEntityTextURL, got %T", entities[0])
	}
	if link.URL != "https://example.com" {
		t.Errorf("link URL = %q, want %q", link.URL, "https://example.com")
	}
	if link.Offset != 0 || link.Length != 10 {
		t.Errorf("link entity = {Offset: %d, Length: %d}, want {0, 10}", link.Offset, link.Length)
	}
}

func TestConvertToEntities_Heading(t *testing.T) {
	text, entities, err := ConvertToEntities([]byte("# My Heading"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "My Heading" {
		t.Errorf("text = %q, want %q", text, "My Heading")
	}
	if len(entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(entities))
	}
	bold, ok := entities[0].(*tg.MessageEntityBold)
	if !ok {
		t.Fatalf("expected MessageEntityBold for heading, got %T", entities[0])
	}
	if bold.Offset != 0 || bold.Length != 10 {
		t.Errorf("heading entity = {Offset: %d, Length: %d}, want {0, 10}", bold.Offset, bold.Length)
	}
}

func TestConvertToEntities_Blockquote(t *testing.T) {
	text, entities, err := ConvertToEntities([]byte("> quoted text"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "quoted text" {
		t.Errorf("text = %q, want %q", text, "quoted text")
	}
	if len(entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(entities))
	}
	bq, ok := entities[0].(*tg.MessageEntityBlockquote)
	if !ok {
		t.Fatalf("expected MessageEntityBlockquote, got %T", entities[0])
	}
	if bq.Offset != 0 || bq.Length != 11 {
		t.Errorf("blockquote entity = {Offset: %d, Length: %d}, want {0, 11}", bq.Offset, bq.Length)
	}
}

func TestConvertToEntities_MixedFormatting(t *testing.T) {
	source := "Hello **bold** and *italic*"
	text, entities, err := ConvertToEntities([]byte(source))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "Hello bold and italic" {
		t.Errorf("text = %q, want %q", text, "Hello bold and italic")
	}
	if len(entities) != 2 {
		t.Fatalf("expected 2 entities, got %d", len(entities))
	}

	bold, ok := entities[0].(*tg.MessageEntityBold)
	if !ok {
		t.Fatalf("expected MessageEntityBold, got %T", entities[0])
	}
	if bold.Offset != 6 || bold.Length != 4 {
		t.Errorf("bold = {Offset: %d, Length: %d}, want {6, 4}", bold.Offset, bold.Length)
	}

	italic, ok := entities[1].(*tg.MessageEntityItalic)
	if !ok {
		t.Fatalf("expected MessageEntityItalic, got %T", entities[1])
	}
	if italic.Offset != 15 || italic.Length != 6 {
		t.Errorf("italic = {Offset: %d, Length: %d}, want {15, 6}", italic.Offset, italic.Length)
	}
}

func TestConvertToEntities_NoEscaping(t *testing.T) {
	// Special chars should NOT be escaped (unlike MarkdownV2 converter)
	text, _, err := ConvertToEntities([]byte("Price is 100.50 (USD) #awesome!"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "Price is 100.50 (USD) #awesome!" {
		t.Errorf("text = %q, want plain text without escaping", text)
	}
}

func TestConvertToEntities_FullDocument(t *testing.T) {
	source, err := os.ReadFile("../testdata/formatted.md")
	if err != nil {
		t.Fatal(err)
	}

	text, entities, err := ConvertToEntities(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Plain text should contain unescaped content
	assertContains(t, text, "bold")
	assertContains(t, text, "italic")
	assertContains(t, text, "strikethrough")
	assertContains(t, text, "inline code")
	assertContains(t, text, "Heading One")
	assertContains(t, text, "bullet one")
	assertContains(t, text, "fmt.Println")

	// Should have multiple entities
	if len(entities) < 5 {
		t.Errorf("expected at least 5 entities for formatted.md, got %d", len(entities))
	}
}

func TestConvertToEntities_UTF16Offsets(t *testing.T) {
	// Emoji (U+1F600) is 2 UTF-16 code units
	text, entities, err := ConvertToEntities([]byte("😀**bold**"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "😀bold" {
		t.Errorf("text = %q, want %q", text, "😀bold")
	}
	if len(entities) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(entities))
	}
	bold := entities[0].(*tg.MessageEntityBold)
	// 😀 is 2 UTF-16 code units, so bold starts at offset 2
	if bold.Offset != 2 || bold.Length != 4 {
		t.Errorf("bold = {Offset: %d, Length: %d}, want {2, 4}", bold.Offset, bold.Length)
	}
}

func TestUTF16Len(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"hello", 5},
		{"日本語", 3},
		{"😀", 2},       // U+1F600 → surrogate pair
		{"a😀b", 4},     // 1 + 2 + 1
		{"hello 🌍", 8}, // 6 + 2
	}
	for _, tt := range tests {
		got := utf16Len(tt.input)
		if got != tt.want {
			t.Errorf("utf16Len(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

package converter

import (
	"os"
	"testing"
)

// --- High-level: full document conversion ---

func TestConvert_FullDocument(t *testing.T) {
	source, err := os.ReadFile("../testdata/formatted.md")
	if err != nil {
		t.Fatal(err)
	}

	result, err := Convert(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should contain bold markers (Telegram uses * for bold)
	assertContains(t, result, "*bold*")
	// Should contain italic markers (Telegram uses _ for italic)
	assertContains(t, result, "_italic_")
	// Should contain strikethrough
	assertContains(t, result, "~strikethrough~")
	// Should contain inline code
	assertContains(t, result, "`inline code`")
	// Should contain link
	assertContains(t, result, "[a link](https://example.com)")
	// Should contain blockquote prefix
	assertContains(t, result, ">")
	// Should contain fenced code block
	assertContains(t, result, "```go")
	// Should not contain raw markdown heading markers
	assertNotContains(t, result, "# Heading")
}

func TestConvert_EmptyInput(t *testing.T) {
	result, err := Convert([]byte(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty output, got %q", result)
	}
}

func TestConvert_PlainText_EscapesSpecialChars(t *testing.T) {
	source := []byte("Hello world! Price is 100.50 (USD) #awesome")
	result, err := Convert(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// These chars must be escaped: ! . ( ) #
	assertContains(t, result, `\!`)
	assertContains(t, result, `\.`)
	assertContains(t, result, `\(`)
	assertContains(t, result, `\)`)
	assertContains(t, result, `\#`)
}

// --- Escaping ---

func TestEscapeMarkdownV2_AllSpecialChars(t *testing.T) {
	special := `_ * [ ] ( ) ~ ` + "`" + ` > # + - = | { } . ! \`
	result := EscapeMarkdownV2(special)

	// Every special char should be preceded by backslash
	for _, ch := range []string{"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!", `\`} {
		assertContains(t, result, `\`+ch)
	}
}

func TestEscapeMarkdownV2_RegularText_Unchanged(t *testing.T) {
	input := "Hello world"
	result := EscapeMarkdownV2(input)
	if result != "Hello world" {
		t.Errorf("plain text should not change, got %q", result)
	}
}

func TestEscapeMarkdownV2_Unicode_PassThrough(t *testing.T) {
	input := "Привет мир 🎉 日本語"
	result := EscapeMarkdownV2(input)
	if result != input {
		t.Errorf("unicode should pass through, got %q", result)
	}
}

func TestEscapeCodeSpan_OnlyEscapesBacktickAndBackslash(t *testing.T) {
	input := "fmt.Println(`hello`) // test! #wow"
	result := EscapeCodeSpan(input)

	// Backtick and backslash should be escaped
	assertContains(t, result, "\\`")
	// Other special chars should NOT be escaped inside code
	assertNotContains(t, result, `\!`)
	assertNotContains(t, result, `\#`)
	assertNotContains(t, result, `\(`)
}

func TestEscapeURL_OnlyEscapesParenAndBackslash(t *testing.T) {
	input := "https://example.com/path(1)/page"
	result := EscapeURL(input)

	assertContains(t, result, `\)`)
	// Dots and slashes should NOT be escaped in URLs
	assertNotContains(t, result, `\.`)
}

// --- Node-level conversion ---

func TestConvert_Bold(t *testing.T) {
	result := mustConvert(t, []byte("**bold text**"))
	assertContains(t, result, "*bold text*")
}

func TestConvert_Italic(t *testing.T) {
	result := mustConvert(t, []byte("*italic text*"))
	assertContains(t, result, "_italic text_")
}

func TestConvert_Strikethrough(t *testing.T) {
	result := mustConvert(t, []byte("~~deleted~~"))
	assertContains(t, result, "~deleted~")
}

func TestConvert_InlineCode(t *testing.T) {
	result := mustConvert(t, []byte("use `fmt.Println` here"))
	assertContains(t, result, "`fmt.Println`")
}

func TestConvert_FencedCodeBlock(t *testing.T) {
	source := "```go\nfunc main() {}\n```"
	result := mustConvert(t, []byte(source))
	assertContains(t, result, "```go\n")
	assertContains(t, result, "func main()")
	assertContains(t, result, "\n```")
}

func TestConvert_FencedCodeBlock_NoLanguage(t *testing.T) {
	source := "```\nsome code\n```"
	result := mustConvert(t, []byte(source))
	assertContains(t, result, "```\n")
	assertContains(t, result, "some code")
}

func TestConvert_Link(t *testing.T) {
	result := mustConvert(t, []byte("[click here](https://example.com)"))
	assertContains(t, result, "[click here](https://example.com)")
}

func TestConvert_Link_WithParensInURL(t *testing.T) {
	result := mustConvert(t, []byte("[wiki](https://en.wikipedia.org/wiki/Go_(lang))"))
	// The closing paren in the URL should be escaped
	assertContains(t, result, `\)`)
}

func TestConvert_Blockquote(t *testing.T) {
	result := mustConvert(t, []byte("> quoted text"))
	assertContains(t, result, ">")
	assertContains(t, result, "quoted text")
}

func TestConvert_Heading_RenderedAsBold(t *testing.T) {
	result := mustConvert(t, []byte("# My Heading"))
	// Heading should render as bold (no # in Telegram)
	assertContains(t, result, "*My Heading*")
	assertNotContains(t, result, "# ")
}

func TestConvert_UnorderedList(t *testing.T) {
	source := "- item one\n- item two\n- item three"
	result := mustConvert(t, []byte(source))
	assertContains(t, result, "item one")
	assertContains(t, result, "item two")
	assertContains(t, result, "item three")
}

func TestConvert_OrderedList(t *testing.T) {
	source := "1. first\n2. second\n3. third"
	result := mustConvert(t, []byte(source))
	assertContains(t, result, "1")
	assertContains(t, result, "first")
	assertContains(t, result, "2")
	assertContains(t, result, "second")
}

func TestConvert_NestedFormatting(t *testing.T) {
	// Bold inside italic, etc.
	result := mustConvert(t, []byte("***bold and italic***"))
	// Should have both bold and italic markers
	assertContains(t, result, "*")
	assertContains(t, result, "_")
}

func TestConvert_Paragraph_Spacing(t *testing.T) {
	source := "First paragraph.\n\nSecond paragraph."
	result := mustConvert(t, []byte(source))
	assertContains(t, result, "First paragraph")
	assertContains(t, result, "Second paragraph")
	// Should have double newline between paragraphs
	assertContains(t, result, "\n\n")
}

func TestConvert_CodeBlock_NoInnerEscaping(t *testing.T) {
	// Special chars inside code blocks should NOT be fully escaped
	source := "```\nfoo.bar! #test (hello) *bold*\n```"
	result := mustConvert(t, []byte(source))
	// Inside code block, only ` and \ get escaped
	assertNotContains(t, result, `\!`)
	assertNotContains(t, result, `\#`)
	assertNotContains(t, result, `\*`)
}

func TestConvert_InlineCode_NoInnerEscaping(t *testing.T) {
	result := mustConvert(t, []byte("use `foo.bar!` ok"))
	// Inside inline code, only ` and \ get escaped
	assertNotContains(t, result, `\!`)
}

// --- Helpers ---

func mustConvert(t *testing.T, source []byte) string {
	t.Helper()
	result, err := Convert(source)
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	return result
}

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !containsStr(haystack, needle) {
		t.Errorf("expected output to contain %q, got:\n%s", needle, haystack)
	}
}

func assertNotContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if containsStr(haystack, needle) {
		t.Errorf("expected output NOT to contain %q, got:\n%s", needle, haystack)
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

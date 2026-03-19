package converter

import (
	"strings"

	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
)

type telegramRenderer struct {
	buf              strings.Builder
	source           []byte
	insideBlockquote bool
	listCounters     []int // stack: -1 = unordered, >0 = ordered counter
}

func newRenderer(source []byte) *telegramRenderer {
	return &telegramRenderer{source: source}
}

func (r *telegramRenderer) String() string {
	return strings.TrimRight(r.buf.String(), "\n")
}

func (r *telegramRenderer) walk(node ast.Node, entering bool) (ast.WalkStatus, error) {
	switch n := node.(type) {
	case *ast.Document:
		// noop

	case *ast.Paragraph:
		if !entering {
			if !isInsideListItem(n) {
				r.buf.WriteString("\n\n")
			}
			// Inside list items, paragraph exit is a noop —
			// ListItem handles the newline on exit.
		}

	case *ast.TextBlock:
		// TextBlock is used inside tight list items (no blank lines between items).
		// No extra newlines needed — ListItem handles spacing.

	case *ast.Heading:
		if entering {
			r.buf.WriteString("*")
		} else {
			r.buf.WriteString("*\n\n")
		}

	case *ast.Emphasis:
		if n.Level == 2 {
			r.buf.WriteString("*")
		} else {
			r.buf.WriteString("_")
		}

	case *east.Strikethrough:
		r.buf.WriteString("~")

	case *ast.CodeSpan:
		if entering {
			raw := extractCodeSpanContent(n, r.source)
			r.buf.WriteString("`")
			r.buf.WriteString(EscapeCodeSpan(raw))
			r.buf.WriteString("`")
			return ast.WalkSkipChildren, nil
		}

	case *ast.FencedCodeBlock:
		if entering {
			lang := string(n.Language(r.source))
			r.buf.WriteString("```")
			r.buf.WriteString(lang)
			r.buf.WriteString("\n")
			// Read raw code lines
			lines := n.Lines()
			for i := 0; i < lines.Len(); i++ {
				seg := lines.At(i)
				line := string(seg.Value(r.source))
				r.buf.WriteString(EscapeCodeSpan(line))
			}
			r.buf.WriteString("```\n\n")
			return ast.WalkSkipChildren, nil
		}

	case *ast.Blockquote:
		if entering {
			r.insideBlockquote = true
		} else {
			r.insideBlockquote = false
		}

	case *ast.Link:
		if entering {
			r.buf.WriteString("[")
		} else {
			url := EscapeURL(string(n.Destination))
			r.buf.WriteString("](")
			r.buf.WriteString(url)
			r.buf.WriteString(")")
		}

	case *ast.List:
		if entering {
			if n.IsOrdered() {
				r.listCounters = append(r.listCounters, n.Start)
			} else {
				r.listCounters = append(r.listCounters, -1)
			}
		} else {
			r.listCounters = r.listCounters[:len(r.listCounters)-1]
			r.buf.WriteString("\n")
		}

	case *ast.ListItem:
		if entering {
			if len(r.listCounters) > 0 {
				idx := len(r.listCounters) - 1
				if r.listCounters[idx] == -1 {
					r.buf.WriteString("• ")
				} else {
					r.buf.WriteString(itoa(r.listCounters[idx]))
					r.buf.WriteString("\\. ")
					r.listCounters[idx]++
				}
			}
		} else {
			r.buf.WriteString("\n")
		}

	case *ast.Text:
		if entering {
			text := string(n.Text(r.source))
			if r.insideBlockquote {
				lines := strings.Split(text, "\n")
				for i, line := range lines {
					if i > 0 {
						r.buf.WriteString("\n")
					}
					r.buf.WriteString(">")
					r.buf.WriteString(EscapeMarkdownV2(line))
				}
			} else {
				r.buf.WriteString(EscapeMarkdownV2(text))
			}
			if n.SoftLineBreak() {
				r.buf.WriteString("\n")
			}
			if n.HardLineBreak() {
				r.buf.WriteString("\n")
			}
		}

	case *ast.String:
		if entering {
			r.buf.WriteString(EscapeMarkdownV2(string(n.Value)))
		}
	}

	return ast.WalkContinue, nil
}

func isInsideListItem(n ast.Node) bool {
	parent := n.Parent()
	for parent != nil {
		if _, ok := parent.(*ast.ListItem); ok {
			return true
		}
		parent = parent.Parent()
	}
	return false
}

func extractCodeSpanContent(n *ast.CodeSpan, source []byte) string {
	var sb strings.Builder
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			sb.Write(t.Text(source))
		}
	}
	return sb.String()
}

func itoa(n int) string {
	if n < 0 {
		return "-" + uitoa(uint(-n))
	}
	return uitoa(uint(n))
}

func uitoa(n uint) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

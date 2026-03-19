package converter

import (
	"strings"

	"github.com/gotd/td/tg"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// ConvertToEntities converts Markdown source to plain text with Telegram MTProto entities.
// Unlike Convert(), this produces raw text + entity objects instead of MarkdownV2 escaped text.
func ConvertToEntities(source []byte) (string, []tg.MessageEntityClass, error) {
	if len(source) == 0 {
		return "", nil, nil
	}

	md := goldmark.New(goldmark.WithExtensions(extension.Strikethrough))
	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader)

	r := &entityRenderer{source: source}
	err := ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		return r.walk(node, entering)
	})
	if err != nil {
		return "", nil, err
	}

	return strings.TrimRight(r.buf.String(), "\n"), r.entities, nil
}

type entityRenderer struct {
	buf          strings.Builder
	source       []byte
	entities     []tg.MessageEntityClass
	utf16Off     int
	listCounters []int
	entityStack  []entityMark
}

type entityMark struct {
	offset int
	kind   string
	extra  string
}

func (r *entityRenderer) writeString(s string) {
	r.buf.WriteString(s)
	r.utf16Off += utf16Len(s)
}

func (r *entityRenderer) pushEntity(kind string) {
	r.entityStack = append(r.entityStack, entityMark{offset: r.utf16Off, kind: kind})
}

func (r *entityRenderer) pushEntityExtra(kind, extra string) {
	r.entityStack = append(r.entityStack, entityMark{offset: r.utf16Off, kind: kind, extra: extra})
}

func (r *entityRenderer) popEntity() {
	if len(r.entityStack) == 0 {
		return
	}
	mark := r.entityStack[len(r.entityStack)-1]
	r.entityStack = r.entityStack[:len(r.entityStack)-1]

	length := r.utf16Off - mark.offset
	if length <= 0 {
		return
	}

	var entity tg.MessageEntityClass
	switch mark.kind {
	case "bold":
		entity = &tg.MessageEntityBold{Offset: mark.offset, Length: length}
	case "italic":
		entity = &tg.MessageEntityItalic{Offset: mark.offset, Length: length}
	case "strike":
		entity = &tg.MessageEntityStrike{Offset: mark.offset, Length: length}
	case "blockquote":
		entity = &tg.MessageEntityBlockquote{Offset: mark.offset, Length: length}
	case "textUrl":
		entity = &tg.MessageEntityTextURL{Offset: mark.offset, Length: length, URL: mark.extra}
	}

	if entity != nil {
		r.entities = append(r.entities, entity)
	}
}

func (r *entityRenderer) walk(node ast.Node, entering bool) (ast.WalkStatus, error) {
	switch n := node.(type) {
	case *ast.Document:
		// noop

	case *ast.Paragraph:
		if !entering && !isInsideListItem(n) {
			r.writeString("\n\n")
		}

	case *ast.TextBlock:
		// noop — tight list items

	case *ast.Heading:
		if entering {
			r.pushEntity("bold")
		} else {
			r.popEntity()
			r.writeString("\n\n")
		}

	case *ast.Emphasis:
		if entering {
			if n.Level == 2 {
				r.pushEntity("bold")
			} else {
				r.pushEntity("italic")
			}
		} else {
			r.popEntity()
		}

	case *east.Strikethrough:
		if entering {
			r.pushEntity("strike")
		} else {
			r.popEntity()
		}

	case *ast.CodeSpan:
		if entering {
			raw := extractCodeSpanContent(n, r.source)
			offset := r.utf16Off
			r.writeString(raw)
			r.entities = append(r.entities, &tg.MessageEntityCode{
				Offset: offset,
				Length:  r.utf16Off - offset,
			})
			return ast.WalkSkipChildren, nil
		}

	case *ast.FencedCodeBlock:
		if entering {
			lang := string(n.Language(r.source))
			offset := r.utf16Off
			var code strings.Builder
			lines := n.Lines()
			for i := 0; i < lines.Len(); i++ {
				seg := lines.At(i)
				code.Write(seg.Value(r.source))
			}
			codeStr := strings.TrimRight(code.String(), "\n")
			r.writeString(codeStr)
			r.entities = append(r.entities, &tg.MessageEntityPre{
				Offset:   offset,
				Length:    r.utf16Off - offset,
				Language: lang,
			})
			r.writeString("\n\n")
			return ast.WalkSkipChildren, nil
		}

	case *ast.Blockquote:
		if entering {
			r.pushEntity("blockquote")
		} else {
			// Trim trailing newlines before closing entity so the
			// blockquote length doesn't include paragraph spacing.
			text := r.buf.String()
			trimmed := strings.TrimRight(text, "\n")
			r.buf.Reset()
			r.buf.WriteString(trimmed)
			r.utf16Off = utf16Len(trimmed)
			r.popEntity()
			r.writeString("\n\n")
		}

	case *ast.Link:
		if entering {
			r.pushEntityExtra("textUrl", string(n.Destination))
		} else {
			r.popEntity()
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
			r.writeString("\n")
		}

	case *ast.ListItem:
		if entering {
			if len(r.listCounters) > 0 {
				idx := len(r.listCounters) - 1
				if r.listCounters[idx] == -1 {
					r.writeString("• ")
				} else {
					r.writeString(itoa(r.listCounters[idx]) + ". ")
					r.listCounters[idx]++
				}
			}
		} else {
			r.writeString("\n")
		}

	case *ast.Text:
		if entering {
			t := string(n.Text(r.source))
			r.writeString(t)
			if n.SoftLineBreak() {
				r.writeString("\n")
			}
			if n.HardLineBreak() {
				r.writeString("\n")
			}
		}

	case *ast.String:
		if entering {
			r.writeString(string(n.Value))
		}
	}

	return ast.WalkContinue, nil
}

// utf16Len returns the number of UTF-16 code units needed to encode s.
// Telegram entity offsets and lengths are measured in UTF-16 code units.
func utf16Len(s string) int {
	n := 0
	for _, r := range s {
		if r >= 0x10000 {
			n += 2 // surrogate pair
		} else {
			n++
		}
	}
	return n
}

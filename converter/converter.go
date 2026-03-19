package converter

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

func Convert(source []byte) (string, error) {
	if len(source) == 0 {
		return "", nil
	}

	md := goldmark.New(
		goldmark.WithExtensions(
			extension.Strikethrough,
		),
	)

	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader)

	r := newRenderer(source)
	err := ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		return r.walk(node, entering)
	})
	if err != nil {
		return "", err
	}

	return r.String(), nil
}

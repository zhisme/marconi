package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/zhisme/marconi/converter"
)

func RunPreview(mdFile string, w io.Writer) error {
	source, err := os.ReadFile(mdFile)
	if err != nil {
		return fmt.Errorf("file not found: %s", mdFile)
	}

	converted, err := converter.Convert(source)
	if err != nil {
		return fmt.Errorf("failed to convert markdown: %w", err)
	}

	fmt.Fprintln(w, converted)
	return nil
}

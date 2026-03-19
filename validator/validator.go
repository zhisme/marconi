package validator

import "fmt"

const (
	MaxTextLength    = 4096
	MaxCaptionLength = 1024
)

func Validate(text string, hasImage bool) error {
	if hasImage {
		if len(text) > MaxCaptionLength {
			return fmt.Errorf("caption too long (%d/%d chars)", len(text), MaxCaptionLength)
		}
	} else {
		if len(text) > MaxTextLength {
			return fmt.Errorf("message too long (%d/%d chars)", len(text), MaxTextLength)
		}
	}
	return nil
}

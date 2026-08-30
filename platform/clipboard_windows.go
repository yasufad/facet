//go:build windows

package platform

import (
	"fmt"

	"github.com/yasufad/facet/third_party/w32"
)

// windowsClipboard is the Windows implementation of [Clipboard]. It wraps
// the w32 clipboard functions, which lock the OS thread for the duration of
// each call to avoid the OpenClipboard/CloseClipboard deadlock that occurs
// when a goroutine switches threads between them.
type windowsClipboard struct{}

// Text returns the current clipboard text, or an empty string if the
// clipboard does not contain text.
func (windowsClipboard) Text() (string, error) {
	text, err := w32.GetClipboardText()
	if err != nil {
		return "", fmt.Errorf("read clipboard: %w", err)
	}
	return text, nil
}

// SetText replaces the clipboard contents with text.
func (windowsClipboard) SetText(text string) error {
	if err := w32.SetClipboardText(text); err != nil {
		return fmt.Errorf("write clipboard: %w", err)
	}
	return nil
}

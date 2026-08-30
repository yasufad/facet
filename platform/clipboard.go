package platform

// Clipboard is the system clipboard. It handles text only; rich content
// (images, file lists) is out of scope for the interface and may be added per
// backend if a need arises.
//
// Methods may be called from any goroutine; a backend marshals onto the
// platform thread where the platform's clipboard API requires it.
type Clipboard interface {
	// SetText writes text to the clipboard, replacing whatever was there.
	SetText(text string) error

	// Text reads the current clipboard text. If the clipboard does not hold
	// text, the error distinguishes that from a failure to read.
	Text() (string, error)
}

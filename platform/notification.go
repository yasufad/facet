package platform

// Notification is a system notification — a banner or toast that the
// operating system displays, with optional activation handling.
type Notification struct {
	// Title is the heading line of the notification.
	Title string

	// Body is the detail text beneath the title.
	Body string

	// ActionHandler is called on the platform thread when the user activates
	// the notification (clicks the banner, selects it in Notification Center,
	// etc.). It is nil-safe.
	ActionHandler func()
}

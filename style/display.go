package style

// Display sets the layout strategy used for an element's children.
type Display uint8

const (
	// DisplayFlex lays children out according to the flexbox algorithm.
	DisplayFlex Display = iota
	// DisplayBlock lays children out according to standard block layout.
	DisplayBlock
	// DisplayNone removes the element from layout calculation entirely.
	DisplayNone
)

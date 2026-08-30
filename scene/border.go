package scene

// BorderStyle is the line style of a Quad's border. The renderer samples the
// border differently for each style.
type BorderStyle uint8

const (
	// BorderSolid draws a continuous border.
	BorderSolid BorderStyle = iota
	// BorderDashed draws a dashed border.
	BorderDashed
)

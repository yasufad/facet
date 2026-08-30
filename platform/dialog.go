package platform

// MessageDialog is a modal alert with buttons and an icon. It is shown by
// [Platform.ShowMessageDialog], which blocks until the user dismisses it and
// returns the button the user chose.
type MessageDialog struct {
	Title   string
	Message string
	Buttons MessageButtons
	Icon    MessageDialogIcon
}

// MessageButtons selects the button set a [MessageDialog] shows.
type MessageButtons int

const (
	ButtonsOK MessageButtons = iota
	ButtonsOKCancel
	ButtonsYesNo
	ButtonsYesNoCancel
)

// MessageDialogIcon selects the icon a [MessageDialog] shows.
type MessageDialogIcon int

const (
	IconInfo MessageDialogIcon = iota
	IconWarning
	IconError
	IconQuestion
)

// DialogResult is the button the user chose in a modal dialog.
type DialogResult int

const (
	ResultNone DialogResult = iota
	ResultOK
	ResultCancel
	ResultYes
	ResultNo
)

// FileFilter restricts a file dialog to a set of extensions. Name is the
// human-readable label ("Text files"); Extensions lists the extensions
// without a leading dot (".txt" is written as "txt").
type FileFilter struct {
	Name       string
	Extensions []string
}

// OpenFileDialog is a modal file-open dialog, shown by
// [Platform.ShowOpenDialog].
type OpenFileDialog struct {
	Title     string
	Directory string // initial directory; empty means the platform default
	Filters   []FileFilter
	Multiple  bool // allow selecting more than one file
}

// SaveFileDialog is a modal file-save dialog, shown by
// [Platform.ShowSaveDialog].
type SaveFileDialog struct {
	Title       string
	Directory   string // initial directory; empty means the platform default
	DefaultName string // suggested filename
	Filters     []FileFilter
}

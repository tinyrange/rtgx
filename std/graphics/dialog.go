package graphics

// FileDialogOptions configures a native single-file open or save dialog.
type FileDialogOptions struct {
	Title            string
	InitialDirectory string
	DefaultName      string
}

// FolderDialogOptions configures a native directory chooser. Native choosers
// expose their normal create-folder affordance where the host supports one.
type FolderDialogOptions struct {
	Title            string
	InitialDirectory string
}

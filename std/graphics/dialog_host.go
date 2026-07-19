//go:build !renvo

package graphics

// Host graphics windows are deliberately headless for deterministic renderer
// tests, so their dialogs behave like a user cancellation.
func (w *Window) OpenFileDialog(options FileDialogOptions) (string, bool) {
	return "", false
}

func (w *Window) SaveFileDialog(options FileDialogOptions) (string, bool) {
	return "", false
}

func (w *Window) SelectFolderDialog(options FolderDialogOptions) (string, bool) {
	return "", false
}

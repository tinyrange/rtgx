package main

import (
	"os"
	"path/filepath"
	"testing"

	"j5.nz/rtg/rtg/std/graphics"
)

func TestMainFormGeneratedLayoutAndOpenSaveCallbacks(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "main.go")
	if err := os.WriteFile(path, []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	form := NewMainForm(root)
	controls := form.Controls()
	if len(controls) != 2 || controls[0] != &form.explorer.Control || controls[1] != &form.editor.Control {
		t.Fatalf("generated controls = %#v", controls)
	}
	if form.explorer.Font == nil || form.explorer.Font != form.editor.Font {
		t.Fatal("generated form did not share one UI font between its panes")
	}
	form.explorerOpenFile(path)
	if form.currentPath != path || form.editor.Document.Text() != "package main\n" {
		t.Fatalf("opened state = %q, %q", form.currentPath, form.editor.Document.Text())
	}
	form.editor.Document.MoveDocumentEnd(false)
	form.editor.Document.Insert("// saved\n")
	form.saveCurrentFile()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "package main\n// saved\n" || form.editor.Document.Dirty() {
		t.Fatalf("saved state = %q, dirty %v", data, form.editor.Document.Dirty())
	}
}

func TestMainFormRendersAndResizesOnlyItsPanes(t *testing.T) {
	form := NewMainForm(t.TempDir())
	surface := graphics.NewSurface(1000, 700)
	if !form.Paint(surface) {
		t.Fatal("initial form did not paint")
	}
	if _, ok := surface.DirtyRect(); !ok {
		t.Fatal("initial form paint produced no pixels")
	}
	form.Dispatch(graphics.Event{Type: graphics.EventWindowResize, Dirty: graphics.R(0, 0, 720, 480)})
	if form.explorer.Bounds() != graphics.R(0, 0, 260, 480) {
		t.Fatalf("explorer bounds = %#v", form.explorer.Bounds())
	}
	if form.editor.Bounds() != graphics.R(260, 0, 460, 480) {
		t.Fatalf("editor bounds = %#v", form.editor.Bounds())
	}
}

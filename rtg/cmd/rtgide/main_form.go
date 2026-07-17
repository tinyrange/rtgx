package main

import (
	"j5.nz/rtg/rtg/forms"
	"j5.nz/rtg/rtg/ide"
	rtgos "j5.nz/rtg/rtg/std/os"
)

// MainForm contains one IDE window. main_form_generated.go owns construction
// and property assignment; this file contains application state and callbacks.
type MainForm struct {
	forms.Form
	explorer    *ide.ExplorerControl
	editor      *ide.EditorControl
	currentPath string
}

func NewMainForm(root string) *MainForm {
	form := &MainForm{}
	form.initializeComponent(root)
	return form
}

func (f *MainForm) explorerOpenFile(path string) {
	data, err := rtgos.ReadFile(path)
	if err != nil {
		return
	}
	f.currentPath = path
	f.editor.SetDocument(ide.NewDocument(data))
}

func (f *MainForm) saveCurrentFile() {
	if f.currentPath == "" || f.editor.Document == nil || !f.editor.Document.Dirty() {
		return
	}
	if rtgos.WriteFile(f.currentPath, f.editor.Document.Bytes(), 0644) == nil {
		f.editor.Document.MarkSaved()
		f.editor.Invalidate()
	}
}

func (f *MainForm) formResize() {
	width, height := f.Size()
	explorerWidth := 260
	if width < 520 {
		explorerWidth = width / 3
	}
	f.explorer.SetBounds(rect(0, 0, explorerWidth, height))
	f.editor.SetBounds(rect(explorerWidth, 0, width-explorerWidth, height))
}

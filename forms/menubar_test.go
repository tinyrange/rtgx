package forms

import (
	"testing"

	"renvo.dev/std/graphics"
)

func TestMenuBarShortcutRunsBeforeFocusedControl(t *testing.T) {
	var form Form
	form.Initialize(360, 180)
	box := NewTextBox()
	box.SetBounds(graphics.R(10, 50, 240, 30))
	box.SetFont(graphics.NewBuiltinFont(1))
	menu := NewMenuBar()
	menu.SetBounds(graphics.R(0, 0, 220, 36))
	menu.SetFont(graphics.NewBuiltinFont(1))
	file := NewMenu("File")
	save := NewMenuItem("Save")
	save.SetShortcut(Shortcut{Key: graphics.KeyS, Primary: true, Text: "Ctrl/Cmd+S"})
	called := 0
	save.Activate = func() { called++ }
	file.Add(save)
	menu.Add(file)
	form.Add(&box.Control)
	form.Add(&menu.Control)
	form.SetMenuBar(menu)
	box.Focus()

	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyS, Modifiers: graphics.ModifierControl})
	if called != 1 || !box.Focused() {
		t.Fatalf("shortcut state = called %d focused %v", called, box.Focused())
	}
	save.SetEnabled(false)
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyS, Modifiers: graphics.ModifierCommand})
	if called != 1 {
		t.Fatalf("disabled shortcut called %d times", called)
	}
}

func TestMenuBarOpensSelectsAndDismissesWithoutOwningFocus(t *testing.T) {
	var form Form
	form.Initialize(360, 180)
	menu := NewMenuBar()
	menu.SetBounds(graphics.R(10, 5, 220, 34))
	menu.SetFont(graphics.NewBuiltinFont(1))
	file := NewMenu("File")
	open := NewMenuItem("Open...")
	called := 0
	open.Activate = func() { called++ }
	file.Add(open)
	menu.Add(file)
	form.Add(&menu.Control)
	form.SetMenuBar(menu)

	closed := menu.Bounds()
	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: 25, Y: 20})
	if !menu.Open() || menu.Bounds().Height() <= closed.Height() {
		t.Fatalf("menu did not expand: open %v bounds %#v", menu.Open(), menu.Bounds())
	}
	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: 30, Y: 55})
	if called != 1 || menu.Open() || menu.Bounds() != closed {
		t.Fatalf("selection state = called %d open %v bounds %#v", called, menu.Open(), menu.Bounds())
	}

	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: 25, Y: 20})
	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: 300, Y: 150})
	if menu.Open() {
		t.Fatal("outside click did not dismiss menu")
	}
}

func TestMenuBarKeyboardNavigationActivatesSelectedItem(t *testing.T) {
	var form Form
	form.Initialize(360, 180)
	menu := NewMenuBar()
	menu.SetBounds(graphics.R(0, 0, 220, 34))
	menu.SetFont(graphics.NewBuiltinFont(1))
	file := NewMenu("File")
	file.Add(NewMenuItem("Open"))
	save := NewMenuItem("Save")
	called := 0
	save.Activate = func() { called++ }
	file.Add(save)
	menu.Add(file)
	form.Add(&menu.Control)
	form.SetMenuBar(menu)

	form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: 15, Y: 15})
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyDown})
	form.Dispatch(graphics.Event{Type: graphics.EventKeyDown, Key: graphics.KeyEnter})
	if called != 1 || menu.Open() {
		t.Fatalf("keyboard selection = called %d open %v", called, menu.Open())
	}
}

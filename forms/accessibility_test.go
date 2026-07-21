package forms

import (
	"testing"

	"renvo.dev/std/graphics"
)

func TestAccessibilityUpdatesTrackOnlySemanticMutations(t *testing.T) {
	var form Form
	form.Initialize(320, 180)
	button := NewButton()
	button.SetAccessibilityID("save")
	button.SetText("Save")
	button.SetBounds(graphics.R(12, 16, 90, 30))
	label := NewLabel()
	label.SetAccessibilityID("status")
	label.SetText("Ready")
	label.SetBounds(graphics.R(12, 56, 120, 24))
	form.Add(&button.Control)
	form.Add(&label.Control)

	initial, ok := form.TakeAccessibilityUpdate()
	if !ok || !initial.Reset || len(initial.Nodes) != 2 {
		t.Fatalf("initial update = %#v, %v", initial, ok)
	}
	if initial.Nodes[0].ID != "save" || initial.Nodes[0].Role != AccessibilityRoleButton || initial.Nodes[0].Name != "Save" {
		t.Fatalf("button semantics = %#v", initial.Nodes[0])
	}

	button.SetBounds(graphics.R(20, 16, 90, 30))
	button.SetText("Save all")
	update, ok := form.TakeAccessibilityUpdate()
	if !ok || update.Reset || len(update.Nodes) != 1 || update.Nodes[0].ID != "save" {
		t.Fatalf("incremental update = %#v, %v", update, ok)
	}
	if _, ok := form.TakeAccessibilityUpdate(); ok {
		t.Fatal("unchanged tree produced another update")
	}

	form.Remove(&label.Control)
	removed, ok := form.TakeAccessibilityUpdate()
	if !ok || len(removed.Removed) != 1 || removed.Removed[0] != "status" {
		t.Fatalf("removal update = %#v, %v", removed, ok)
	}
}

func TestAutomationDriverUsesControlActions(t *testing.T) {
	var form Form
	form.Initialize(320, 180)
	box := NewTextBox()
	box.SetAccessibilityID("project-name")
	box.SetAccessibilityName("Project name")
	check := NewCheckBox()
	check.SetAccessibilityID("enabled")
	check.SetText("Enabled")
	clicks := 0
	check.Click = func() { clicks++ }
	form.Add(&box.Control)
	form.Add(&check.Control)
	driver := NewAutomationDriver(&form)

	if !driver.SetValue("project-name", "hello") || box.Text() != "hello" {
		t.Fatalf("text value = %q", box.Text())
	}
	if !driver.Focus("project-name") || !box.Focused() {
		t.Fatal("text box did not receive semantic focus")
	}
	if !driver.Invoke("enabled") || !check.Checked() || clicks != 1 {
		t.Fatalf("checkbox state = checked %v clicks %d", check.Checked(), clicks)
	}
	found := driver.Find(AccessibilityRoleTextBox, "Project name")
	if len(found) != 1 || found[0].Value != "hello" || found[0].Actions&AccessibilitySupportsSetValue == 0 {
		t.Fatalf("text box query = %#v", found)
	}
}

func TestSelectionChangesDoNotRetransmitTextValues(t *testing.T) {
	var form Form
	form.Initialize(320, 180)
	editor := NewControl()
	editor.SetAccessibilityID("editor")
	editor.SetAccessibilityRole(AccessibilityRoleTextBox)
	editor.SetText("a large document value")
	start, end := 0, 0
	editor.AccessibilitySelectionStart = func() int { return start }
	editor.AccessibilitySelectionEnd = func() int { return end }
	editor.AccessibilitySetSelection = func(nextStart, nextEnd int) {
		start, end = nextStart, nextEnd
	}
	form.Add(editor)
	form.TakeAccessibilityUpdate()

	driver := NewAutomationDriver(&form)
	if !driver.SetSelection("editor", 2, 7) || start != 2 || end != 7 {
		t.Fatalf("selection = %d:%d", start, end)
	}
	update, ok := form.TakeAccessibilityUpdate()
	if !ok || len(update.Nodes) != 0 || len(update.Selections) != 1 || update.Selections[0].ID != "editor" {
		t.Fatalf("selection update = %#v, %v", update, ok)
	}
}

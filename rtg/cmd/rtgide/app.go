package main

import "j5.nz/rtg/rtg/std/graphics"

func run(args []string) int {
	root := "."
	if len(args) > 1 && args[1] != "" {
		root = args[1]
	}
	window := graphics.NewWindow(graphics.WindowOptions{Title: "RTG Forms", Width: 1000, Height: 700})
	if window == nil {
		return 1
	}
	form := NewMainForm(root)
	for {
		if form.Paint(window.Surface()) {
			if !window.Present() {
				window.Close()
				return 1
			}
		}
		event, ok := window.Wait()
		if !ok {
			window.Close()
			return 0
		}
		if event.Type == graphics.EventWindowClose {
			window.Close()
			return 0
		}
		form.Dispatch(event)
	}
}

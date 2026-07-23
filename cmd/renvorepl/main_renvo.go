//go:build renvo

package main

import (
	"renvo.dev/internal/repl"
	"renvo.dev/std/strconv"
)

const replHelp = `Renvo experimental REPL
Enter expressions, statements, imports, or declarations.
Expressions are printed automatically. Variables, functions, and closures are
linked into the live session; earlier statements are never executed again.

Commands:
  :help     show this help
  :history  show accepted submissions
  :source   show the current linked-generation source
  :reset    clear retained session state
  :quit     exit the REPL

Line editing:
  Left/Right and Home/End move the cursor; Up/Down browse history.
  Tab completes symbols and imports; inside a call it shows the active argument.
  Repeating Tab cycles ambiguous semantic matches.
  Ctrl-A/E move to the start/end, Ctrl-W deletes a word, and Ctrl-C cancels.
`

func appMain(args []string, env []string) int {
	configureRunPlatform()
	if len(args) > 1 {
		if len(args) == 2 && (args[1] == "-h" || args[1] == "--help") {
			print(replHelp)
			return 0
		}
		print("usage: renvorepl\n")
		return 1
	}
	print("Renvo experimental REPL. Type :help for help.\n")
	var state repl.State
	var session repl.Session
	var input replInput
	input.open()
	pending := ""
	for {
		prompt := "renvo> "
		if pending == "" {
			prompt = "renvo> "
		} else {
			prompt = "...> "
		}
		line, ok := input.line(prompt, &state, pending, env)
		if !ok {
			if pending != "" {
				replSubmit(&state, &session, pending, env)
			}
			session.Reset()
			input.close()
			print("\n")
			return 0
		}
		if input.cancelled() {
			pending = ""
			continue
		}
		clean := repl.CleanInput(line)
		if pending == "" && len(clean) > 0 && clean[0] == ':' {
			if replCommand(&state, &session, clean) {
				session.Reset()
				input.close()
				return 0
			}
			continue
		}
		if pending != "" {
			pending = pending + "\n" + line
		} else {
			pending = line
		}
		if !repl.InputComplete(pending) {
			continue
		}
		replSubmit(&state, &session, pending, env)
		pending = ""
	}
}

func replCommand(state *repl.State, session *repl.Session, command string) bool {
	if command == ":quit" || command == ":q" || command == ":exit" {
		return true
	}
	if command == ":help" || command == ":h" {
		print(replHelp)
		return false
	}
	if command == ":reset" || command == ":clear" {
		state.Reset()
		session.Reset()
		print("session reset\n")
		return false
	}
	if command == ":source" {
		print(state.Source())
		return false
	}
	if command == ":history" {
		history := state.History()
		for i := 0; i < len(history); i++ {
			print(strconv.Itoa(i + 1))
			print("  ")
			print(history[i])
			print("\n")
		}
		return false
	}
	print("unknown command: ")
	print(command)
	print("\n")
	return false
}

func replSubmit(state *repl.State, session *repl.Session, input string, env []string) {
	prepared := state.Prepare(input)
	if prepared.Kind == repl.SubmissionInvalid {
		if prepared.Error != "" {
			print("renvorepl: ")
			print(prepared.Error)
			print("\n")
		}
		return
	}
	result := session.Evaluate(prepared.First, env)
	attempt := 0
	if !result.Compiled && len(prepared.Second) > 0 {
		result = session.Evaluate(prepared.Second, env)
		attempt = 1
	}
	if !result.Compiled {
		print(string(result.Diagnostic))
		return
	}
	if result.ExitCode != 0 {
		print("renvorepl: program exited with status ")
		print(strconv.Itoa(result.ExitCode))
		print("\n")
		return
	}
	state.Accept(prepared, attempt)
}

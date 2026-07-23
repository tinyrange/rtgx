//go:build renvo

package main

import (
	"renvo.dev/internal/repl"
	"renvo.dev/std/strconv"
)

type replInput struct {
	pending []byte
	eof     bool
	raw     bool
	history []string
	cancel  bool
}

type replLine struct {
	data         []byte
	cursor       int
	historyIndex int
	saved        string
	complete     replLineCompletion
}

type replLineCompletion struct {
	active bool
	head   []byte
	tail   []byte
	names  [][]byte
	index  int
}

func (r *replInput) open() {
	r.raw = replTerminalEnable()
}

func (r *replInput) close() {
	if r.raw {
		replTerminalDisable()
		r.raw = false
	}
}

func (r *replInput) line(prompt string, state *repl.State, pending string, env []string) (string, bool) {
	if !r.raw {
		return r.bufferedLine(prompt)
	}
	print(prompt)
	line := replLine{historyIndex: len(r.history)}
	for {
		ch, ok := r.nextByte()
		if !ok {
			print("\n")
			if len(line.data) == 0 {
				return "", false
			}
			return string(line.data), true
		}
		changed := false
		if ch != 9 {
			line.complete.active = false
		}
		if ch == '\r' || ch == '\n' {
			if ch == '\r' && len(r.pending) > 0 && r.pending[0] == '\n' {
				r.pending = r.pending[1:]
			}
			print("\n")
			value := string(line.data)
			if value != "" && (len(r.history) == 0 || r.history[len(r.history)-1] != value) {
				r.history = append(r.history, value)
			}
			return value, true
		}
		if ch == 1 {
			line.cursor = 0
			changed = true
		} else if ch == 2 {
			line.moveLeft()
			changed = true
		} else if ch == 3 {
			print("^C\n")
			r.cancel = true
			return "", true
		} else if ch == 4 {
			if len(line.data) == 0 {
				print("\n")
				return "", false
			}
			line.delete()
			changed = true
		} else if ch == 5 {
			line.cursor = len(line.data)
			changed = true
		} else if ch == 6 {
			line.moveRight()
			changed = true
		} else if ch == 11 {
			line.data = line.data[:line.cursor]
			changed = true
		} else if ch == 12 {
			print("\x1b[2J\x1b[H")
			changed = true
		} else if ch == 14 {
			line.historyDown(r.history)
			changed = true
		} else if ch == 16 {
			line.historyUp(r.history)
			changed = true
		} else if ch == 21 {
			line.data = append(line.data[:0], line.data[line.cursor:]...)
			line.cursor = 0
			changed = true
		} else if ch == 23 {
			line.deleteWord()
			changed = true
		} else if ch == 8 || ch == 127 {
			line.backspace()
			changed = true
		} else if ch == 9 {
			changed = r.completeLine(&line, prompt, state, pending, env)
		} else if ch == 27 {
			changed = r.escape(&line)
		} else if ch >= 32 {
			line.insert(ch)
			changed = true
		}
		if changed {
			replRedraw(prompt, line.data, line.cursor)
		}
	}
}

func (r *replInput) completeLine(line *replLine, prompt string, state *repl.State, pending string, env []string) bool {
	if line.complete.active && len(line.complete.names) > 0 {
		line.complete.index++
		if line.complete.index >= len(line.complete.names) {
			line.complete.index = 0
		}
		line.applyCompletion(line.complete.index)
		return true
	}

	start := line.cursor
	for start > 0 && replIdentifierByte(line.data[start-1]) {
		start--
	}
	if start > 0 && line.data[start-1] == ':' && start-1 == replFirstNonSpace(line.data) {
		start--
	}
	prefix := string(line.data[start:line.cursor])
	var items []repl.Completion
	if len(prefix) > 0 && prefix[0] == ':' {
		items = replCommandCompletions(prefix)
	} else {
		input := string(line.data)
		cursor := line.cursor
		if pending != "" {
			input = pending + "\n" + input
			cursor += len(pending) + 1
		}
		if prefix == "" {
			help := state.Signature(input, cursor, env)
			if help.Ok {
				replPrintSignature(help)
				return true
			}
		} else {
			probe := make([]byte, 0, len(input)+1)
			probe = append(probe, input[:cursor]...)
			probe = append(probe, '(')
			probe = append(probe, input[cursor:]...)
			help := state.Signature(string(probe), cursor+1, env)
			if help.Ok {
				replPrintSignature(help)
				return true
			}
		}
		items = state.Complete(input, cursor, env)
		if len(items) > 0 && items[0].ReplaceStart >= 0 {
			lineStart := 0
			if pending != "" {
				lineStart = len(pending) + 1
			}
			replacement := items[0].ReplaceStart - lineStart
			if replacement >= 0 && replacement <= line.cursor {
				start = replacement
			}
		}
	}
	if len(items) == 0 {
		if replOnlySpace(line.data[:line.cursor]) {
			for i := 0; i < 4; i++ {
				line.insert(' ')
			}
			return true
		}
		print("\a")
		return false
	}

	line.complete = replLineCompletion{
		active: true,
		head:   replCopyBytes(line.data[:start]),
		tail:   replCopyBytes(line.data[line.cursor:]),
		names:  make([][]byte, 0, len(items)),
	}
	for i := 0; i < len(items); i++ {
		insert := items[i].Insert
		if insert == "" {
			insert = items[i].Name
		}
		line.complete.names = append(line.complete.names, replCopyBytes([]byte(insert)))
	}
	line.applyCompletion(0)
	if len(items) > 1 || items[0].Signature != "" {
		replPrintCompletions(items)
	}
	return true
}

func (line *replLine) applyCompletion(index int) {
	if index < 0 || index >= len(line.complete.names) {
		return
	}
	data := make([]byte, 0, len(line.complete.head)+len(line.complete.names[index])+len(line.complete.tail))
	data = append(data, line.complete.head...)
	data = append(data, line.complete.names[index]...)
	line.cursor = len(data)
	data = append(data, line.complete.tail...)
	line.data = data
}

func replCommandCompletions(prefix string) []repl.Completion {
	commands := []string{":help", ":history", ":source", ":reset", ":quit", ":exit"}
	var out []repl.Completion
	for i := 0; i < len(commands); i++ {
		if replHasPrefix(commands[i], prefix) && commands[i] != prefix {
			out = append(out, repl.Completion{Name: commands[i], Detail: "REPL command"})
		}
	}
	return out
}

func replPrintCompletions(items []repl.Completion) {
	print("\n")
	limit := len(items)
	if limit > 12 {
		limit = 12
	}
	detailed := false
	for i := 0; i < limit; i++ {
		if items[i].Signature != "" {
			detailed = true
			break
		}
	}
	for i := 0; i < limit; i++ {
		if detailed {
			if items[i].Signature != "" {
				print(items[i].Signature)
			} else {
				print(items[i].Name)
				if items[i].Detail != "" {
					print(" - ")
					print(items[i].Detail)
				}
			}
			print("\n")
		} else {
			if i > 0 {
				print("  ")
			}
			print(items[i].Name)
		}
	}
	if !detailed {
		if limit < len(items) {
			print("  ...")
		}
		print("\n")
	} else if limit < len(items) {
		print("...\n")
	}
	if detailed && limit == 0 {
		print("\n")
	}
}

func replPrintSignature(help repl.SignatureHelp) {
	print("\n")
	print(help.Label)
	print("\n")
	active := help.ActiveParameter
	if active < 0 || active >= len(help.Parameters) {
		return
	}
	print("argument ")
	print(strconv.Itoa(active + 1))
	print(": ")
	if help.Parameters[active].Name != "" {
		print(help.Parameters[active].Name)
		if help.Parameters[active].Type != "" {
			print(" ")
		}
	}
	print(help.Parameters[active].Type)
	print("\n")
}

func replCopyBytes(data []byte) []byte {
	out := make([]byte, len(data))
	copy(out, data)
	return out
}

func replIdentifierByte(ch byte) bool {
	return ch == '_' || ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9'
}

func replFirstNonSpace(data []byte) int {
	for i := 0; i < len(data); i++ {
		if data[i] != ' ' && data[i] != '\t' {
			return i
		}
	}
	return len(data)
}

func replOnlySpace(data []byte) bool {
	return replFirstNonSpace(data) == len(data)
}

func replHasPrefix(value string, prefix string) bool {
	if len(prefix) > len(value) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if value[i] != prefix[i] {
			return false
		}
	}
	return true
}

func (r *replInput) cancelled() bool {
	if !r.cancel {
		return false
	}
	r.cancel = false
	return true
}

func (r *replInput) bufferedLine(prompt string) (string, bool) {
	print(prompt)
	for {
		for i := 0; i < len(r.pending); i++ {
			if r.pending[i] == '\n' {
				line := string(r.pending[:i])
				r.pending = r.pending[i+1:]
				if len(line) > 0 && line[len(line)-1] == '\r' {
					line = line[:len(line)-1]
				}
				return line, true
			}
		}
		if r.eof {
			if len(r.pending) == 0 {
				return "", false
			}
			line := string(r.pending)
			r.pending = nil
			return line, true
		}
		buf := make([]byte, 4096)
		n := read(0, buf, -1)
		if n <= 0 {
			r.eof = true
			continue
		}
		r.pending = append(r.pending, buf[:n]...)
	}
}

func (r *replInput) nextByte() (byte, bool) {
	for len(r.pending) == 0 {
		buf := make([]byte, 64)
		n := read(0, buf, -1)
		if n <= 0 {
			return 0, false
		}
		r.pending = append(r.pending, buf[:n]...)
	}
	ch := r.pending[0]
	r.pending = r.pending[1:]
	return ch, true
}

func (r *replInput) escape(line *replLine) bool {
	first, ok := r.nextByte()
	if !ok {
		return false
	}
	if first == 'b' || first == 'B' {
		line.wordLeft()
		return true
	}
	if first == 'f' || first == 'F' {
		line.wordRight()
		return true
	}
	if first == 'O' {
		last, lastOK := r.nextByte()
		if !lastOK {
			return false
		}
		return replApplyEscape(line, last, 0, r.history)
	}
	if first != '[' {
		return false
	}
	number := 0
	for count := 0; count < 12; count++ {
		last, lastOK := r.nextByte()
		if !lastOK {
			return false
		}
		if last >= '0' && last <= '9' {
			number = number*10 + int(last-'0')
			continue
		}
		if last == ';' {
			number = 0
			continue
		}
		return replApplyEscape(line, last, number, r.history)
	}
	return false
}

func replApplyEscape(line *replLine, last byte, number int, history []string) bool {
	if last == 'A' {
		line.historyUp(history)
		return true
	}
	if last == 'B' {
		line.historyDown(history)
		return true
	}
	if last == 'C' {
		line.moveRight()
		return true
	}
	if last == 'D' {
		line.moveLeft()
		return true
	}
	if last == 'H' || last == '~' && (number == 1 || number == 7) {
		line.cursor = 0
		return true
	}
	if last == 'F' || last == '~' && (number == 4 || number == 8) {
		line.cursor = len(line.data)
		return true
	}
	if last == '~' && number == 3 {
		line.delete()
		return true
	}
	return false
}

func (line *replLine) insert(ch byte) {
	line.data = append(line.data, 0)
	copy(line.data[line.cursor+1:], line.data[line.cursor:len(line.data)-1])
	line.data[line.cursor] = ch
	line.cursor++
}

func (line *replLine) moveLeft() {
	line.cursor = replPreviousRune(line.data, line.cursor)
}

func (line *replLine) moveRight() {
	line.cursor = replNextRune(line.data, line.cursor)
}

func (line *replLine) backspace() {
	previous := replPreviousRune(line.data, line.cursor)
	if previous == line.cursor {
		return
	}
	copy(line.data[previous:], line.data[line.cursor:])
	line.data = line.data[:len(line.data)-(line.cursor-previous)]
	line.cursor = previous
}

func (line *replLine) delete() {
	next := replNextRune(line.data, line.cursor)
	if next == line.cursor {
		return
	}
	copy(line.data[line.cursor:], line.data[next:])
	line.data = line.data[:len(line.data)-(next-line.cursor)]
}

func (line *replLine) deleteWord() {
	start := line.cursor
	for start > 0 && replSpaceByte(line.data[replPreviousRune(line.data, start)]) {
		start = replPreviousRune(line.data, start)
	}
	for start > 0 && !replSpaceByte(line.data[replPreviousRune(line.data, start)]) {
		start = replPreviousRune(line.data, start)
	}
	copy(line.data[start:], line.data[line.cursor:])
	line.data = line.data[:len(line.data)-(line.cursor-start)]
	line.cursor = start
}

func (line *replLine) wordLeft() {
	for line.cursor > 0 && replSpaceByte(line.data[replPreviousRune(line.data, line.cursor)]) {
		line.moveLeft()
	}
	for line.cursor > 0 && !replSpaceByte(line.data[replPreviousRune(line.data, line.cursor)]) {
		line.moveLeft()
	}
}

func (line *replLine) wordRight() {
	for line.cursor < len(line.data) && replSpaceByte(line.data[line.cursor]) {
		line.moveRight()
	}
	for line.cursor < len(line.data) && !replSpaceByte(line.data[line.cursor]) {
		line.moveRight()
	}
}

func (line *replLine) historyUp(history []string) {
	if len(history) == 0 || line.historyIndex == 0 {
		return
	}
	if line.historyIndex == len(history) {
		line.saved = string(line.data)
	}
	line.historyIndex--
	line.data = []byte(history[line.historyIndex])
	line.cursor = len(line.data)
}

func (line *replLine) historyDown(history []string) {
	if line.historyIndex >= len(history) {
		return
	}
	line.historyIndex++
	if line.historyIndex == len(history) {
		line.data = []byte(line.saved)
	} else {
		line.data = []byte(history[line.historyIndex])
	}
	line.cursor = len(line.data)
}

func replPreviousRune(data []byte, cursor int) int {
	if cursor <= 0 {
		return 0
	}
	cursor--
	for cursor > 0 && data[cursor]&0xc0 == 0x80 {
		cursor--
	}
	return cursor
}

func replNextRune(data []byte, cursor int) int {
	if cursor >= len(data) {
		return len(data)
	}
	cursor++
	for cursor < len(data) && data[cursor]&0xc0 == 0x80 {
		cursor++
	}
	return cursor
}

func replSpaceByte(ch byte) bool {
	return ch == ' ' || ch == '\t'
}

func replRedraw(prompt string, data []byte, cursor int) {
	print("\r")
	print(prompt)
	print(string(data))
	print("\x1b[K")
	right := replRuneCount(data[cursor:])
	if right > 0 {
		print("\x1b[")
		print(strconv.Itoa(right))
		print("D")
	}
}

func replRuneCount(data []byte) int {
	count := 0
	for i := 0; i < len(data); i++ {
		if data[i]&0xc0 != 0x80 {
			count++
		}
	}
	return count
}

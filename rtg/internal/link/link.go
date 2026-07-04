package link

import (
	"j5.nz/rtg/rtg/internal/build"
	"j5.nz/rtg/rtg/internal/unit"
)

const (
	LinkOK = iota
	LinkErrBuild
	LinkErrRoot
	LinkErrUnit
)

type Result struct {
	Program      unit.Program
	Data         []byte
	Ok           bool
	Error        int
	ErrorPackage int
}

func LinkBuild(result build.Result) Result {
	out := Result{Ok: true, Error: LinkOK, ErrorPackage: -1}
	if !result.Ok {
		return linkFail(out, LinkErrBuild, result.ErrorPackage)
	}
	if result.Root < 0 || result.Root >= len(result.Units) {
		return linkFail(out, LinkErrRoot, -1)
	}
	program, ok := LinkUnits(result.Units, result.Root)
	if !ok {
		return linkFail(out, LinkErrUnit, -1)
	}
	data, ok := unit.Marshal(program)
	if !ok {
		return linkFail(out, LinkErrUnit, -1)
	}
	out.Program = program
	out.Data = data
	return out
}

func LinkUnits(units []build.PackageUnit, root int) (unit.Program, bool) {
	if root < 0 || root >= len(units) {
		return unit.Program{}, false
	}
	program := unit.Program{Package: units[root].Name}
	finalEOF := countLinkedTokens(units)
	lineOffset := 0
	for i := 0; i < len(units); i++ {
		ok := appendProgram(&program, units[i].Program, finalEOF, lineOffset, i+1 < len(units))
		if !ok {
			return unit.Program{}, false
		}
		lineOffset = nextLineOffset(lineOffset, units[i].Program.Text, i+1 < len(units))
	}
	program.Tokens = append(program.Tokens, unit.Token{
		Kind:  unit.TokenEOF,
		Start: len(program.Text),
		Size:  0,
		Line:  lineOffset + 1,
	})
	return program, true
}

func appendProgram(dst *unit.Program, src unit.Program, finalEOF int, lineOffset int, hasNext bool) bool {
	if src.Package == "" || len(src.Text) == 0 || len(src.Tokens) == 0 {
		return false
	}
	textOffset := len(dst.Text)
	oldToNew := make([]int, len(src.Tokens))
	for i := 0; i < len(src.Tokens); i++ {
		tok := src.Tokens[i]
		if tok.Kind == unit.TokenEOF {
			oldToNew[i] = finalEOF
			continue
		}
		oldToNew[i] = len(dst.Tokens)
		tok.Start += textOffset
		tok.Line += lineOffset
		dst.Tokens = append(dst.Tokens, tok)
	}
	dst.Text = append(dst.Text, src.Text...)
	for i := 0; i < len(src.Decls); i++ {
		decl := src.Decls[i]
		decl.NameStart += textOffset
		decl.NameEnd += textOffset
		decl.StartTok = mapToken(oldToNew, decl.StartTok, finalEOF)
		decl.EndTok = mapToken(oldToNew, decl.EndTok, finalEOF)
		dst.Decls = append(dst.Decls, decl)
	}
	for i := 0; i < len(src.Funcs); i++ {
		fn := src.Funcs[i]
		fn.NameStart += textOffset
		fn.NameEnd += textOffset
		fn.StartTok = mapToken(oldToNew, fn.StartTok, finalEOF)
		fn.NameTok = mapToken(oldToNew, fn.NameTok, finalEOF)
		fn.ReceiverStart = mapToken(oldToNew, fn.ReceiverStart, finalEOF)
		fn.ReceiverEnd = mapToken(oldToNew, fn.ReceiverEnd, finalEOF)
		fn.BodyStart = mapToken(oldToNew, fn.BodyStart, finalEOF)
		fn.BodyEnd = mapToken(oldToNew, fn.BodyEnd, finalEOF)
		fn.EndTok = mapToken(oldToNew, fn.EndTok, finalEOF)
		dst.Funcs = append(dst.Funcs, fn)
	}
	if hasNext && (len(src.Text) == 0 || src.Text[len(src.Text)-1] != '\n') {
		dst.Text = append(dst.Text, '\n')
	}
	return true
}

func countLinkedTokens(units []build.PackageUnit) int {
	count := 0
	for i := 0; i < len(units); i++ {
		tokens := units[i].Program.Tokens
		for j := 0; j < len(tokens); j++ {
			if tokens[j].Kind != unit.TokenEOF {
				count++
			}
		}
	}
	return count
}

func nextLineOffset(lineOffset int, text []byte, hasNext bool) int {
	lineOffset += countNewlines(text)
	if hasNext && (len(text) == 0 || text[len(text)-1] != '\n') {
		lineOffset++
	}
	return lineOffset
}

func mapToken(oldToNew []int, tok int, eof int) int {
	if tok < 0 || tok >= len(oldToNew) {
		return eof
	}
	return oldToNew[tok]
}

func countNewlines(text []byte) int {
	count := 0
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			count++
		}
	}
	return count
}

func linkFail(result Result, err int, pkg int) Result {
	result.Ok = false
	result.Error = err
	result.ErrorPackage = pkg
	return result
}

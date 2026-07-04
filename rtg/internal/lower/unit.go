package lower

import (
	"j5.nz/rtg/rtg/internal/check"
	"j5.nz/rtg/rtg/internal/load"
	"j5.nz/rtg/rtg/internal/syntax"
	"j5.nz/rtg/rtg/internal/unit"
)

const (
	EmitOK = iota
	EmitErrGraph
	EmitErrPackage
	EmitErrToken
	EmitErrUnit
	EmitErrCheck
)

type Result struct {
	Program    unit.Program
	Ok         bool
	Error      int
	ErrorFile  int
	ErrorToken int
}

func EmitRoot(graph load.Graph) Result {
	if !graph.Ok {
		return emitFail(Result{}, EmitErrGraph, -1, -1)
	}
	for i := 0; i < len(graph.Packages); i++ {
		if graph.Packages[i].Ref.ImportPath == graph.Root {
			return EmitPackage(graph.Packages[i])
		}
	}
	return emitFail(Result{}, EmitErrGraph, -1, -1)
}

func EmitPackage(pkg load.Package) Result {
	result := Result{Ok: true, Error: EmitOK, ErrorFile: -1, ErrorToken: -1}
	if !pkg.Ok || pkg.Name == "" || len(pkg.Files) == 0 {
		return emitFail(result, EmitErrPackage, -1, -1)
	}
	var builder unitBuilder
	builder.program.Package = pkg.Name
	builder.finalEOF = countPackageTokens(pkg)
	for i := 0; i < len(pkg.Files); i++ {
		if !pkg.Files[i].File.Ok {
			return emitFail(result, EmitErrPackage, i, pkg.Files[i].File.ErrorTok)
		}
		if !builder.addFile(pkg.Files[i].File, i, i+1 < len(pkg.Files)) {
			return emitFail(result, builder.err, builder.errFile, builder.errToken)
		}
	}
	line := builder.lineOffset + 1
	builder.program.Tokens = append(builder.program.Tokens, unit.Token{
		Kind:  unit.TokenEOF,
		Start: len(builder.program.Text),
		Size:  0,
		Line:  line,
	})
	if _, ok := unit.Marshal(builder.program); !ok {
		return emitFail(result, EmitErrUnit, -1, -1)
	}
	result.Program = builder.program
	return result
}

func EmitRootChecked(graph load.Graph, prog check.Program) Result {
	if !graph.Ok || !prog.Ok || len(graph.Packages) != len(prog.Packages) {
		return emitFail(Result{}, EmitErrGraph, -1, -1)
	}
	for i := 0; i < len(graph.Packages); i++ {
		if graph.Packages[i].Ref.ImportPath == graph.Root {
			return EmitCheckedPackage(graph.Packages[i], prog.Packages[i])
		}
	}
	return emitFail(Result{}, EmitErrGraph, -1, -1)
}

func EmitCheckedPackage(pkg load.Package, info check.PackageInfo) Result {
	result := Result{Ok: true, Error: EmitOK, ErrorFile: -1, ErrorToken: -1}
	if !pkg.Ok || pkg.Name == "" || len(pkg.Files) == 0 {
		return emitFail(result, EmitErrPackage, -1, -1)
	}
	if info.Name != pkg.Name {
		return emitFail(result, EmitErrCheck, -1, -1)
	}
	var builder unitBuilder
	builder.program.Package = pkg.Name
	builder.finalEOF = countPackageTokens(pkg)
	files := make([]fileTokens, len(pkg.Files))
	for i := 0; i < len(pkg.Files); i++ {
		if !pkg.Files[i].File.Ok {
			return emitFail(result, EmitErrPackage, i, pkg.Files[i].File.ErrorTok)
		}
		oldToNew, ok := builder.addFileTokens(pkg.Files[i].File, i, i+1 < len(pkg.Files))
		if !ok {
			return emitFail(result, builder.err, builder.errFile, builder.errToken)
		}
		files[i] = fileTokens{file: pkg.Files[i].File, oldToNew: oldToNew}
	}
	if !builder.addCheckedDecls(info, files) {
		return emitFail(result, builder.err, builder.errFile, builder.errToken)
	}
	if !builder.addCheckedFuncs(info, files) {
		return emitFail(result, builder.err, builder.errFile, builder.errToken)
	}
	if !builder.finishUnit() {
		return emitFail(result, builder.err, builder.errFile, builder.errToken)
	}
	result.Program = builder.program
	return result
}

type unitBuilder struct {
	program    unit.Program
	lineOffset int
	finalEOF   int
	err        int
	errFile    int
	errToken   int
}

type fileTokens struct {
	file     syntax.File
	oldToNew []int
}

func (b *unitBuilder) addFile(file syntax.File, fileIndex int, hasNext bool) bool {
	oldToNew, ok := b.addFileTokens(file, fileIndex, hasNext)
	if !ok {
		return false
	}
	for i := 0; i < len(file.Decls); i++ {
		decl := file.Decls[i]
		nameTok := mapToken(oldToNew, decl.NameTok, b.finalEOF)
		if !b.addDecl(file, decl, nameTok, oldToNew, fileIndex) {
			return false
		}
	}
	for i := 0; i < len(file.Funcs); i++ {
		if !b.addFunc(file, file.Funcs[i], oldToNew, fileIndex) {
			return false
		}
	}
	return true
}

func (b *unitBuilder) addFileTokens(file syntax.File, fileIndex int, hasNext bool) ([]int, bool) {
	base := len(b.program.Text)
	lineOffset := b.lineOffset
	oldToNew := make([]int, len(file.Tokens))
	for i := 0; i < len(file.Tokens); i++ {
		tok := file.Tokens[i]
		if tok.Kind == syntax.TokenEOF {
			oldToNew[i] = b.finalEOF
			continue
		}
		kind, ok := unitTokenKind(file.Src, tok)
		if !ok {
			b.setErr(EmitErrToken, fileIndex, i)
			return nil, false
		}
		oldToNew[i] = len(b.program.Tokens)
		b.program.Tokens = append(b.program.Tokens, unit.Token{
			Kind:  kind,
			Start: base + tok.Start,
			Size:  tok.End - tok.Start,
			Line:  lineOffset + tok.Line,
		})
	}
	b.program.Text = append(b.program.Text, file.Src...)
	b.lineOffset += countNewlines(file.Src)
	if hasNext && (len(file.Src) == 0 || file.Src[len(file.Src)-1] != '\n') {
		b.program.Text = append(b.program.Text, '\n')
		b.lineOffset++
	}
	return oldToNew, true
}

func (b *unitBuilder) finishUnit() bool {
	line := b.lineOffset + 1
	b.program.Tokens = append(b.program.Tokens, unit.Token{
		Kind:  unit.TokenEOF,
		Start: len(b.program.Text),
		Size:  0,
		Line:  line,
	})
	if _, ok := unit.Marshal(b.program); !ok {
		b.setErr(EmitErrUnit, -1, -1)
		return false
	}
	return true
}

func (b *unitBuilder) addDecl(file syntax.File, decl syntax.TopDecl, nameTok int, oldToNew []int, fileIndex int) bool {
	if nameTok < 0 || nameTok >= len(b.program.Tokens) {
		b.setErr(EmitErrToken, fileIndex, decl.NameTok)
		return false
	}
	kind, ok := unitDeclKind(decl.Kind)
	if !ok {
		b.setErr(EmitErrToken, fileIndex, decl.NameTok)
		return false
	}
	name := b.program.Tokens[nameTok]
	b.program.Decls = append(b.program.Decls, unit.Decl{
		Kind:      kind,
		NameStart: name.Start,
		NameEnd:   name.Start + name.Size,
		StartTok:  mapToken(oldToNew, decl.StartTok, b.finalEOF),
		EndTok:    mapToken(oldToNew, decl.EndTok, b.finalEOF),
	})
	return true
}

func (b *unitBuilder) addFunc(file syntax.File, fn syntax.FuncDecl, oldToNew []int, fileIndex int) bool {
	nameTok := mapToken(oldToNew, fn.NameTok, b.finalEOF)
	if nameTok < 0 || nameTok >= len(b.program.Tokens) {
		b.setErr(EmitErrToken, fileIndex, fn.NameTok)
		return false
	}
	bodyEnd := fn.BodyEnd - 1
	if bodyEnd < fn.BodyStart {
		b.setErr(EmitErrToken, fileIndex, fn.BodyEnd)
		return false
	}
	name := b.program.Tokens[nameTok]
	b.program.Funcs = append(b.program.Funcs, unit.Func{
		NameStart:     name.Start,
		NameEnd:       name.Start + name.Size,
		StartTok:      mapToken(oldToNew, fn.StartTok, b.finalEOF),
		NameTok:       nameTok,
		ReceiverStart: mapToken(oldToNew, fn.ReceiverStart, b.finalEOF),
		ReceiverEnd:   mapToken(oldToNew, fn.ReceiverEnd, b.finalEOF),
		BodyStart:     mapToken(oldToNew, fn.BodyStart, b.finalEOF),
		BodyEnd:       mapToken(oldToNew, bodyEnd, b.finalEOF),
		EndTok:        mapToken(oldToNew, fn.EndTok, b.finalEOF),
	})
	return true
}

func (b *unitBuilder) addCheckedDecls(info check.PackageInfo, files []fileTokens) bool {
	if len(info.DeclOrder) != len(info.Decls) {
		b.setErr(EmitErrCheck, -1, -1)
		return false
	}
	seen := make([]bool, len(info.Decls))
	for i := 0; i < len(info.DeclOrder); i++ {
		index := info.DeclOrder[i]
		if index < 0 || index >= len(info.Decls) || seen[index] {
			b.setErr(EmitErrCheck, -1, -1)
			return false
		}
		seen[index] = true
		declInfo := info.Decls[index]
		if declInfo.File < 0 || declInfo.File >= len(files) {
			b.setErr(EmitErrCheck, -1, declInfo.Token)
			return false
		}
		file := files[declInfo.File].file
		decl := findFileDecl(file, declInfo.Token)
		if decl.NameTok < 0 {
			b.setErr(EmitErrCheck, declInfo.File, declInfo.Token)
			return false
		}
		nameTok := mapToken(files[declInfo.File].oldToNew, decl.NameTok, b.finalEOF)
		if !b.addDecl(file, decl, nameTok, files[declInfo.File].oldToNew, declInfo.File) {
			return false
		}
	}
	return true
}

func (b *unitBuilder) addCheckedFuncs(info check.PackageInfo, files []fileTokens) bool {
	for i := 0; i < len(info.Bodies); i++ {
		body := info.Bodies[i]
		if body.File < 0 || body.File >= len(files) {
			b.setErr(EmitErrCheck, -1, -1)
			return false
		}
		file := files[body.File].file
		if body.Func < 0 || body.Func >= len(file.Funcs) {
			b.setErr(EmitErrCheck, body.File, -1)
			return false
		}
		fn := file.Funcs[body.Func]
		if fn.NameTok < 0 || fn.NameTok >= len(file.Tokens) {
			b.setErr(EmitErrCheck, body.File, fn.NameTok)
			return false
		}
		if !b.addFunc(file, fn, files[body.File].oldToNew, body.File) {
			return false
		}
	}
	return true
}

func findFileDecl(file syntax.File, nameTok int) syntax.TopDecl {
	for i := 0; i < len(file.Decls); i++ {
		if file.Decls[i].NameTok == nameTok {
			return file.Decls[i]
		}
	}
	return syntax.TopDecl{NameTok: -1}
}

func (b *unitBuilder) setErr(err int, file int, tok int) {
	b.err = err
	b.errFile = file
	b.errToken = tok
}

func countPackageTokens(pkg load.Package) int {
	count := 0
	for i := 0; i < len(pkg.Files); i++ {
		file := pkg.Files[i].File
		for j := 0; j < len(file.Tokens); j++ {
			if file.Tokens[j].Kind != syntax.TokenEOF {
				count++
			}
		}
	}
	return count
}

func unitTokenKind(src []byte, tok syntax.Token) (int, bool) {
	if tok.Kind == syntax.TokenEOF {
		return unit.TokenEOF, true
	}
	if tok.Kind == syntax.TokenIdent {
		return unit.TokenIdent, true
	}
	if tok.Kind == syntax.TokenNumber {
		if isFloatNumber(src, tok) {
			return unit.TokenFloat, true
		}
		return unit.TokenNumber, true
	}
	if tok.Kind == syntax.TokenString {
		return unit.TokenString, true
	}
	if tok.Kind == syntax.TokenChar {
		return unit.TokenChar, true
	}
	if tok.Kind == syntax.TokenOperator {
		return unit.TokenOp, true
	}
	if tok.Kind == syntax.TokenPackage {
		return unit.TokenPackage, true
	}
	if tok.Kind == syntax.TokenConst {
		return unit.TokenConst, true
	}
	if tok.Kind == syntax.TokenVar {
		return unit.TokenVar, true
	}
	if tok.Kind == syntax.TokenType {
		return unit.TokenType, true
	}
	if tok.Kind == syntax.TokenFunc {
		return unit.TokenFunc, true
	}
	if tok.Kind == syntax.TokenStruct {
		return unit.TokenStruct, true
	}
	if tok.Kind == syntax.TokenReturn {
		return unit.TokenReturn, true
	}
	if tok.Kind == syntax.TokenIf {
		return unit.TokenIf, true
	}
	if tok.Kind == syntax.TokenElse {
		return unit.TokenElse, true
	}
	if tok.Kind == syntax.TokenFor {
		return unit.TokenFor, true
	}
	if tok.Kind == syntax.TokenBreak {
		return unit.TokenBreak, true
	}
	if tok.Kind == syntax.TokenContinue {
		return unit.TokenContinue, true
	}
	if tok.Kind == syntax.TokenGoto {
		return unit.TokenGoto, true
	}
	if tok.Kind == syntax.TokenSwitch {
		return unit.TokenSwitch, true
	}
	if tok.Kind == syntax.TokenCase {
		return unit.TokenCase, true
	}
	if tok.Kind == syntax.TokenDefault {
		return unit.TokenDefault, true
	}
	return unit.TokenIdent, true
}

func unitDeclKind(kind int) (int, bool) {
	if kind == syntax.TokenConst {
		return unit.TokenConst, true
	}
	if kind == syntax.TokenVar {
		return unit.TokenVar, true
	}
	if kind == syntax.TokenType {
		return unit.TokenType, true
	}
	return 0, false
}

func mapToken(oldToNew []int, tok int, eof int) int {
	if tok < 0 || tok >= len(oldToNew) {
		return eof
	}
	return oldToNew[tok]
}

func countNewlines(src []byte) int {
	count := 0
	for i := 0; i < len(src); i++ {
		if src[i] == '\n' {
			count++
		}
	}
	return count
}

func isFloatNumber(src []byte, tok syntax.Token) bool {
	for i := tok.Start; i < tok.End && i < len(src); i++ {
		c := src[i]
		if c == '.' || c == 'e' || c == 'E' || c == 'p' || c == 'P' {
			return true
		}
	}
	return false
}

func emitFail(result Result, err int, file int, tok int) Result {
	result.Ok = false
	result.Error = err
	result.ErrorFile = file
	result.ErrorToken = tok
	return result
}

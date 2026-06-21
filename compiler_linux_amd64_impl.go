package main

const rtgTokEOF = 0
const rtgTokIdent = 1
const rtgTokNumber = 2
const rtgTokFloat = 3
const rtgTokString = 4
const rtgTokChar = 5
const rtgTokPackage = 6
const rtgTokConst = 7
const rtgTokVar = 8
const rtgTokType = 9
const rtgTokFunc = 10
const rtgTokStruct = 11
const rtgTokReturn = 12
const rtgTokIf = 13
const rtgTokElse = 14
const rtgTokFor = 15
const rtgTokBreak = 16
const rtgTokContinue = 17
const rtgTokGoto = 18
const rtgTokOp = 19

type rtgToken struct {
	kind  int
	start int
	end   int
	line  int
}

type rtgDecl struct {
	kind      int
	nameStart int
	nameEnd   int
	startTok  int
	endTok    int
}

type rtgFuncDecl struct {
	nameStart int
	nameEnd   int
	startTok  int
	bodyStart int
	bodyEnd   int
	endTok    int
}

type rtgProgram struct {
	src   []byte
	toks  []rtgToken
	decls []rtgDecl
	funcs []rtgFuncDecl
	ok    bool
}

const rtgExprBad = 0
const rtgExprIdent = 1
const rtgExprInt = 2
const rtgExprFloat = 3
const rtgExprString = 4
const rtgExprChar = 5
const rtgExprBool = 6
const rtgExprUnary = 7
const rtgExprBinary = 8
const rtgExprCall = 9
const rtgExprIndex = 10
const rtgExprSelector = 11
const rtgExprComposite = 12

const rtgStmtBad = 0
const rtgStmtReturn = 1
const rtgStmtIf = 2
const rtgStmtFor = 3
const rtgStmtBreak = 4
const rtgStmtContinue = 5
const rtgStmtGoto = 6
const rtgStmtLabel = 7
const rtgStmtVar = 8
const rtgStmtShort = 9
const rtgStmtAssign = 10
const rtgStmtExpr = 11

type rtgExpr struct {
	kind      int
	tok       int
	left      int
	right     int
	firstArg  int
	argCount  int
	nameStart int
	nameEnd   int
}

type rtgExprParse struct {
	prog   rtgProgram
	pos    int
	end    int
	exprs  []rtgExpr
	args   []int
	fields []rtgCompositeField
	ok     bool
}

type rtgCompositeField struct {
	nameStart int
	nameEnd   int
	expr      int
}

type rtgStmt struct {
	kind      int
	startTok  int
	endTok    int
	exprStart int
	exprEnd   int
	bodyStart int
	bodyEnd   int
	elseStart int
	elseEnd   int
	nameStart int
	nameEnd   int
}

type rtgBodyParse struct {
	prog  rtgProgram
	stmts []rtgStmt
	ok    bool
}

const rtgTypeInvalid = 0
const rtgTypeInt = 1
const rtgTypeInt64 = 2
const rtgTypeByte = 3
const rtgTypeBool = 4
const rtgTypeString = 5
const rtgTypeFloat64 = 6
const rtgTypePointer = 7
const rtgTypeSlice = 8
const rtgTypeStruct = 9
const rtgTypeNamed = 10

type rtgTypeInfo struct {
	kind      int
	elem      int
	first     int
	count     int
	size      int
	nameStart int
	nameEnd   int
}

type rtgFieldInfo struct {
	nameStart int
	nameEnd   int
	typ       int
	offset    int
}

type rtgSymbolInfo struct {
	nameStart int
	nameEnd   int
	kind      int
	typ       int
	initStart int
	initEnd   int
}

type rtgFuncInfo struct {
	declIndex  int
	nameStart  int
	nameEnd    int
	firstParam int
	paramCount int
	resultType int
	bodyStart  int
	bodyEnd    int
}

type rtgMeta struct {
	prog    rtgProgram
	types   []rtgTypeInfo
	fields  []rtgFieldInfo
	globals []rtgSymbolInfo
	params  []rtgSymbolInfo
	funcs   []rtgFuncInfo
	ok      bool
}

const rtgRel32 = 1

type rtgLabelRef struct {
	at    int
	label int
	kind  int
}

type rtgDataRef struct {
	at  int
	off int
}

type rtgAbsRef struct {
	at   int
	off  int
	kind int
}

type rtgAsm struct {
	code       []byte
	labelPos   []int
	labelSet   []bool
	relocs     []rtgLabelRef
	dataRelocs []rtgDataRef
	absRelocs  []rtgAbsRef
	data       []byte
	bssSize    int
	imageBase  int
	codeOffset int
	dataOffset int
}

type rtgLocalInfo struct {
	nameStart int
	nameEnd   int
	offset    int
	typ       int
	size      int
}

type rtgGlobalInfo struct {
	nameStart int
	nameEnd   int
	offset    int
}

type rtgSliceLocation struct {
	offset int
	typ    int
	ok     bool
}

type rtgCompileResult struct {
	data []byte
	ok   bool
}

type rtgConstResult struct {
	value int
	ok    bool
}

type rtgTypeResult struct {
	typ  int
	next int
}

type rtgLinearGen struct {
	prog           rtgProgram
	meta           rtgMeta
	asm            rtgAsm
	funcLabels     []int
	currentFunc    int
	returnStruct   int
	returnExits    bool
	locals         []rtgLocalInfo
	stackUsed      int
	globals        []rtgGlobalInfo
	breakLabels    []int
	continueLabels []int
	breakDepth     int
	continueDepth  int
	stackSize      int
}

func compileLinuxAmd64(input []int, output int) int {
	var src []byte
	for i := 0; i < len(input); i++ {
		part := rtgReadAll(input[i])
		for j := 0; j < len(part); j++ {
			src = append(src, part[j])
		}
		src = append(src, '\n')
	}

	prog := rtgParseProgram(src)
	if !prog.ok {
		print("parse error\n")
		return 1
	}
	meta := rtgBuildMeta(prog)
	if !meta.ok {
		print("parse error\n")
		return 1
	}
	result := rtgTryCompileScalarProgram(prog, meta)
	if result.ok {
		write(output, result.data, 0)
		return 0
	}
	result = rtgTryCompileLinearAppMain(prog, meta)
	if result.ok {
		write(output, result.data, 0)
		return 0
	}

	print("compiler backend not implemented\n")
	return 1
}

func rtgReadAll(fd int) []byte {
	var out []byte
	var buf []byte
	for len(buf) < 4096 {
		buf = append(buf, 0)
	}
	for {
		n := read(fd, buf, -1)
		if n <= 0 {
			return out
		}
		for i := 0; i < n; i++ {
			out = append(out, buf[i])
		}
	}
}

func rtgParseProgram(src []byte) rtgProgram {
	var p rtgProgram
	p.src = src
	p.toks = rtgScan(src)
	p.ok = true

	i := 0
	if !rtgTokIsKeyword(p, i, rtgTokPackage) {
		p.ok = false
		return p
	}
	i++
	if !rtgTokIsKind(p, i, rtgTokIdent) {
		p.ok = false
		return p
	}
	i++

	for i < len(p.toks) && p.toks[i].kind != rtgTokEOF {
		if rtgTokIsKeyword(p, i, rtgTokPackage) {
			i++
			if !rtgTokIsKind(p, i, rtgTokIdent) {
				p.ok = false
				return p
			}
			i++
			continue
		}
		if rtgTokIsKeyword(p, i, rtgTokConst) || rtgTokIsKeyword(p, i, rtgTokVar) || rtgTokIsKeyword(p, i, rtgTokType) {
			start := i
			kind := p.toks[i].kind
			i++
			if rtgTokTextIs(p, i, "(") {
				end := rtgSkipBalanced(p, i, "(", ")")
				if end <= i {
					p.ok = false
					return p
				}
				p.decls = append(p.decls, rtgDecl{kind: kind, nameStart: p.toks[start].start, nameEnd: p.toks[start].end, startTok: start, endTok: end})
				i = end
				continue
			}
			if !rtgTokIsKind(p, i, rtgTokIdent) {
				p.ok = false
				return p
			}
			name := p.toks[i]
			i++
			end := rtgSkipTopLevelLine(p, i)
			p.decls = append(p.decls, rtgDecl{kind: kind, nameStart: name.start, nameEnd: name.end, startTok: start, endTok: end})
			i = end
			continue
		}
		if rtgTokIsKeyword(p, i, rtgTokFunc) {
			fn := rtgParseFuncDecl(p, i)
			if fn.endTok <= i {
				p.ok = false
				return p
			}
			p.funcs = append(p.funcs, fn)
			i = fn.endTok
			continue
		}
		i++
	}

	return p
}

func rtgParseFuncDecl(p rtgProgram, start int) rtgFuncDecl {
	var fn rtgFuncDecl
	fn.startTok = start
	i := start + 1
	if !rtgTokIsKind(p, i, rtgTokIdent) {
		return fn
	}
	fn.nameStart = p.toks[i].start
	fn.nameEnd = p.toks[i].end
	i++

	for i < len(p.toks) && !rtgTokTextIs(p, i, "{") && p.toks[i].kind != rtgTokEOF {
		i++
	}
	if !rtgTokTextIs(p, i, "{") {
		return fn
	}
	fn.bodyStart = i
	depth := 1
	i++
	for i < len(p.toks) && depth > 0 {
		if rtgTokTextIs(p, i, "{") {
			depth++
		} else if rtgTokTextIs(p, i, "}") {
			depth--
		}
		i++
	}
	if depth != 0 {
		return rtgFuncDecl{}
	}
	fn.bodyEnd = i - 1
	fn.endTok = i
	return fn
}

func rtgSkipBalanced(p rtgProgram, start int, open string, close string) int {
	if !rtgTokTextIs(p, start, open) {
		return start
	}
	depth := 1
	i := start + 1
	for i < len(p.toks) && depth > 0 {
		if rtgTokTextIs(p, i, open) {
			depth++
		} else if rtgTokTextIs(p, i, close) {
			depth--
		}
		i++
	}
	if depth != 0 {
		return start
	}
	return i
}

func rtgSkipTopLevelLine(p rtgProgram, start int) int {
	if start >= len(p.toks) {
		return start
	}
	line := p.toks[start-1].line
	i := start
	depth := 0
	for i < len(p.toks) {
		if p.toks[i].kind == rtgTokEOF {
			return i
		}
		if p.toks[i].line != line && depth == 0 {
			return i
		}
		if rtgTokTextIs(p, i, "{") || rtgTokTextIs(p, i, "(") {
			depth++
		} else if rtgTokTextIs(p, i, "}") || rtgTokTextIs(p, i, ")") {
			depth--
		}
		i++
	}
	return i
}

func rtgScan(src []byte) []rtgToken {
	var toks []rtgToken
	i := 0
	line := 1
	for i < len(src) {
		c := src[i]
		if c == ' ' || c == '\t' || c == '\r' {
			i++
			continue
		}
		if c == '\n' {
			line++
			i++
			continue
		}
		if c == '/' && i+1 < len(src) && src[i+1] == '/' {
			i += 2
			for i < len(src) && src[i] != '\n' {
				i++
			}
			continue
		}
		if c == '/' && i+1 < len(src) && src[i+1] == '*' {
			i += 2
			for i+1 < len(src) && !(src[i] == '*' && src[i+1] == '/') {
				if src[i] == '\n' {
					line++
				}
				i++
			}
			if i+1 < len(src) {
				i += 2
			}
			continue
		}
		if rtgIsIdentStart(c) {
			start := i
			i++
			for i < len(src) && rtgIsIdentPart(src[i]) {
				i++
			}
			toks = append(toks, rtgToken{kind: rtgKeywordKind(src, start, i), start: start, end: i, line: line})
			continue
		}
		if rtgIsDigit(c) {
			start := i
			kind := rtgTokNumber
			if c == '0' && i+1 < len(src) && (src[i+1] == 'x' || src[i+1] == 'X' || src[i+1] == 'b' || src[i+1] == 'B') {
				i += 2
				for i < len(src) && rtgIsBasedDigit(src[i]) {
					i++
				}
			} else {
				i++
				for i < len(src) && rtgIsDigit(src[i]) {
					i++
				}
				if i < len(src) && src[i] == '.' {
					kind = rtgTokFloat
					i++
					for i < len(src) && rtgIsDigit(src[i]) {
						i++
					}
				}
			}
			toks = append(toks, rtgToken{kind: kind, start: start, end: i, line: line})
			continue
		}
		if c == '"' {
			start := i
			i++
			for i < len(src) && src[i] != '"' {
				if src[i] == '\\' && i+1 < len(src) {
					i += 2
				} else {
					if src[i] == '\n' {
						line++
					}
					i++
				}
			}
			if i < len(src) {
				i++
			}
			toks = append(toks, rtgToken{kind: rtgTokString, start: start, end: i, line: line})
			continue
		}
		if c == '\'' {
			start := i
			i++
			for i < len(src) && src[i] != '\'' {
				if src[i] == '\\' && i+1 < len(src) {
					i += 2
				} else {
					i++
				}
			}
			if i < len(src) {
				i++
			}
			toks = append(toks, rtgToken{kind: rtgTokChar, start: start, end: i, line: line})
			continue
		}
		start := i
		i = rtgScanOperator(src, i)
		toks = append(toks, rtgToken{kind: rtgTokOp, start: start, end: i, line: line})
	}
	toks = append(toks, rtgToken{kind: rtgTokEOF, start: len(src), end: len(src), line: line})
	return toks
}

func rtgScanOperator(src []byte, i int) int {
	if i+2 <= len(src) {
		if src[i] == ':' && src[i+1] == '=' {
			return i + 2
		}
		if src[i] == '=' && src[i+1] == '=' {
			return i + 2
		}
		if src[i] == '!' && src[i+1] == '=' {
			return i + 2
		}
		if src[i] == '<' && src[i+1] == '=' {
			return i + 2
		}
		if src[i] == '>' && src[i+1] == '=' {
			return i + 2
		}
		if src[i] == '&' && src[i+1] == '&' {
			return i + 2
		}
		if src[i] == '|' && src[i+1] == '|' {
			return i + 2
		}
		if src[i] == '<' && src[i+1] == '<' {
			return i + 2
		}
		if src[i] == '>' && src[i+1] == '>' {
			return i + 2
		}
		if src[i] == '&' && src[i+1] == '^' {
			return i + 2
		}
		if src[i+1] == '=' && (src[i] == '+' || src[i] == '-' || src[i] == '*' || src[i] == '/' || src[i] == '%') {
			return i + 2
		}
		if src[i] == '+' && src[i+1] == '+' {
			return i + 2
		}
		if src[i] == '-' && src[i+1] == '-' {
			return i + 2
		}
	}
	return i + 1
}

func rtgKeywordKind(src []byte, start int, end int) int {
	if end-start == 7 && rtgBytesEqualTextAt(src, start, "package") {
		return rtgTokPackage
	}
	if end-start == 5 && rtgBytesEqualTextAt(src, start, "const") {
		return rtgTokConst
	}
	if end-start == 3 && rtgBytesEqualTextAt(src, start, "var") {
		return rtgTokVar
	}
	if end-start == 4 && rtgBytesEqualTextAt(src, start, "type") {
		return rtgTokType
	}
	if end-start == 4 && rtgBytesEqualTextAt(src, start, "func") {
		return rtgTokFunc
	}
	if end-start == 6 && rtgBytesEqualTextAt(src, start, "struct") {
		return rtgTokStruct
	}
	if end-start == 6 && rtgBytesEqualTextAt(src, start, "return") {
		return rtgTokReturn
	}
	if end-start == 2 && rtgBytesEqualTextAt(src, start, "if") {
		return rtgTokIf
	}
	if end-start == 4 && rtgBytesEqualTextAt(src, start, "else") {
		return rtgTokElse
	}
	if end-start == 3 && rtgBytesEqualTextAt(src, start, "for") {
		return rtgTokFor
	}
	if end-start == 5 && rtgBytesEqualTextAt(src, start, "break") {
		return rtgTokBreak
	}
	if end-start == 8 && rtgBytesEqualTextAt(src, start, "continue") {
		return rtgTokContinue
	}
	if end-start == 4 && rtgBytesEqualTextAt(src, start, "goto") {
		return rtgTokGoto
	}
	return rtgTokIdent
}

func rtgTokIsKeyword(p rtgProgram, i int, kind int) bool {
	return i >= 0 && i < len(p.toks) && p.toks[i].kind == kind
}

func rtgTokIsKind(p rtgProgram, i int, kind int) bool {
	return i >= 0 && i < len(p.toks) && p.toks[i].kind == kind
}

func rtgTokTextIs(p rtgProgram, i int, text string) bool {
	if i < 0 || i >= len(p.toks) {
		return false
	}
	return rtgBytesEqualText(p.src, p.toks[i].start, p.toks[i].end, text)
}

func rtgIsIdentStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func rtgIsIdentPart(c byte) bool {
	return rtgIsIdentStart(c) || rtgIsDigit(c)
}

func rtgIsDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func rtgIsBasedDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func rtgBytesEqualText(src []byte, start int, end int, text string) bool {
	if end-start != len(text) {
		return false
	}
	return rtgBytesEqualTextAt(src, start, text)
}

func rtgBytesEqualTextAt(src []byte, start int, text string) bool {
	for i := 0; i < len(text); i++ {
		if src[start+i] != text[i] {
			return false
		}
	}
	return true
}

func rtgDecodeStringToken(src []byte, tok rtgToken) []byte {
	var out []byte
	i := tok.start + 1
	end := tok.end - 1
	for i < end {
		if src[i] == '\\' && i+1 < end {
			i++
			if src[i] == 'n' {
				out = append(out, '\n')
			} else if src[i] == 't' {
				out = append(out, '\t')
			} else if src[i] == '"' {
				out = append(out, '"')
			} else if src[i] == '\\' {
				out = append(out, '\\')
			} else {
				out = append(out, src[i])
			}
			i++
			continue
		}
		out = append(out, src[i])
		i++
	}
	return out
}

func rtgParseIntToken(src []byte, tok rtgToken) int {
	start := tok.start
	base := 10
	if tok.end-start > 2 && src[start] == '0' && (src[start+1] == 'x' || src[start+1] == 'X') {
		base = 16
		start += 2
	} else if tok.end-start > 2 && src[start] == '0' && (src[start+1] == 'b' || src[start+1] == 'B') {
		base = 2
		start += 2
	}
	n := 0
	for i := start; i < tok.end; i++ {
		d := 0
		if src[i] >= '0' && src[i] <= '9' {
			d = int(src[i] - '0')
		} else if src[i] >= 'a' && src[i] <= 'f' {
			d = int(src[i]-'a') + 10
		} else if src[i] >= 'A' && src[i] <= 'F' {
			d = int(src[i]-'A') + 10
		}
		n = n*base + d
	}
	return n
}

func rtgParseCharToken(src []byte, tok rtgToken) int {
	i := tok.start + 1
	if i >= tok.end-1 {
		return 0
	}
	if src[i] != '\\' {
		return int(src[i])
	}
	i++
	if i >= tok.end-1 {
		return 0
	}
	if src[i] == 'n' {
		return 10
	}
	if src[i] == 't' {
		return 9
	}
	if src[i] == '\\' {
		return 92
	}
	if src[i] == '\'' {
		return 39
	}
	return int(src[i])
}

func rtgTryCompileScalarProgram(p rtgProgram, meta rtgMeta) rtgCompileResult {
	appIndex := rtgFindFuncInfoByName(meta, "appMain")
	if appIndex < 0 {
		var r rtgCompileResult
		return r
	}
	var g rtgLinearGen
	g.prog = p
	g.meta = meta
	g.stackSize = 4096
	rtgAsmInit(&g.asm)
	for i := 0; i < len(meta.funcs); i++ {
		g.funcLabels = append(g.funcLabels, rtgAsmNewLabel(&g.asm))
	}
	if !rtgLinearInitGlobals(&g) {
		var r rtgCompileResult
		return r
	}
	if !rtgEmitProgramEntryArgs(&g, meta.funcs[appIndex]) {
		var r rtgCompileResult
		return r
	}
	rtgAsmCallLabel(&g.asm, g.funcLabels[appIndex])
	rtgAsmMovRdiRax(&g.asm)
	rtgAsmMovRaxImm(&g.asm, 60)
	rtgAsmSyscall(&g.asm)
	for i := 0; i < len(meta.funcs); i++ {
		if !rtgEmitScalarFunction(&g, i) {
			var r rtgCompileResult
			return r
		}
	}
	return rtgCompileResult{data: rtgAsmImage(&g.asm), ok: true}
}

func rtgEmitProgramEntryArgs(g *rtgLinearGen, app rtgFuncInfo) bool {
	if app.paramCount == 0 {
		return true
	}
	first := g.meta.params[app.firstParam]
	if !rtgTypeIsSlice(&g.meta, first.typ) {
		return false
	}
	argsOff := g.asm.bssSize
	g.asm.bssSize += 32768
	rtgAsmBuildArgvSlice(&g.asm, argsOff)
	return true
}

func rtgEmitScalarFunction(g *rtgLinearGen, fnInfoIndex int) bool {
	metaFn := g.meta.funcs[fnInfoIndex]
	fn := g.prog.funcs[metaFn.declIndex]
	oldLocals := g.locals
	oldBreak := g.breakDepth
	oldContinue := g.continueDepth
	oldCurrent := g.currentFunc
	oldReturnStruct := g.returnStruct
	var locals []rtgLocalInfo
	g.locals = locals
	g.breakDepth = 0
	g.continueDepth = 0
	g.currentFunc = fnInfoIndex
	g.returnStruct = 0
	rtgAsmMarkLabel(&g.asm, g.funcLabels[fnInfoIndex])
	rtgAsmPushRbp(&g.asm)
	rtgAsmMovRbpRsp(&g.asm)
	rtgAsmSubRspImm(&g.asm, g.stackSize)
	if rtgTypeIsStruct(&g.meta, metaFn.resultType) {
		g.returnStruct = rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
		rtgAsmStoreRdiStack(&g.asm, g.returnStruct)
	}
	if !rtgBindFunctionParams(g, metaFn) {
		return false
	}
	if !rtgEmitLinearRange(g, fn.bodyStart+1, fn.bodyEnd) {
		return false
	}
	rtgAsmMovRaxImm(&g.asm, 0)
	rtgAsmLeave(&g.asm)
	rtgAsmRet(&g.asm)
	g.locals = oldLocals
	g.breakDepth = oldBreak
	g.continueDepth = oldContinue
	g.currentFunc = oldCurrent
	g.returnStruct = oldReturnStruct
	return true
}

func rtgBindFunctionParams(g *rtgLinearGen, fn rtgFuncInfo) bool {
	reg := 0
	if rtgTypeIsStruct(&g.meta, fn.resultType) {
		reg = 1
	}
	for i := 0; i < fn.paramCount; i++ {
		param := g.meta.params[fn.firstParam+i]
		offset := rtgAddTypedLocal(g, param.nameStart, param.nameEnd, param.typ)
		if rtgTypeIsSlice(&g.meta, param.typ) {
			if !rtgStoreParamWord(g, reg, offset) || !rtgStoreParamWord(g, reg+1, offset-8) || !rtgStoreParamWord(g, reg+2, offset-16) {
				return false
			}
			reg += 3
			continue
		}
		if rtgTypeIsString(&g.meta, param.typ) {
			if !rtgStoreParamWord(g, reg, offset) || !rtgStoreParamWord(g, reg+1, offset-8) {
				return false
			}
			reg += 2
			continue
		}
		if rtgTypeIsStruct(&g.meta, param.typ) {
			size := rtgTypeSize(&g.meta, param.typ)
			for at := 0; at < size; at += 8 {
				if !rtgStoreParamWord(g, reg, offset-at) {
					return false
				}
				reg++
			}
			continue
		}
		if !rtgStoreParamWord(g, reg, offset) {
			return false
		}
		reg++
	}
	return true
}

func rtgStoreParamWord(g *rtgLinearGen, reg int, offset int) bool {
	if reg == 0 {
		rtgAsmStoreRdiStack(&g.asm, offset)
		return true
	}
	if reg == 1 {
		rtgAsmStoreRsiStack(&g.asm, offset)
		return true
	}
	if reg == 2 {
		rtgAsmStoreRdxStack(&g.asm, offset)
		return true
	}
	if reg == 3 {
		rtgAsmStoreRcxStack(&g.asm, offset)
		return true
	}
	if reg == 4 {
		rtgAsmStoreR8Stack(&g.asm, offset)
		return true
	}
	if reg == 5 {
		rtgAsmStoreR9Stack(&g.asm, offset)
		return true
	}
	rtgAsmLoadRaxFrameArg(&g.asm, 16+(reg-6)*8)
	rtgAsmStoreRaxStack(&g.asm, offset)
	return true
}

func rtgTryCompileLinearAppMain(p rtgProgram, meta rtgMeta) rtgCompileResult {
	fnIndex := rtgFindFuncByName(p, "appMain")
	if fnIndex < 0 {
		var r rtgCompileResult
		return r
	}
	fn := p.funcs[fnIndex]
	body := rtgParseFunctionBody(p, fn)
	if !body.ok {
		var r rtgCompileResult
		return r
	}
	var g rtgLinearGen
	g.prog = p
	g.meta = meta
	g.stackSize = 4096
	rtgAsmInit(&g.asm)
	rtgAsmPushRbp(&g.asm)
	rtgAsmMovRbpRsp(&g.asm)
	rtgAsmSubRspImm(&g.asm, g.stackSize)
	if !rtgLinearInitGlobals(&g) {
		var r rtgCompileResult
		return r
	}
	if !rtgEmitLinearRange(&g, fn.bodyStart+1, fn.bodyEnd) {
		var r rtgCompileResult
		return r
	}
	rtgAsmMovRaxImm(&g.asm, 60)
	rtgAsmMovRdiImm(&g.asm, 0)
	rtgAsmSyscall(&g.asm)
	return rtgCompileResult{data: rtgAsmImage(&g.asm), ok: true}
}

func rtgEmitLinearRange(g *rtgLinearGen, start int, end int) bool {
	var bp rtgBodyParse
	bp.prog = g.prog
	bp.ok = true
	rtgParseStatements(&bp, start, end)
	if !bp.ok {
		return false
	}
	for i := 0; i < len(bp.stmts); i++ {
		if !rtgEmitLinearStmt(g, bp.stmts[i]) {
			return false
		}
	}
	return true
}

func rtgEmitLinearStmt(g *rtgLinearGen, stmt rtgStmt) bool {
	p := g.prog
	if stmt.kind == rtgStmtExpr {
		if rtgEmitLinearPrintStmt(&g.asm, p, stmt) {
			return true
		}
		if rtgEmitLinearIncDec(g, stmt.startTok, stmt.endTok) {
			return true
		}
		ep := rtgParseExpression(p, stmt.exprStart, stmt.exprEnd)
		if !ep.ok || len(ep.exprs) == 0 {
			return false
		}
		root := ep.exprs[len(ep.exprs)-1]
		if root.kind != rtgExprCall {
			return false
		}
		return rtgEmitIntExpr(g, ep, len(ep.exprs)-1)
	}
	if stmt.kind == rtgStmtVar || stmt.kind == rtgStmtShort || stmt.kind == rtgStmtAssign {
		return rtgEmitLinearAssign(g, stmt)
	}
	if stmt.kind == rtgStmtReturn {
		if stmt.exprStart == stmt.exprEnd {
			rtgAsmMovRaxImm(&g.asm, 0)
			if g.returnExits {
				rtgAsmMovRdiRax(&g.asm)
				rtgAsmMovRaxImm(&g.asm, 60)
				rtgAsmSyscall(&g.asm)
			} else {
				rtgAsmLeave(&g.asm)
				rtgAsmRet(&g.asm)
			}
			return true
		}
		ep := rtgParseExpression(p, stmt.exprStart, stmt.exprEnd)
		if !ep.ok || len(ep.exprs) == 0 {
			return false
		}
		if rtgTypeIsStruct(&g.meta, rtgCurrentResultType(g)) {
			if !rtgEmitStructReturnExpr(g, ep, len(ep.exprs)-1) {
				return false
			}
		} else if rtgTypeIsSlice(&g.meta, rtgCurrentResultType(g)) {
			if !rtgEmitSliceReturnExpr(g, ep, len(ep.exprs)-1) {
				return false
			}
		} else {
			if !rtgEmitIntExpr(g, ep, len(ep.exprs)-1) {
				return false
			}
		}
		if g.returnExits {
			rtgAsmMovRdiRax(&g.asm)
			rtgAsmMovRaxImm(&g.asm, 60)
			rtgAsmSyscall(&g.asm)
		} else {
			rtgAsmLeave(&g.asm)
			rtgAsmRet(&g.asm)
		}
		return true
	}
	if stmt.kind == rtgStmtIf {
		return rtgEmitLinearIf(g, stmt)
	}
	if stmt.kind == rtgStmtFor {
		return rtgEmitLinearFor(g, stmt)
	}
	if stmt.kind == rtgStmtBreak {
		if g.breakDepth == 0 {
			return false
		}
		rtgAsmJmpLabel(&g.asm, g.breakLabels[g.breakDepth-1])
		return true
	}
	if stmt.kind == rtgStmtContinue {
		if g.continueDepth == 0 {
			return false
		}
		rtgAsmJmpLabel(&g.asm, g.continueLabels[g.continueDepth-1])
		return true
	}
	return false
}

func rtgEmitLinearIf(g *rtgLinearGen, stmt rtgStmt) bool {
	p := g.prog
	ep := rtgParseExpression(p, stmt.exprStart, stmt.exprEnd)
	if !ep.ok || len(ep.exprs) == 0 {
		return false
	}
	if !rtgEmitIntExpr(g, ep, len(ep.exprs)-1) {
		return false
	}
	elseLabel := rtgAsmNewLabel(&g.asm)
	endLabel := rtgAsmNewLabel(&g.asm)
	rtgAsmCmpRaxImm8(&g.asm, 0)
	rtgAsmJzLabel(&g.asm, elseLabel)
	if !rtgEmitLinearRange(g, stmt.bodyStart, stmt.bodyEnd) {
		return false
	}
	rtgAsmJmpLabel(&g.asm, endLabel)
	rtgAsmMarkLabel(&g.asm, elseLabel)
	if stmt.elseStart > 0 {
		if rtgTokIsKeyword(p, stmt.elseStart, rtgTokIf) && rtgTokIsKeyword(p, stmt.elseStart-1, rtgTokElse) {
			var nested rtgBodyParse
			nested.prog = p
			nested.ok = true
			next := rtgParseOneStatement(&nested, stmt.elseStart, stmt.elseEnd)
			if !nested.ok || next != stmt.elseEnd || len(nested.stmts) != 1 {
				return false
			}
			if !rtgEmitLinearStmt(g, nested.stmts[0]) {
				return false
			}
		} else {
			if !rtgEmitLinearRange(g, stmt.elseStart, stmt.elseEnd) {
				return false
			}
		}
	}
	rtgAsmMarkLabel(&g.asm, endLabel)
	return true
}

func rtgEmitLinearFor(g *rtgLinearGen, stmt rtgStmt) bool {
	p := g.prog
	semi1 := rtgFindTokenTextAfter(p, stmt.exprStart, stmt.exprEnd, ";")
	if semi1 >= stmt.exprStart {
		return rtgEmitLinearClassicFor(g, stmt, semi1)
	}
	startLabel := rtgAsmNewLabel(&g.asm)
	endLabel := rtgAsmNewLabel(&g.asm)
	if len(g.breakLabels) <= g.breakDepth {
		g.breakLabels = append(g.breakLabels, endLabel)
	} else {
		g.breakLabels[g.breakDepth] = endLabel
	}
	if len(g.continueLabels) <= g.continueDepth {
		g.continueLabels = append(g.continueLabels, startLabel)
	} else {
		g.continueLabels[g.continueDepth] = startLabel
	}
	g.breakDepth++
	g.continueDepth++
	rtgAsmMarkLabel(&g.asm, startLabel)
	if stmt.exprStart < stmt.exprEnd {
		ep := rtgParseExpression(p, stmt.exprStart, stmt.exprEnd)
		if !ep.ok || len(ep.exprs) == 0 {
			return false
		}
		if !rtgEmitIntExpr(g, ep, len(ep.exprs)-1) {
			return false
		}
		rtgAsmCmpRaxImm8(&g.asm, 0)
		rtgAsmJzLabel(&g.asm, endLabel)
	}
	if !rtgEmitLinearRange(g, stmt.bodyStart, stmt.bodyEnd) {
		return false
	}
	rtgAsmJmpLabel(&g.asm, startLabel)
	rtgAsmMarkLabel(&g.asm, endLabel)
	g.breakDepth--
	g.continueDepth--
	return true
}

func rtgEmitLinearClassicFor(g *rtgLinearGen, stmt rtgStmt, semi1 int) bool {
	p := g.prog
	semi2 := rtgFindTokenTextAfter(p, semi1+1, stmt.exprEnd, ";")
	if semi2 <= semi1 {
		return false
	}
	if !rtgEmitLinearSimpleRange(g, stmt.exprStart, semi1) {
		return false
	}
	startLabel := rtgAsmNewLabel(&g.asm)
	postLabel := rtgAsmNewLabel(&g.asm)
	endLabel := rtgAsmNewLabel(&g.asm)
	if len(g.breakLabels) <= g.breakDepth {
		g.breakLabels = append(g.breakLabels, endLabel)
	} else {
		g.breakLabels[g.breakDepth] = endLabel
	}
	if len(g.continueLabels) <= g.continueDepth {
		g.continueLabels = append(g.continueLabels, postLabel)
	} else {
		g.continueLabels[g.continueDepth] = postLabel
	}
	g.breakDepth++
	g.continueDepth++
	rtgAsmMarkLabel(&g.asm, startLabel)
	if semi1+1 < semi2 {
		ep := rtgParseExpression(p, semi1+1, semi2)
		if !ep.ok || len(ep.exprs) == 0 {
			return false
		}
		if !rtgEmitIntExpr(g, ep, len(ep.exprs)-1) {
			return false
		}
		rtgAsmCmpRaxImm8(&g.asm, 0)
		rtgAsmJzLabel(&g.asm, endLabel)
	}
	if !rtgEmitLinearRange(g, stmt.bodyStart, stmt.bodyEnd) {
		return false
	}
	rtgAsmMarkLabel(&g.asm, postLabel)
	if !rtgEmitLinearSimpleRange(g, semi2+1, stmt.exprEnd) {
		return false
	}
	rtgAsmJmpLabel(&g.asm, startLabel)
	rtgAsmMarkLabel(&g.asm, endLabel)
	g.breakDepth--
	g.continueDepth--
	return true
}

func rtgEmitLinearSimpleRange(g *rtgLinearGen, start int, end int) bool {
	p := g.prog
	if start >= end {
		return true
	}
	if rtgEmitLinearIncDec(g, start, end) {
		return true
	}
	assignTok := rtgFindAssignmentToken(p, start, end)
	if assignTok > start {
		kind := rtgStmtAssign
		if rtgTokTextIs(p, assignTok, ":=") {
			kind = rtgStmtShort
		}
		nameStart := 0
		nameEnd := 0
		if rtgTokIsKind(p, start, rtgTokIdent) {
			nameStart = p.toks[start].start
			nameEnd = p.toks[start].end
		}
		return rtgEmitLinearAssign(g, rtgStmt{kind: kind, startTok: start, endTok: end, exprStart: assignTok + 1, exprEnd: end, nameStart: nameStart, nameEnd: nameEnd})
	}
	return false
}

func rtgEmitLinearIncDec(g *rtgLinearGen, start int, end int) bool {
	p := g.prog
	if start+2 != end || !rtgTokIsKind(p, start, rtgTokIdent) {
		return false
	}
	if !rtgTokTextIs(p, start+1, "++") && !rtgTokTextIs(p, start+1, "--") {
		return false
	}
	offset := rtgFindLocalOffset(g, p.toks[start].start, p.toks[start].end)
	if offset < 0 {
		return false
	}
	rtgAsmLoadRaxStack(&g.asm, offset)
	rtgAsmPushRax(&g.asm)
	rtgAsmMovRaxImm(&g.asm, 1)
	rtgAsmPopRcx(&g.asm)
	if rtgTokTextIs(p, start+1, "++") {
		rtgAsmAddRaxRcx(&g.asm)
	} else {
		rtgAsmSubLeftRcxRightRax(&g.asm)
	}
	rtgAsmStoreRaxStack(&g.asm, offset)
	return true
}

func rtgEmitLinearPrintStmt(a *rtgAsm, p rtgProgram, stmt rtgStmt) bool {
	if !rtgTokTextIs(p, stmt.exprStart, "print") {
		return false
	}
	if !rtgTokTextIs(p, stmt.exprStart+1, "(") || !rtgTokIsKind(p, stmt.exprStart+2, rtgTokString) || !rtgTokTextIs(p, stmt.exprStart+3, ")") {
		return false
	}
	msg := rtgDecodeStringToken(p.src, p.toks[stmt.exprStart+2])
	msgOff := len(a.data)
	for i := 0; i < len(msg); i++ {
		a.data = append(a.data, msg[i])
	}
	rtgAsmMovRaxImm(a, 1)
	rtgAsmMovRdiImm(a, 1)
	rtgAsmLeaRsiData(a, msgOff)
	rtgAsmMovRdxImm(a, len(msg))
	rtgAsmSyscall(a)
	return true
}

func rtgLinearInitGlobals(g *rtgLinearGen) bool {
	for i := 0; i < len(g.meta.globals); i++ {
		s := g.meta.globals[i]
		if s.kind != rtgTokVar {
			continue
		}
		off := len(g.globals) * 8
		g.globals = append(g.globals, rtgGlobalInfo{nameStart: s.nameStart, nameEnd: s.nameEnd, offset: off})
		g.asm.bssSize += 8
		if s.initStart < s.initEnd {
			ep := rtgParseExpression(g.prog, s.initStart, s.initEnd)
			if !ep.ok || len(ep.exprs) == 0 {
				return false
			}
			constResult := rtgEvalConstExpr(g, ep, len(ep.exprs)-1)
			if !constResult.ok {
				return false
			}
			rtgAsmMovRaxImm(&g.asm, constResult.value)
			rtgAsmStoreRaxBss(&g.asm, off)
		}
	}
	return true
}

func rtgEmitLinearAssign(g *rtgLinearGen, stmt rtgStmt) bool {
	p := g.prog
	nameStart := stmt.nameStart
	nameEnd := stmt.nameEnd
	if nameEnd <= nameStart {
		return false
	}
	assignTok := rtgFindAssignmentToken(p, stmt.startTok, stmt.endTok)
	offset := rtgFindLocalOffset(g, nameStart, nameEnd)
	globalOffset := -1
	fieldStackOffset := -1
	if rtgTokIsKind(p, stmt.startTok, rtgTokIdent) && rtgTokTextIs(p, stmt.startTok+1, ".") && rtgTokIsKind(p, stmt.startTok+2, rtgTokIdent) {
		localIndex := rtgFindLocalIndex(g, p.toks[stmt.startTok].start, p.toks[stmt.startTok].end)
		if localIndex < 0 {
			return false
		}
		fieldOffset := rtgStructFieldOffset(g, g.locals[localIndex].typ, p.toks[stmt.startTok+2].start, p.toks[stmt.startTok+2].end)
		if fieldOffset < 0 {
			return false
		}
		fieldStackOffset = g.locals[localIndex].offset - fieldOffset
		offset = fieldStackOffset
	}
	if offset < 0 {
		if stmt.kind == rtgStmtAssign {
			globalOffset = rtgFindGlobalOffset(g, nameStart, nameEnd)
			if globalOffset < 0 {
				return false
			}
		} else {
			localType := rtgTypeInt
			if stmt.kind == rtgStmtVar {
				typeEnd := assignTok
				if assignTok <= stmt.startTok {
					typeEnd = stmt.endTok
				}
				if stmt.startTok+2 < typeEnd {
					typeResult := rtgParseType(&g.meta, stmt.startTok+2, typeEnd)
					if typeResult.typ != 0 {
						localType = typeResult.typ
					}
				}
			}
			if stmt.kind == rtgStmtShort {
				inferredType := rtgInferExprType(g, assignTok+1, stmt.endTok)
				if inferredType != 0 {
					localType = inferredType
				}
			}
			offset = rtgAddTypedLocal(g, nameStart, nameEnd, localType)
		}
	}
	if assignTok <= stmt.startTok {
		if globalOffset >= 0 {
			rtgAsmMovRaxImm(&g.asm, 0)
			rtgAsmStoreRaxBss(&g.asm, globalOffset)
		} else {
			rtgZeroLocalAtOffset(g, offset)
		}
		return true
	}
	ep := rtgParseExpression(p, assignTok+1, stmt.endTok)
	if !ep.ok || len(ep.exprs) == 0 {
		return false
	}
	if rtgEmitAppendAssignGeneral(g, stmt, ep) {
		return true
	}
	if rtgEmitAppendAssign(g, stmt, ep, offset, globalOffset) {
		return true
	}
	if rtgTokTextIs(p, assignTok, "+=") || rtgTokTextIs(p, assignTok, "-=") || rtgTokTextIs(p, assignTok, "*=") || rtgTokTextIs(p, assignTok, "/=") || rtgTokTextIs(p, assignTok, "%=") {
		if globalOffset >= 0 {
			rtgAsmLoadRaxBss(&g.asm, globalOffset)
		} else {
			rtgAsmLoadRaxStack(&g.asm, offset)
		}
		rtgAsmPushRax(&g.asm)
		if !rtgEmitIntExpr(g, ep, len(ep.exprs)-1) {
			return false
		}
		rtgAsmPopRcx(&g.asm)
		if rtgTokTextIs(p, assignTok, "+=") {
			rtgAsmAddRaxRcx(&g.asm)
		} else if rtgTokTextIs(p, assignTok, "-=") {
			rtgAsmSubLeftRcxRightRax(&g.asm)
		} else if rtgTokTextIs(p, assignTok, "*=") {
			rtgAsmMulRaxRcx(&g.asm)
		} else if rtgTokTextIs(p, assignTok, "/=") {
			rtgAsmDivLeftRcxRightRax(&g.asm, false)
		} else {
			rtgAsmDivLeftRcxRightRax(&g.asm, true)
		}
		if globalOffset >= 0 {
			rtgAsmStoreRaxBss(&g.asm, globalOffset)
		} else {
			rtgAsmStoreRaxStack(&g.asm, offset)
		}
		return true
	}
	if globalOffset < 0 && rtgEmitTypedAssign(g, ep, len(ep.exprs)-1, offset) {
		return true
	}
	if !rtgEmitIntExpr(g, ep, len(ep.exprs)-1) {
		return false
	}
	if globalOffset >= 0 {
		rtgAsmStoreRaxBss(&g.asm, globalOffset)
	} else {
		rtgAsmStoreRaxStack(&g.asm, offset)
	}
	return true
}

func rtgCurrentResultType(g *rtgLinearGen) int {
	if g.currentFunc >= 0 && g.currentFunc < len(g.meta.funcs) {
		return g.meta.funcs[g.currentFunc].resultType
	}
	return 0
}

func rtgInferExprType(g *rtgLinearGen, start int, end int) int {
	ep := rtgParseExpression(g.prog, start, end)
	if !ep.ok || len(ep.exprs) == 0 {
		return 0
	}
	return rtgInferParsedExprType(g, ep, len(ep.exprs)-1)
}

func rtgInferParsedExprType(g *rtgLinearGen, ep rtgExprParse, idx int) int {
	if idx < 0 || idx >= len(ep.exprs) {
		return 0
	}
	e := ep.exprs[idx]
	if e.kind == rtgExprInt || e.kind == rtgExprChar || e.kind == rtgExprBool {
		return rtgTypeInt
	}
	if e.kind == rtgExprString {
		return rtgTypeString
	}
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			return g.locals[localIndex].typ
		}
		for i := 0; i < len(g.meta.globals); i++ {
			s := g.meta.globals[i]
			if rtgBytesEqualRange(g.prog.src, s.nameStart, s.nameEnd, e.nameStart, e.nameEnd) {
				return s.typ
			}
		}
		return rtgTypeInt
	}
	if e.kind == rtgExprCall {
		if rtgExprIsIdentText(g.prog, ep, e.left, "append") && e.argCount == 2 {
			return rtgInferParsedExprType(g, ep, ep.args[e.firstArg])
		}
		if rtgExprIsIdentText(g.prog, ep, e.left, "int") || rtgExprIsIdentText(g.prog, ep, e.left, "int64") || rtgExprIsIdentText(g.prog, ep, e.left, "byte") || rtgExprIsIdentText(g.prog, ep, e.left, "len") || rtgExprIsIdentText(g.prog, ep, e.left, "open") || rtgExprIsIdentText(g.prog, ep, e.left, "close") || rtgExprIsIdentText(g.prog, ep, e.left, "read") || rtgExprIsIdentText(g.prog, ep, e.left, "write") || rtgExprIsIdentText(g.prog, ep, e.left, "chmod") {
			return rtgTypeInt
		}
		fnIndex := rtgFuncInfoFromCall(g, ep, e.left)
		if fnIndex >= 0 {
			return g.meta.funcs[fnIndex].resultType
		}
	}
	if e.kind == rtgExprIndex {
		leftType := rtgInferParsedExprType(g, ep, e.left)
		t := rtgResolveType(&g.meta, leftType)
		if t.kind == rtgTypeSlice {
			return t.elem
		}
		if t.kind == rtgTypeString {
			return rtgTypeByte
		}
	}
	if e.kind == rtgExprSelector {
		baseType := rtgInferParsedExprType(g, ep, e.left)
		t := rtgResolveType(&g.meta, baseType)
		if t.kind == rtgTypePointer {
			t = rtgResolveType(&g.meta, t.elem)
		}
		if t.kind == rtgTypeStruct {
			for i := 0; i < t.count; i++ {
				field := g.meta.fields[t.first+i]
				if rtgBytesEqualRange(g.prog.src, field.nameStart, field.nameEnd, e.nameStart, e.nameEnd) {
					return field.typ
				}
			}
		}
	}
	if e.kind == rtgExprComposite {
		return rtgNamedTypeFromToken(&g.meta, g.prog.toks[e.tok])
	}
	return rtgTypeInt
}

func rtgLocalTypeAtOffset(g *rtgLinearGen, offset int) int {
	for i := 0; i < len(g.locals); i++ {
		if g.locals[i].offset == offset {
			return g.locals[i].typ
		}
	}
	return 0
}

func rtgEmitTypedAssign(g *rtgLinearGen, ep rtgExprParse, idx int, offset int) bool {
	destType := rtgLocalTypeAtOffset(g, offset)
	if idx < 0 || idx >= len(ep.exprs) {
		return false
	}
	e := ep.exprs[idx]
	destResolved := rtgResolveType(&g.meta, destType)
	if destResolved.kind == rtgTypeStruct {
		if e.kind == rtgExprCall {
			return rtgEmitStructCallToLocal(g, ep, e, destType, offset)
		}
		if e.kind == rtgExprIndex {
			return rtgEmitIndexedStructToLocal(g, ep, e, destType, offset)
		}
		return false
	}
	if !rtgTypeIsSlice(&g.meta, destType) {
		return false
	}
	if e.kind == rtgExprCall {
		fnIndex := rtgFuncInfoFromCall(g, ep, e.left)
		if fnIndex < 0 || !rtgTypeIsSlice(&g.meta, g.meta.funcs[fnIndex].resultType) {
			return false
		}
		if !rtgEmitUserCall(g, ep, e) {
			return false
		}
		rtgAsmStoreRaxStack(&g.asm, offset)
		rtgAsmStoreRdxStack(&g.asm, offset-8)
		rtgAsmStoreRcxStack(&g.asm, offset-16)
		return true
	}
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 || !rtgTypeIsSlice(&g.meta, g.locals[localIndex].typ) {
			return false
		}
		rtgAsmLoadRaxStack(&g.asm, g.locals[localIndex].offset)
		rtgAsmStoreRaxStack(&g.asm, offset)
		rtgAsmLoadRaxStack(&g.asm, g.locals[localIndex].offset-8)
		rtgAsmStoreRaxStack(&g.asm, offset-8)
		rtgAsmLoadRaxStack(&g.asm, g.locals[localIndex].offset-16)
		rtgAsmStoreRaxStack(&g.asm, offset-16)
		return true
	}
	return false
}

func rtgEmitIndexedStructToLocal(g *rtgLinearGen, ep rtgExprParse, e rtgExpr, destType int, offset int) bool {
	leftType := rtgInferParsedExprType(g, ep, e.left)
	sliceType := rtgResolveType(&g.meta, leftType)
	if sliceType.kind != rtgTypeSlice {
		return false
	}
	elemType := rtgResolveType(&g.meta, sliceType.elem)
	destResolved := rtgResolveType(&g.meta, destType)
	if elemType.kind != rtgTypeStruct || destResolved.kind != rtgTypeStruct {
		return false
	}
	elemSize := rtgTypeSize(&g.meta, sliceType.elem)
	if rtgTypeSize(&g.meta, destType) != elemSize {
		return false
	}
	if !rtgEmitIntExpr(g, ep, e.right) {
		return false
	}
	rtgAsmPushRax(&g.asm)
	if !rtgEmitSlicePtrLen(g, ep, e.left) {
		return false
	}
	rtgAsmPopRcx(&g.asm)
	rtgAsmImulRcxImm(&g.asm, elemSize)
	rtgAsmMovRdxRax(&g.asm)
	rtgAsmAddRdxRcx(&g.asm)
	for at := 0; at < elemSize; at += 8 {
		rtgAsmLoadRaxMemRdxDisp(&g.asm, at)
		rtgAsmStoreRaxStack(&g.asm, offset-at)
	}
	return true
}

func rtgEmitStructCallToLocal(g *rtgLinearGen, ep rtgExprParse, e rtgExpr, destType int, offset int) bool {
	fnIndex := rtgFuncInfoFromCall(g, ep, e.left)
	if fnIndex < 0 || !rtgTypeIsStruct(&g.meta, g.meta.funcs[fnIndex].resultType) {
		return false
	}
	if rtgTypeSize(&g.meta, destType) != rtgTypeSize(&g.meta, g.meta.funcs[fnIndex].resultType) {
		return false
	}
	wordCount := 1
	for i := e.argCount - 1; i >= 0; i-- {
		words := rtgEmitCallArgReverse(g, ep, ep.args[e.firstArg+i])
		if words < 0 {
			return false
		}
		wordCount += words
	}
	rtgAsmLeaRaxStack(&g.asm, offset)
	rtgAsmPushRax(&g.asm)
	if wordCount > 0 {
		rtgAsmPopRdi(&g.asm)
	}
	if wordCount > 1 {
		rtgAsmPopRsi(&g.asm)
	}
	if wordCount > 2 {
		rtgAsmPopRdx(&g.asm)
	}
	if wordCount > 3 {
		rtgAsmPopRcx(&g.asm)
	}
	if wordCount > 4 {
		rtgAsmPopR8(&g.asm)
	}
	if wordCount > 5 {
		rtgAsmPopR9(&g.asm)
	}
	rtgAsmCallLabel(&g.asm, g.funcLabels[fnIndex])
	if wordCount > 6 {
		rtgAsmAddRspImm(&g.asm, (wordCount-6)*8)
	}
	return true
}

func rtgEmitStructReturnExpr(g *rtgLinearGen, ep rtgExprParse, idx int) bool {
	if g.returnStruct <= 0 || idx < 0 || idx >= len(ep.exprs) {
		return false
	}
	e := ep.exprs[idx]
	resultType := rtgCurrentResultType(g)
	size := rtgTypeSize(&g.meta, resultType)
	rtgAsmLoadRdxStack(&g.asm, g.returnStruct)
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 || rtgTypeSize(&g.meta, g.locals[localIndex].typ) != size {
			return false
		}
		for at := 0; at < size; at += 8 {
			rtgAsmLoadRaxStack(&g.asm, g.locals[localIndex].offset-at)
			rtgAsmStoreRaxMemRdxDisp(&g.asm, at)
		}
		return true
	}
	if e.kind == rtgExprComposite {
		rtgAsmMovRaxImm(&g.asm, 0)
		for at := 0; at < size; at += 8 {
			rtgAsmStoreRaxMemRdxDisp(&g.asm, at)
		}
		for i := 0; i < e.argCount; i++ {
			field := ep.fields[e.firstArg+i]
			fieldOffset := rtgStructFieldOffset(g, resultType, field.nameStart, field.nameEnd)
			if fieldOffset < 0 {
				return false
			}
			if !rtgEmitIntExpr(g, ep, field.expr) {
				return false
			}
			rtgAsmLoadRdxStack(&g.asm, g.returnStruct)
			rtgAsmStoreRaxMemRdxDisp(&g.asm, fieldOffset)
		}
		return true
	}
	return false
}

func rtgEmitSliceReturnExpr(g *rtgLinearGen, ep rtgExprParse, idx int) bool {
	if idx < 0 || idx >= len(ep.exprs) {
		return false
	}
	e := ep.exprs[idx]
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 || !rtgTypeIsSlice(&g.meta, g.locals[localIndex].typ) {
			return false
		}
		rtgAsmLoadRaxStack(&g.asm, g.locals[localIndex].offset)
		rtgAsmLoadRdxStack(&g.asm, g.locals[localIndex].offset-8)
		rtgAsmLoadRcxStack(&g.asm, g.locals[localIndex].offset-16)
		return true
	}
	if e.kind == rtgExprCall {
		fnIndex := rtgFuncInfoFromCall(g, ep, e.left)
		if fnIndex < 0 || !rtgTypeIsSlice(&g.meta, g.meta.funcs[fnIndex].resultType) {
			return false
		}
		return rtgEmitUserCall(g, ep, e)
	}
	return false
}

func rtgEmitUserCall(g *rtgLinearGen, ep rtgExprParse, e rtgExpr) bool {
	a := &g.asm
	fnIndex := rtgFuncInfoFromCall(g, ep, e.left)
	if fnIndex < 0 {
		return false
	}
	wordCount := 0
	for i := e.argCount - 1; i >= 0; i-- {
		words := rtgEmitCallArgReverse(g, ep, ep.args[e.firstArg+i])
		if words < 0 {
			return false
		}
		wordCount += words
	}
	if wordCount > 0 {
		rtgAsmPopRdi(a)
	}
	if wordCount > 1 {
		rtgAsmPopRsi(a)
	}
	if wordCount > 2 {
		rtgAsmPopRdx(a)
	}
	if wordCount > 3 {
		rtgAsmPopRcx(a)
	}
	if wordCount > 4 {
		rtgAsmPopR8(a)
	}
	if wordCount > 5 {
		rtgAsmPopR9(a)
	}
	rtgAsmCallLabel(a, g.funcLabels[fnIndex])
	if wordCount > 6 {
		rtgAsmAddRspImm(a, (wordCount-6)*8)
	}
	return true
}

func rtgEmitCallArgReverse(g *rtgLinearGen, ep rtgExprParse, idx int) int {
	typ := rtgInferParsedExprType(g, ep, idx)
	if rtgTypeIsSlice(&g.meta, typ) {
		if rtgEmitSliceArgReverse(g, ep, idx) {
			return 3
		}
		return -1
	}
	if rtgTypeIsString(&g.meta, typ) {
		if rtgEmitStringArgReverse(g, ep, idx) {
			return 2
		}
		return -1
	}
	if rtgTypeIsStruct(&g.meta, typ) {
		return rtgEmitStructArgReverse(g, ep, idx, typ)
	}
	if !rtgEmitIntExpr(g, ep, idx) {
		return -1
	}
	rtgAsmPushRax(&g.asm)
	return 1
}

func rtgEmitStructArgReverse(g *rtgLinearGen, ep rtgExprParse, idx int, typ int) int {
	if idx < 0 || idx >= len(ep.exprs) {
		return -1
	}
	size := rtgTypeSize(&g.meta, typ)
	if size <= 0 {
		return -1
	}
	e := ep.exprs[idx]
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 || rtgTypeSize(&g.meta, g.locals[localIndex].typ) != size {
			return -1
		}
		for at := size - 8; at >= 0; at -= 8 {
			rtgAsmLoadRaxStack(&g.asm, g.locals[localIndex].offset-at)
			rtgAsmPushRax(&g.asm)
		}
		return size / 8
	}
	if e.kind == rtgExprIndex {
		leftType := rtgInferParsedExprType(g, ep, e.left)
		sliceType := rtgResolveType(&g.meta, leftType)
		elemType := rtgResolveType(&g.meta, sliceType.elem)
		if sliceType.kind != rtgTypeSlice || elemType.kind != rtgTypeStruct || rtgTypeSize(&g.meta, sliceType.elem) != size {
			return -1
		}
		if !rtgEmitIntExpr(g, ep, e.right) {
			return -1
		}
		rtgAsmPushRax(&g.asm)
		if !rtgEmitSlicePtrLen(g, ep, e.left) {
			return -1
		}
		rtgAsmPopRcx(&g.asm)
		rtgAsmImulRcxImm(&g.asm, size)
		rtgAsmMovRdxRax(&g.asm)
		rtgAsmAddRdxRcx(&g.asm)
		for at := size - 8; at >= 0; at -= 8 {
			rtgAsmLoadRaxMemRdxDisp(&g.asm, at)
			rtgAsmPushRax(&g.asm)
		}
		return size / 8
	}
	if e.kind == rtgExprSelector {
		base := ep.exprs[e.left]
		if base.kind != rtgExprIdent {
			return -1
		}
		localIndex := rtgFindLocalIndex(g, base.nameStart, base.nameEnd)
		if localIndex < 0 {
			return -1
		}
		fieldOffset := rtgStructFieldOffset(g, g.locals[localIndex].typ, e.nameStart, e.nameEnd)
		if fieldOffset < 0 {
			return -1
		}
		fieldType := rtgStructFieldType(g, g.locals[localIndex].typ, e.nameStart, e.nameEnd)
		if !rtgTypeIsStruct(&g.meta, fieldType) || rtgTypeSize(&g.meta, fieldType) != size {
			return -1
		}
		stackOffset := g.locals[localIndex].offset - fieldOffset
		for at := size - 8; at >= 0; at -= 8 {
			rtgAsmLoadRaxStack(&g.asm, stackOffset-at)
			rtgAsmPushRax(&g.asm)
		}
		return size / 8
	}
	return -1
}

func rtgEmitSliceArgReverse(g *rtgLinearGen, ep rtgExprParse, idx int) bool {
	if idx < 0 || idx >= len(ep.exprs) {
		return false
	}
	e := ep.exprs[idx]
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 || !rtgTypeIsSlice(&g.meta, g.locals[localIndex].typ) {
			return false
		}
		rtgAsmLoadRaxStack(&g.asm, g.locals[localIndex].offset-16)
		rtgAsmPushRax(&g.asm)
		rtgAsmLoadRaxStack(&g.asm, g.locals[localIndex].offset-8)
		rtgAsmPushRax(&g.asm)
		rtgAsmLoadRaxStack(&g.asm, g.locals[localIndex].offset)
		rtgAsmPushRax(&g.asm)
		return true
	}
	if e.kind == rtgExprSelector {
		fieldType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsSlice(&g.meta, fieldType) {
			return false
		}
		if !rtgEmitSelectorAddressRdx(g, ep, e) {
			return false
		}
		rtgAsmLoadRaxMemRdxDisp(&g.asm, 16)
		rtgAsmPushRax(&g.asm)
		rtgAsmLoadRaxMemRdxDisp(&g.asm, 8)
		rtgAsmPushRax(&g.asm)
		rtgAsmLoadRaxMemRdxDisp(&g.asm, 0)
		rtgAsmPushRax(&g.asm)
		return true
	}
	if e.kind == rtgExprCall {
		fnIndex := rtgFuncInfoFromCall(g, ep, e.left)
		if fnIndex < 0 || !rtgTypeIsSlice(&g.meta, g.meta.funcs[fnIndex].resultType) {
			return false
		}
		if !rtgEmitUserCall(g, ep, e) {
			return false
		}
		rtgAsmPushRcx(&g.asm)
		rtgAsmPushRdx(&g.asm)
		rtgAsmPushRax(&g.asm)
		return true
	}
	return false
}

func rtgEmitStringArgReverse(g *rtgLinearGen, ep rtgExprParse, idx int) bool {
	if idx < 0 || idx >= len(ep.exprs) {
		return false
	}
	e := ep.exprs[idx]
	if e.kind == rtgExprString {
		msg := rtgDecodeStringToken(g.prog.src, g.prog.toks[e.tok])
		msgOff := len(g.asm.data)
		for i := 0; i < len(msg); i++ {
			g.asm.data = append(g.asm.data, msg[i])
		}
		g.asm.data = append(g.asm.data, 0)
		rtgAsmMovRaxImm(&g.asm, len(msg))
		rtgAsmPushRax(&g.asm)
		rtgAsmMovRaxDataAddr(&g.asm, msgOff)
		rtgAsmPushRax(&g.asm)
		return true
	}
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 || !rtgTypeIsString(&g.meta, g.locals[localIndex].typ) {
			return false
		}
		rtgAsmLoadRaxStack(&g.asm, g.locals[localIndex].offset-8)
		rtgAsmPushRax(&g.asm)
		rtgAsmLoadRaxStack(&g.asm, g.locals[localIndex].offset)
		rtgAsmPushRax(&g.asm)
		return true
	}
	return false
}

func rtgEmitIntExpr(g *rtgLinearGen, ep rtgExprParse, idx int) bool {
	p := g.prog
	a := &g.asm
	if idx < 0 || idx >= len(ep.exprs) {
		return false
	}
	e := ep.exprs[idx]
	if e.kind == rtgExprInt {
		rtgAsmMovRaxImm(a, rtgParseIntToken(p.src, p.toks[e.tok]))
		return true
	}
	if e.kind == rtgExprIdent {
		offset := rtgFindLocalOffset(g, e.nameStart, e.nameEnd)
		if offset < 0 {
			constResult := rtgEvalConstByName(g, e.nameStart, e.nameEnd)
			if !constResult.ok {
				globalOffset := rtgFindGlobalOffset(g, e.nameStart, e.nameEnd)
				if globalOffset < 0 {
					return false
				}
				rtgAsmLoadRaxBss(a, globalOffset)
				return true
			}
			rtgAsmMovRaxImm(a, constResult.value)
			return true
		}
		rtgAsmLoadRaxStack(a, offset)
		return true
	}
	if e.kind == rtgExprChar {
		rtgAsmMovRaxImm(a, rtgParseCharToken(p.src, p.toks[e.tok]))
		return true
	}
	if e.kind == rtgExprBool {
		if rtgBytesEqualText(p.src, p.toks[e.tok].start, p.toks[e.tok].end, "true") {
			rtgAsmMovRaxImm(a, 1)
		} else {
			rtgAsmMovRaxImm(a, 0)
		}
		return true
	}
	if e.kind == rtgExprCall {
		if e.argCount == 1 && (rtgExprIsIdentText(p, ep, e.left, "int") || rtgExprIsIdentText(p, ep, e.left, "int64")) {
			return rtgEmitIntExpr(g, ep, ep.args[e.firstArg])
		}
		if e.argCount == 1 && rtgExprIsIdentText(p, ep, e.left, "byte") {
			if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg]) {
				return false
			}
			rtgAsmAndRaxImm(&g.asm, 255)
			return true
		}
		if e.argCount == 1 && rtgExprIsIdentText(p, ep, e.left, "len") {
			arg := ep.exprs[ep.args[e.firstArg]]
			if arg.kind == rtgExprString {
				msg := rtgDecodeStringToken(p.src, p.toks[arg.tok])
				rtgAsmMovRaxImm(a, len(msg))
				return true
			}
			if arg.kind == rtgExprIdent {
				localIndex := rtgFindLocalIndex(g, arg.nameStart, arg.nameEnd)
				if localIndex >= 0 && (rtgTypeIsSlice(&g.meta, g.locals[localIndex].typ) || rtgTypeIsString(&g.meta, g.locals[localIndex].typ)) {
					rtgAsmLoadRaxStack(a, g.locals[localIndex].offset-8)
					return true
				}
			}
			if arg.kind == rtgExprSelector {
				if !rtgEmitSlicePtrLen(g, ep, ep.args[e.firstArg]) {
					return false
				}
				rtgAsmMovRaxRcx(a)
				return true
			}
		}
		if rtgExprIsIdentText(p, ep, e.left, "open") {
			return rtgEmitBuiltinOpen(g, ep, e)
		}
		if rtgExprIsIdentText(p, ep, e.left, "close") {
			return rtgEmitBuiltinClose(g, ep, e)
		}
		if rtgExprIsIdentText(p, ep, e.left, "chmod") {
			return rtgEmitBuiltinChmod(g, ep, e)
		}
		if rtgExprIsIdentText(p, ep, e.left, "read") {
			return rtgEmitBuiltinRead(g, ep, e)
		}
		if rtgExprIsIdentText(p, ep, e.left, "write") {
			return rtgEmitBuiltinWrite(g, ep, e)
		}
		return rtgEmitUserCall(g, ep, e)
	}
	if e.kind == rtgExprIndex {
		return rtgEmitIndexExpr(g, ep, e)
	}
	if e.kind == rtgExprSelector {
		base := ep.exprs[e.left]
		if base.kind == rtgExprIndex {
			return rtgEmitIndexedStructField(g, ep, base, e.nameStart, e.nameEnd)
		}
		if !rtgEmitSelectorAddressRdx(g, ep, e) {
			return false
		}
		rtgAsmLoadRaxMemRdxDisp(a, 0)
		return true
	}
	if e.kind == rtgExprUnary {
		if rtgTokTextIs(p, e.tok, "&") {
			inner := ep.exprs[e.left]
			if inner.kind == rtgExprIdent {
				localIndex := rtgFindLocalIndex(g, inner.nameStart, inner.nameEnd)
				if localIndex < 0 {
					return false
				}
				rtgAsmLeaRaxStack(a, g.locals[localIndex].offset)
				return true
			}
			if inner.kind == rtgExprSelector {
				if !rtgEmitSelectorAddressRdx(g, ep, inner) {
					return false
				}
				rtgAsmMovRaxRdx(a)
				return true
			}
			return false
		}
		if !rtgEmitIntExpr(g, ep, e.left) {
			return false
		}
		if rtgTokTextIs(p, e.tok, "-") {
			rtgAsmNegRax(a)
			return true
		}
		if rtgTokTextIs(p, e.tok, "+") {
			return true
		}
		if rtgTokTextIs(p, e.tok, "!") {
			rtgAsmBoolNotRax(a)
			return true
		}
		return false
	}
	if e.kind == rtgExprBinary {
		if rtgTokTextIs(p, e.tok, "&&") {
			falseLabel := rtgAsmNewLabel(a)
			endLabel := rtgAsmNewLabel(a)
			if !rtgEmitIntExpr(g, ep, e.left) {
				return false
			}
			rtgAsmCmpRaxImm8(a, 0)
			rtgAsmJzLabel(a, falseLabel)
			if !rtgEmitIntExpr(g, ep, e.right) {
				return false
			}
			rtgAsmCmpRaxImm8(a, 0)
			rtgAsmJzLabel(a, falseLabel)
			rtgAsmMovRaxImm(a, 1)
			rtgAsmJmpLabel(a, endLabel)
			rtgAsmMarkLabel(a, falseLabel)
			rtgAsmMovRaxImm(a, 0)
			rtgAsmMarkLabel(a, endLabel)
			return true
		}
		if rtgTokTextIs(p, e.tok, "||") {
			trueLabel := rtgAsmNewLabel(a)
			endLabel := rtgAsmNewLabel(a)
			if !rtgEmitIntExpr(g, ep, e.left) {
				return false
			}
			rtgAsmCmpRaxImm8(a, 0)
			rtgAsmJnzLabel(a, trueLabel)
			if !rtgEmitIntExpr(g, ep, e.right) {
				return false
			}
			rtgAsmCmpRaxImm8(a, 0)
			rtgAsmJnzLabel(a, trueLabel)
			rtgAsmMovRaxImm(a, 0)
			rtgAsmJmpLabel(a, endLabel)
			rtgAsmMarkLabel(a, trueLabel)
			rtgAsmMovRaxImm(a, 1)
			rtgAsmMarkLabel(a, endLabel)
			return true
		}
		if !rtgEmitIntExpr(g, ep, e.left) {
			return false
		}
		rtgAsmPushRax(a)
		if !rtgEmitIntExpr(g, ep, e.right) {
			return false
		}
		rtgAsmPopRcx(a)
		if rtgTokTextIs(p, e.tok, "+") {
			rtgAsmAddRaxRcx(a)
			return true
		}
		if rtgTokTextIs(p, e.tok, "-") {
			rtgAsmSubLeftRcxRightRax(a)
			return true
		}
		if rtgTokTextIs(p, e.tok, "*") {
			rtgAsmMulRaxRcx(a)
			return true
		}
		if rtgTokTextIs(p, e.tok, "/") {
			rtgAsmDivLeftRcxRightRax(a, false)
			return true
		}
		if rtgTokTextIs(p, e.tok, "%") {
			rtgAsmDivLeftRcxRightRax(a, true)
			return true
		}
		if rtgTokTextIs(p, e.tok, "&") {
			rtgAsmAndRaxRcx(a)
			return true
		}
		if rtgTokTextIs(p, e.tok, "|") {
			rtgAsmOrRaxRcx(a)
			return true
		}
		if rtgTokTextIs(p, e.tok, "^") {
			rtgAsmXorRaxRcx(a)
			return true
		}
		if rtgTokTextIs(p, e.tok, "&^") {
			rtgAsmAndNotLeftRcxRightRax(a)
			return true
		}
		if rtgTokTextIs(p, e.tok, "<<") {
			rtgAsmShiftLeftRcxByRax(a)
			return true
		}
		if rtgTokTextIs(p, e.tok, ">>") {
			rtgAsmShiftRightRcxByRax(a)
			return true
		}
		if rtgTokTextIs(p, e.tok, "==") {
			rtgAsmCmpRcxRaxSet(a, 0x94)
			return true
		}
		if rtgTokTextIs(p, e.tok, "!=") {
			rtgAsmCmpRcxRaxSet(a, 0x95)
			return true
		}
		if rtgTokTextIs(p, e.tok, "<") {
			rtgAsmCmpRcxRaxSet(a, 0x9c)
			return true
		}
		if rtgTokTextIs(p, e.tok, "<=") {
			rtgAsmCmpRcxRaxSet(a, 0x9e)
			return true
		}
		if rtgTokTextIs(p, e.tok, ">") {
			rtgAsmCmpRcxRaxSet(a, 0x9f)
			return true
		}
		if rtgTokTextIs(p, e.tok, ">=") {
			rtgAsmCmpRcxRaxSet(a, 0x9d)
			return true
		}
		return false
	}
	return false
}

func rtgEmitAppendAssignGeneral(g *rtgLinearGen, stmt rtgStmt, ep rtgExprParse) bool {
	if len(ep.exprs) == 0 {
		return false
	}
	root := ep.exprs[len(ep.exprs)-1]
	if root.kind != rtgExprCall || root.argCount != 2 || !rtgExprIsIdentText(g.prog, ep, root.left, "append") {
		return false
	}
	loc := rtgSliceLocationFromExpr(g, ep, ep.args[root.firstArg])
	if !loc.ok {
		return false
	}
	t := rtgResolveType(&g.meta, loc.typ)
	if t.kind != rtgTypeSlice {
		return false
	}
	elem := rtgResolveType(&g.meta, t.elem)
	valueIndex := ep.args[root.firstArg+1]
	if elem.kind == rtgTypeStruct {
		value := ep.exprs[valueIndex]
		if value.kind != rtgExprComposite {
			if value.kind == rtgExprIdent {
				typeTok := value.tok
				if !rtgTokTextIs(g.prog, typeTok+1, "{") {
					typeTok = rtgFindTokenByStart(g.prog, value.nameStart)
				}
				if rtgTokTextIs(g.prog, typeTok+1, "{") {
					return rtgEmitAppendStructCompositeTokens(g, loc, t.elem, typeTok)
				}
				return rtgEmitAppendStructLocal(g, ep, loc, t.elem, value)
			}
			return false
		}
		if !rtgEmitAppendStructComposite(g, ep, loc, t.elem, value) {
			return false
		}
		return true
	}
	if elem.kind == rtgTypeInt || elem.kind == rtgTypeByte {
		if !rtgEmitAppendScalarToLocation(g, ep, loc, elem.kind, valueIndex) {
			return false
		}
		return true
	}
	return false
}

func rtgEmitAppendScalarToLocation(g *rtgLinearGen, ep rtgExprParse, loc rtgSliceLocation, elemKind int, valueIndex int) bool {
	if !rtgEmitIntExpr(g, ep, valueIndex) {
		return false
	}
	rtgAsmPushRax(&g.asm)
	rtgAsmLoadRaxStack(&g.asm, loc.offset)
	rtgAsmPushRax(&g.asm)
	rtgAsmLoadRaxStack(&g.asm, loc.offset-8)
	rtgAsmMovRcxRax(&g.asm)
	rtgAsmPopRdx(&g.asm)
	rtgAsmPopRax(&g.asm)
	if elemKind == rtgTypeByte {
		rtgAsmStoreAlMemRdxRcx1(&g.asm)
	} else {
		rtgAsmStoreRaxMemRdxRcx8(&g.asm)
	}
	rtgAsmIncRcx(&g.asm)
	rtgAsmMovRaxRcx(&g.asm)
	rtgAsmStoreRaxStack(&g.asm, loc.offset-8)
	return true
}

func rtgSliceLocationFromExpr(g *rtgLinearGen, ep rtgExprParse, idx int) rtgSliceLocation {
	var loc rtgSliceLocation
	if idx < 0 || idx >= len(ep.exprs) {
		return loc
	}
	e := ep.exprs[idx]
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 || !rtgTypeIsSlice(&g.meta, g.locals[localIndex].typ) {
			return loc
		}
		loc.offset = g.locals[localIndex].offset
		loc.typ = g.locals[localIndex].typ
		loc.ok = true
		return loc
	}
	if e.kind == rtgExprSelector {
		base := ep.exprs[e.left]
		if base.kind != rtgExprIdent {
			return loc
		}
		localIndex := rtgFindLocalIndex(g, base.nameStart, base.nameEnd)
		if localIndex < 0 {
			return loc
		}
		fieldOffset := rtgStructFieldOffset(g, g.locals[localIndex].typ, e.nameStart, e.nameEnd)
		if fieldOffset < 0 {
			return loc
		}
		fieldType := rtgStructFieldType(g, g.locals[localIndex].typ, e.nameStart, e.nameEnd)
		if !rtgTypeIsSlice(&g.meta, fieldType) {
			return loc
		}
		loc.offset = g.locals[localIndex].offset - fieldOffset
		loc.typ = fieldType
		loc.ok = true
		return loc
	}
	return loc
}

func rtgEmitAppendStructCompositeTokens(g *rtgLinearGen, loc rtgSliceLocation, elemType int, typeTok int) bool {
	openTok := typeTok + 1
	closeTok := rtgSkipBalanced(g.prog, openTok, "{", "}")
	if closeTok <= openTok {
		return false
	}
	elemSize := rtgTypeSize(&g.meta, elemType)
	rtgAsmLoadRaxStack(&g.asm, loc.offset)
	rtgAsmPushRax(&g.asm)
	rtgAsmLoadRaxStack(&g.asm, loc.offset-8)
	rtgAsmMovRcxRax(&g.asm)
	rtgAsmImulRcxImm(&g.asm, elemSize)
	rtgAsmPopRdx(&g.asm)
	rtgAsmAddRdxRcx(&g.asm)
	rtgAsmPushRdx(&g.asm)
	i := openTok + 1
	for i < closeTok-1 {
		if !rtgTokIsKind(g.prog, i, rtgTokIdent) || !rtgTokTextIs(g.prog, i+1, ":") {
			return false
		}
		fieldTok := g.prog.toks[i]
		exprStart := i + 2
		exprEnd := rtgFindExprBoundary(g.prog, exprStart, closeTok-1)
		ep := rtgParseExpression(g.prog, exprStart, exprEnd)
		if !ep.ok || len(ep.exprs) == 0 {
			return false
		}
		fieldOffset := rtgStructFieldOffset(g, elemType, fieldTok.start, fieldTok.end)
		if fieldOffset < 0 {
			return false
		}
		if !rtgEmitIntExpr(g, ep, len(ep.exprs)-1) {
			return false
		}
		rtgAsmPopRdx(&g.asm)
		rtgAsmStoreRaxMemRdxDisp(&g.asm, fieldOffset)
		rtgAsmPushRdx(&g.asm)
		i = exprEnd
		if rtgTokTextIs(g.prog, i, ",") {
			i++
		}
	}
	rtgAsmPopRdx(&g.asm)
	rtgAsmLoadRaxStack(&g.asm, loc.offset-8)
	rtgAsmMovRcxRax(&g.asm)
	rtgAsmIncRcx(&g.asm)
	rtgAsmMovRaxRcx(&g.asm)
	rtgAsmStoreRaxStack(&g.asm, loc.offset-8)
	return true
}

func rtgEmitAppendStructLocal(g *rtgLinearGen, ep rtgExprParse, loc rtgSliceLocation, elemType int, value rtgExpr) bool {
	localIndex := rtgFindLocalIndex(g, value.nameStart, value.nameEnd)
	if localIndex < 0 {
		return false
	}
	elemSize := rtgTypeSize(&g.meta, elemType)
	if rtgTypeSize(&g.meta, g.locals[localIndex].typ) != elemSize {
		return false
	}
	rtgAsmLoadRaxStack(&g.asm, loc.offset)
	rtgAsmPushRax(&g.asm)
	rtgAsmLoadRaxStack(&g.asm, loc.offset-8)
	rtgAsmMovRcxRax(&g.asm)
	rtgAsmImulRcxImm(&g.asm, elemSize)
	rtgAsmPopRdx(&g.asm)
	rtgAsmAddRdxRcx(&g.asm)
	rtgAsmPushRdx(&g.asm)
	for at := 0; at < elemSize; at += 8 {
		rtgAsmLoadRaxStack(&g.asm, g.locals[localIndex].offset-at)
		rtgAsmPopRdx(&g.asm)
		rtgAsmStoreRaxMemRdxDisp(&g.asm, at)
		rtgAsmPushRdx(&g.asm)
	}
	rtgAsmPopRdx(&g.asm)
	rtgAsmLoadRaxStack(&g.asm, loc.offset-8)
	rtgAsmMovRcxRax(&g.asm)
	rtgAsmIncRcx(&g.asm)
	rtgAsmMovRaxRcx(&g.asm)
	rtgAsmStoreRaxStack(&g.asm, loc.offset-8)
	return true
}

func rtgEmitAppendStructComposite(g *rtgLinearGen, ep rtgExprParse, loc rtgSliceLocation, elemType int, value rtgExpr) bool {
	elemSize := rtgTypeSize(&g.meta, elemType)
	rtgAsmLoadRaxStack(&g.asm, loc.offset)
	rtgAsmPushRax(&g.asm)
	rtgAsmLoadRaxStack(&g.asm, loc.offset-8)
	rtgAsmMovRcxRax(&g.asm)
	rtgAsmImulRcxImm(&g.asm, elemSize)
	rtgAsmPopRdx(&g.asm)
	rtgAsmAddRdxRcx(&g.asm)
	rtgAsmPushRdx(&g.asm)
	for i := 0; i < value.argCount; i++ {
		field := ep.fields[value.firstArg+i]
		fieldOffset := rtgStructFieldOffset(g, elemType, field.nameStart, field.nameEnd)
		if fieldOffset < 0 {
			return false
		}
		if !rtgEmitIntExpr(g, ep, field.expr) {
			return false
		}
		rtgAsmPopRdx(&g.asm)
		rtgAsmStoreRaxMemRdxDisp(&g.asm, fieldOffset)
		rtgAsmPushRdx(&g.asm)
	}
	rtgAsmPopRdx(&g.asm)
	rtgAsmLoadRaxStack(&g.asm, loc.offset-8)
	rtgAsmMovRcxRax(&g.asm)
	rtgAsmIncRcx(&g.asm)
	rtgAsmMovRaxRcx(&g.asm)
	rtgAsmStoreRaxStack(&g.asm, loc.offset-8)
	return true
}

func rtgEmitAppendAssign(g *rtgLinearGen, stmt rtgStmt, ep rtgExprParse, offset int, globalOffset int) bool {
	if globalOffset >= 0 || len(ep.exprs) == 0 {
		return false
	}
	root := ep.exprs[len(ep.exprs)-1]
	if root.kind != rtgExprCall || root.argCount != 2 || !rtgExprIsIdentText(g.prog, ep, root.left, "append") {
		return false
	}
	first := ep.exprs[ep.args[root.firstArg]]
	if first.kind != rtgExprIdent {
		return false
	}
	localIndex := rtgFindLocalIndex(g, first.nameStart, first.nameEnd)
	if localIndex < 0 || g.locals[localIndex].offset != offset {
		return false
	}
	t := rtgResolveType(&g.meta, g.locals[localIndex].typ)
	if t.kind != rtgTypeSlice {
		return false
	}
	elem := rtgResolveType(&g.meta, t.elem)
	if elem.kind != rtgTypeInt && elem.kind != rtgTypeByte {
		return false
	}
	if !rtgEmitIntExpr(g, ep, ep.args[root.firstArg+1]) {
		return false
	}
	rtgAsmPushRax(&g.asm)
	rtgAsmLoadRaxStack(&g.asm, offset)
	rtgAsmPushRax(&g.asm)
	rtgAsmLoadRaxStack(&g.asm, offset-8)
	rtgAsmMovRcxRax(&g.asm)
	rtgAsmPopRdx(&g.asm)
	rtgAsmPopRax(&g.asm)
	if elem.kind == rtgTypeByte {
		rtgAsmStoreAlMemRdxRcx1(&g.asm)
	} else {
		rtgAsmStoreRaxMemRdxRcx8(&g.asm)
	}
	rtgAsmIncRcx(&g.asm)
	rtgAsmMovRaxRcx(&g.asm)
	rtgAsmStoreRaxStack(&g.asm, offset-8)
	return true
}

func rtgEmitBuiltinOpen(g *rtgLinearGen, ep rtgExprParse, e rtgExpr) bool {
	if e.argCount != 2 {
		return false
	}
	if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg+1]) {
		return false
	}
	rtgAsmMovRsiRax(&g.asm)
	if !rtgEmitStringPtrExpr(g, ep, ep.args[e.firstArg]) {
		return false
	}
	rtgAsmMovRdiRax(&g.asm)
	rtgAsmMovRdxImm(&g.asm, 438)
	rtgAsmMovRaxImm(&g.asm, 2)
	rtgAsmSyscall(&g.asm)
	return true
}

func rtgEmitBuiltinClose(g *rtgLinearGen, ep rtgExprParse, e rtgExpr) bool {
	if e.argCount != 1 {
		return false
	}
	if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg]) {
		return false
	}
	rtgAsmMovRdiRax(&g.asm)
	rtgAsmMovRaxImm(&g.asm, 3)
	rtgAsmSyscall(&g.asm)
	return true
}

func rtgEmitBuiltinChmod(g *rtgLinearGen, ep rtgExprParse, e rtgExpr) bool {
	if e.argCount != 2 {
		return false
	}
	if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg]) {
		return false
	}
	rtgAsmPushRax(&g.asm)
	if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg+1]) {
		return false
	}
	rtgAsmMovRsiRax(&g.asm)
	rtgAsmPopRdi(&g.asm)
	rtgAsmMovRaxImm(&g.asm, 91)
	rtgAsmSyscall(&g.asm)
	return true
}

func rtgEmitBuiltinRead(g *rtgLinearGen, ep rtgExprParse, e rtgExpr) bool {
	if e.argCount != 3 {
		return false
	}
	if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg]) {
		return false
	}
	rtgAsmPushRax(&g.asm)
	if !rtgEmitSlicePtrLen(g, ep, ep.args[e.firstArg+1]) {
		return false
	}
	rtgAsmMovRsiRax(&g.asm)
	rtgAsmMovRdxRcx(&g.asm)
	rtgAsmPopRdi(&g.asm)
	rtgAsmMovRaxImm(&g.asm, 0)
	rtgAsmSyscall(&g.asm)
	return true
}

func rtgEmitBuiltinWrite(g *rtgLinearGen, ep rtgExprParse, e rtgExpr) bool {
	if e.argCount != 3 {
		return false
	}
	offConst := rtgEvalConstExpr(g, ep, ep.args[e.firstArg+2])
	if !offConst.ok {
		return false
	}
	if offConst.value > 0 {
		return false
	}
	if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg]) {
		return false
	}
	rtgAsmPushRax(&g.asm)
	if !rtgEmitSlicePtrLen(g, ep, ep.args[e.firstArg+1]) {
		return false
	}
	rtgAsmMovRsiRax(&g.asm)
	rtgAsmMovRdxRcx(&g.asm)
	rtgAsmPopRdi(&g.asm)
	rtgAsmMovRaxImm(&g.asm, 1)
	rtgAsmSyscall(&g.asm)
	return true
}

func rtgEmitSlicePtrLen(g *rtgLinearGen, ep rtgExprParse, idx int) bool {
	if idx < 0 || idx >= len(ep.exprs) {
		return false
	}
	e := ep.exprs[idx]
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 {
			return false
		}
		if !rtgTypeIsSlice(&g.meta, g.locals[localIndex].typ) {
			return false
		}
		rtgAsmLoadRaxStack(&g.asm, g.locals[localIndex].offset)
		rtgAsmLoadRcxStack(&g.asm, g.locals[localIndex].offset-8)
		return true
	}
	if e.kind == rtgExprSelector {
		fieldType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsSlice(&g.meta, fieldType) {
			return false
		}
		if !rtgEmitSelectorAddressRdx(g, ep, e) {
			return false
		}
		rtgAsmLoadRaxMemRdxDisp(&g.asm, 0)
		rtgAsmLoadRcxMemRdxDisp(&g.asm, 8)
		return true
	}
	return false
}

func rtgEmitIndexedStructField(g *rtgLinearGen, ep rtgExprParse, indexExpr rtgExpr, fieldStart int, fieldEnd int) bool {
	leftType := rtgInferParsedExprType(g, ep, indexExpr.left)
	sliceType := rtgResolveType(&g.meta, leftType)
	if sliceType.kind != rtgTypeSlice {
		return false
	}
	elemType := rtgResolveType(&g.meta, sliceType.elem)
	if elemType.kind != rtgTypeStruct {
		return false
	}
	fieldOffset := rtgStructFieldOffset(g, sliceType.elem, fieldStart, fieldEnd)
	if fieldOffset < 0 {
		return false
	}
	if !rtgEmitIntExpr(g, ep, indexExpr.right) {
		return false
	}
	rtgAsmPushRax(&g.asm)
	if !rtgEmitSlicePtrLen(g, ep, indexExpr.left) {
		return false
	}
	rtgAsmPopRcx(&g.asm)
	rtgAsmImulRcxImm(&g.asm, rtgTypeSize(&g.meta, sliceType.elem))
	rtgAsmLoadQwordRaxIndexRcxDisp(&g.asm, fieldOffset)
	return true
}

func rtgEmitIndexExpr(g *rtgLinearGen, ep rtgExprParse, e rtgExpr) bool {
	left := ep.exprs[e.left]
	if left.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, left.nameStart, left.nameEnd)
		if localIndex < 0 {
			return false
		}
		t := rtgResolveType(&g.meta, g.locals[localIndex].typ)
		if t.kind == rtgTypeString {
			if !rtgEmitIntExpr(g, ep, e.right) {
				return false
			}
			rtgAsmPushRax(&g.asm)
			rtgAsmLoadRaxStack(&g.asm, g.locals[localIndex].offset)
			rtgAsmPopRcx(&g.asm)
			rtgAsmLoadByteRaxIndexRcx(&g.asm)
			return true
		}
		if t.kind == rtgTypeSlice {
			elem := rtgResolveType(&g.meta, t.elem)
			if elem.kind != rtgTypeInt && elem.kind != rtgTypeByte {
				return false
			}
			if !rtgEmitIntExpr(g, ep, e.right) {
				return false
			}
			rtgAsmPushRax(&g.asm)
			rtgAsmLoadRaxStack(&g.asm, g.locals[localIndex].offset)
			rtgAsmPopRcx(&g.asm)
			if elem.kind == rtgTypeByte {
				rtgAsmLoadByteRaxIndexRcx(&g.asm)
			} else {
				rtgAsmLoadQwordRaxIndexRcx8(&g.asm)
			}
			return true
		}
	}
	if left.kind == rtgExprSelector {
		fieldType := rtgInferParsedExprType(g, ep, e.left)
		t := rtgResolveType(&g.meta, fieldType)
		if !rtgEmitSelectorAddressRdx(g, ep, left) {
			return false
		}
		if t.kind == rtgTypeString {
			if !rtgEmitIntExpr(g, ep, e.right) {
				return false
			}
			rtgAsmPushRax(&g.asm)
			rtgAsmLoadRaxMemRdxDisp(&g.asm, 0)
			rtgAsmPopRcx(&g.asm)
			rtgAsmLoadByteRaxIndexRcx(&g.asm)
			return true
		}
		if t.kind == rtgTypeSlice {
			elem := rtgResolveType(&g.meta, t.elem)
			if elem.kind != rtgTypeInt && elem.kind != rtgTypeByte {
				return false
			}
			if !rtgEmitIntExpr(g, ep, e.right) {
				return false
			}
			rtgAsmPushRax(&g.asm)
			rtgAsmLoadRaxMemRdxDisp(&g.asm, 0)
			rtgAsmPopRcx(&g.asm)
			if elem.kind == rtgTypeByte {
				rtgAsmLoadByteRaxIndexRcx(&g.asm)
			} else {
				rtgAsmLoadQwordRaxIndexRcx8(&g.asm)
			}
			return true
		}
	}
	if left.kind == rtgExprIndex {
		if !rtgEmitIntExpr(g, ep, e.right) {
			return false
		}
		rtgAsmPushRax(&g.asm)
		if !rtgEmitStringPtrExpr(g, ep, e.left) {
			return false
		}
		rtgAsmPopRcx(&g.asm)
		rtgAsmLoadByteRaxIndexRcx(&g.asm)
		return true
	}
	return false
}

func rtgEmitStringPtrExpr(g *rtgLinearGen, ep rtgExprParse, idx int) bool {
	if idx < 0 || idx >= len(ep.exprs) {
		return false
	}
	e := ep.exprs[idx]
	if e.kind == rtgExprString {
		msg := rtgDecodeStringToken(g.prog.src, g.prog.toks[e.tok])
		msgOff := len(g.asm.data)
		for i := 0; i < len(msg); i++ {
			g.asm.data = append(g.asm.data, msg[i])
		}
		g.asm.data = append(g.asm.data, 0)
		rtgAsmMovRaxDataAddr(&g.asm, msgOff)
		return true
	}
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 || !rtgTypeIsString(&g.meta, g.locals[localIndex].typ) {
			return false
		}
		rtgAsmLoadRaxStack(&g.asm, g.locals[localIndex].offset)
		return true
	}
	if e.kind == rtgExprIndex {
		left := ep.exprs[e.left]
		if left.kind != rtgExprIdent {
			return false
		}
		localIndex := rtgFindLocalIndex(g, left.nameStart, left.nameEnd)
		if localIndex < 0 {
			return false
		}
		t := rtgResolveType(&g.meta, g.locals[localIndex].typ)
		if t.kind != rtgTypeSlice {
			return false
		}
		elem := rtgResolveType(&g.meta, t.elem)
		if elem.kind != rtgTypeString {
			return false
		}
		if !rtgEmitIntExpr(g, ep, e.right) {
			return false
		}
		rtgAsmPushRax(&g.asm)
		rtgAsmLoadRaxStack(&g.asm, g.locals[localIndex].offset)
		rtgAsmPopRcx(&g.asm)
		rtgAsmShlRcxImm(&g.asm, 4)
		rtgAsmLoadQwordRaxIndexRcx1(&g.asm)
		return true
	}
	return false
}

func rtgFindLocalOffset(g *rtgLinearGen, nameStart int, nameEnd int) int {
	for i := 0; i < len(g.locals); i++ {
		if rtgBytesEqualRange(g.prog.src, g.locals[i].nameStart, g.locals[i].nameEnd, nameStart, nameEnd) {
			return g.locals[i].offset
		}
	}
	return -1
}

func rtgFindLocalIndex(g *rtgLinearGen, nameStart int, nameEnd int) int {
	for i := 0; i < len(g.locals); i++ {
		if rtgBytesEqualRange(g.prog.src, g.locals[i].nameStart, g.locals[i].nameEnd, nameStart, nameEnd) {
			return i
		}
	}
	return -1
}

func rtgEmitSelectorAddressRdx(g *rtgLinearGen, ep rtgExprParse, e rtgExpr) bool {
	base := ep.exprs[e.left]
	baseType := rtgInferParsedExprType(g, ep, e.left)
	fieldOffset := rtgStructFieldOffset(g, baseType, e.nameStart, e.nameEnd)
	if fieldOffset < 0 {
		return false
	}
	if base.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, base.nameStart, base.nameEnd)
		if localIndex < 0 {
			return false
		}
		t := rtgResolveType(&g.meta, g.locals[localIndex].typ)
		if t.kind == rtgTypePointer {
			rtgAsmLoadRdxStack(&g.asm, g.locals[localIndex].offset)
			if fieldOffset != 0 {
				rtgAsmAddRdxImm(&g.asm, fieldOffset)
			}
			return true
		}
		rtgAsmLeaRdxStack(&g.asm, g.locals[localIndex].offset-fieldOffset)
		return true
	}
	if base.kind == rtgExprSelector {
		if !rtgEmitSelectorAddressRdx(g, ep, base) {
			return false
		}
		t := rtgResolveType(&g.meta, baseType)
		if t.kind == rtgTypePointer {
			rtgAsmLoadRdxMemRdxDisp(&g.asm, 0)
		}
		if fieldOffset != 0 {
			rtgAsmAddRdxImm(&g.asm, fieldOffset)
		}
		return true
	}
	return false
}

func rtgStructFieldOffset(g *rtgLinearGen, typ int, nameStart int, nameEnd int) int {
	if typ < 0 || typ >= len(g.meta.types) {
		return -1
	}
	t := g.meta.types[typ]
	if t.kind == rtgTypeNamed && t.elem > 0 {
		t = g.meta.types[t.elem]
	}
	if t.kind == rtgTypePointer && t.elem > 0 && t.elem < len(g.meta.types) {
		t = rtgResolveType(&g.meta, t.elem)
	}
	if t.kind != rtgTypeStruct {
		return -1
	}
	for i := 0; i < t.count; i++ {
		field := g.meta.fields[t.first+i]
		if rtgBytesEqualRange(g.prog.src, field.nameStart, field.nameEnd, nameStart, nameEnd) {
			return field.offset
		}
	}
	return -1
}

func rtgStructFieldType(g *rtgLinearGen, typ int, nameStart int, nameEnd int) int {
	if typ < 0 || typ >= len(g.meta.types) {
		return 0
	}
	t := g.meta.types[typ]
	if t.kind == rtgTypeNamed && t.elem > 0 {
		t = g.meta.types[t.elem]
	}
	if t.kind == rtgTypePointer && t.elem > 0 && t.elem < len(g.meta.types) {
		t = rtgResolveType(&g.meta, t.elem)
	}
	if t.kind != rtgTypeStruct {
		return 0
	}
	for i := 0; i < t.count; i++ {
		field := g.meta.fields[t.first+i]
		if rtgBytesEqualRange(g.prog.src, field.nameStart, field.nameEnd, nameStart, nameEnd) {
			return field.typ
		}
	}
	return 0
}

func rtgFindGlobalOffset(g *rtgLinearGen, nameStart int, nameEnd int) int {
	for i := 0; i < len(g.globals); i++ {
		if rtgBytesEqualRange(g.prog.src, g.globals[i].nameStart, g.globals[i].nameEnd, nameStart, nameEnd) {
			return g.globals[i].offset
		}
	}
	return -1
}

func rtgAddTypedLocal(g *rtgLinearGen, nameStart int, nameEnd int, typ int) int {
	size := rtgTypeSize(&g.meta, typ)
	if size < 8 {
		size = 8
	}
	g.stackUsed = rtgAlignTo8(g.stackUsed + size)
	offset := g.stackUsed
	g.locals = append(g.locals, rtgLocalInfo{nameStart: nameStart, nameEnd: nameEnd, offset: offset, typ: typ, size: size})
	return offset
}

func rtgZeroLocalAtOffset(g *rtgLinearGen, offset int) {
	size := 8
	typ := rtgTypeInt
	for i := 0; i < len(g.locals); i++ {
		if g.locals[i].offset == offset {
			size = g.locals[i].size
			typ = g.locals[i].typ
		}
	}
	t := rtgResolveType(&g.meta, typ)
	if t.kind == rtgTypeSlice {
		elemSize := rtgTypeSize(&g.meta, t.elem)
		if elemSize < 1 {
			elemSize = 8
		}
		backingSize := 1048576
		backingOff := g.asm.bssSize
		g.asm.bssSize += backingSize
		rtgAsmMovRaxBssAddr(&g.asm, backingOff)
		rtgAsmStoreRaxStack(&g.asm, offset)
		rtgAsmMovRaxImm(&g.asm, 0)
		rtgAsmStoreRaxStack(&g.asm, offset-8)
		rtgAsmMovRaxImm(&g.asm, backingSize/elemSize)
		rtgAsmStoreRaxStack(&g.asm, offset-16)
		return
	}
	rtgAsmMovRaxImm(&g.asm, 0)
	for at := 0; at < size; at += 8 {
		rtgAsmStoreRaxStack(&g.asm, offset-at)
	}
}

func rtgEvalConstByName(g *rtgLinearGen, nameStart int, nameEnd int) rtgConstResult {
	for i := 0; i < len(g.meta.globals); i++ {
		s := g.meta.globals[i]
		if s.kind == rtgTokConst && rtgBytesEqualRange(g.prog.src, s.nameStart, s.nameEnd, nameStart, nameEnd) {
			ep := rtgParseExpression(g.prog, s.initStart, s.initEnd)
			if !ep.ok || len(ep.exprs) == 0 {
				var r rtgConstResult
				return r
			}
			return rtgEvalConstExpr(g, ep, len(ep.exprs)-1)
		}
	}
	var r rtgConstResult
	return r
}

func rtgEvalConstExpr(g *rtgLinearGen, ep rtgExprParse, idx int) rtgConstResult {
	p := g.prog
	if idx < 0 || idx >= len(ep.exprs) {
		var r rtgConstResult
		return r
	}
	e := ep.exprs[idx]
	if e.kind == rtgExprInt {
		return rtgConstResult{value: rtgParseIntToken(p.src, p.toks[e.tok]), ok: true}
	}
	if e.kind == rtgExprChar {
		return rtgConstResult{value: rtgParseCharToken(p.src, p.toks[e.tok]), ok: true}
	}
	if e.kind == rtgExprBool {
		if rtgBytesEqualText(p.src, p.toks[e.tok].start, p.toks[e.tok].end, "true") {
			return rtgConstResult{value: 1, ok: true}
		}
		return rtgConstResult{value: 0, ok: true}
	}
	if e.kind == rtgExprIdent {
		return rtgEvalConstByName(g, e.nameStart, e.nameEnd)
	}
	if e.kind == rtgExprCall {
		if e.argCount == 1 && rtgExprIsIdentText(p, ep, e.left, "int") || e.argCount == 1 && rtgExprIsIdentText(p, ep, e.left, "byte") || e.argCount == 1 && rtgExprIsIdentText(p, ep, e.left, "int64") {
			return rtgEvalConstExpr(g, ep, ep.args[e.firstArg])
		}
		var r rtgConstResult
		return r
	}
	if e.kind == rtgExprUnary {
		inner := rtgEvalConstExpr(g, ep, e.left)
		if !inner.ok {
			var r rtgConstResult
			return r
		}
		if rtgTokTextIs(p, e.tok, "-") {
			return rtgConstResult{value: -inner.value, ok: true}
		}
		if rtgTokTextIs(p, e.tok, "+") {
			return rtgConstResult{value: inner.value, ok: true}
		}
		if rtgTokTextIs(p, e.tok, "!") {
			if inner.value == 0 {
				return rtgConstResult{value: 1, ok: true}
			}
			return rtgConstResult{value: 0, ok: true}
		}
		var r rtgConstResult
		return r
	}
	if e.kind == rtgExprBinary {
		left := rtgEvalConstExpr(g, ep, e.left)
		if !left.ok {
			var r rtgConstResult
			return r
		}
		if rtgTokTextIs(p, e.tok, "&&") {
			if left.value == 0 {
				return rtgConstResult{value: 0, ok: true}
			}
			right := rtgEvalConstExpr(g, ep, e.right)
			if !right.ok {
				var r rtgConstResult
				return r
			}
			if right.value != 0 {
				return rtgConstResult{value: 1, ok: true}
			}
			return rtgConstResult{value: 0, ok: true}
		}
		if rtgTokTextIs(p, e.tok, "||") {
			if left.value != 0 {
				return rtgConstResult{value: 1, ok: true}
			}
			right := rtgEvalConstExpr(g, ep, e.right)
			if !right.ok {
				var r rtgConstResult
				return r
			}
			if right.value != 0 {
				return rtgConstResult{value: 1, ok: true}
			}
			return rtgConstResult{value: 0, ok: true}
		}
		right := rtgEvalConstExpr(g, ep, e.right)
		if !right.ok {
			var r rtgConstResult
			return r
		}
		return rtgEvalConstBinary(g, e.tok, left.value, right.value)
	}
	var r rtgConstResult
	return r
}

func rtgEvalConstBinary(g *rtgLinearGen, tok int, left int, right int) rtgConstResult {
	p := g.prog
	if rtgTokTextIs(p, tok, "+") {
		return rtgConstResult{value: left + right, ok: true}
	}
	if rtgTokTextIs(p, tok, "-") {
		return rtgConstResult{value: left - right, ok: true}
	}
	if rtgTokTextIs(p, tok, "*") {
		return rtgConstResult{value: left * right, ok: true}
	}
	if rtgTokTextIs(p, tok, "/") {
		if right == 0 {
			var r rtgConstResult
			return r
		}
		return rtgConstResult{value: left / right, ok: true}
	}
	if rtgTokTextIs(p, tok, "%") {
		if right == 0 {
			var r rtgConstResult
			return r
		}
		return rtgConstResult{value: left % right, ok: true}
	}
	if rtgTokTextIs(p, tok, "&") {
		return rtgConstResult{value: left & right, ok: true}
	}
	if rtgTokTextIs(p, tok, "|") {
		return rtgConstResult{value: left | right, ok: true}
	}
	if rtgTokTextIs(p, tok, "^") {
		return rtgConstResult{value: left ^ right, ok: true}
	}
	if rtgTokTextIs(p, tok, "&^") {
		return rtgConstResult{value: left &^ right, ok: true}
	}
	if rtgTokTextIs(p, tok, "<<") {
		return rtgConstResult{value: left << right, ok: true}
	}
	if rtgTokTextIs(p, tok, ">>") {
		return rtgConstResult{value: left >> right, ok: true}
	}
	if rtgTokTextIs(p, tok, "==") {
		if left == right {
			return rtgConstResult{value: 1, ok: true}
		}
		return rtgConstResult{value: 0, ok: true}
	}
	if rtgTokTextIs(p, tok, "!=") {
		if left != right {
			return rtgConstResult{value: 1, ok: true}
		}
		return rtgConstResult{value: 0, ok: true}
	}
	if rtgTokTextIs(p, tok, "<") {
		if left < right {
			return rtgConstResult{value: 1, ok: true}
		}
		return rtgConstResult{value: 0, ok: true}
	}
	if rtgTokTextIs(p, tok, "<=") {
		if left <= right {
			return rtgConstResult{value: 1, ok: true}
		}
		return rtgConstResult{value: 0, ok: true}
	}
	if rtgTokTextIs(p, tok, ">") {
		if left > right {
			return rtgConstResult{value: 1, ok: true}
		}
		return rtgConstResult{value: 0, ok: true}
	}
	if rtgTokTextIs(p, tok, ">=") {
		if left >= right {
			return rtgConstResult{value: 1, ok: true}
		}
		return rtgConstResult{value: 0, ok: true}
	}
	var r rtgConstResult
	return r
}

func rtgExprIsIdentText(p rtgProgram, ep rtgExprParse, idx int, text string) bool {
	if idx < 0 || idx >= len(ep.exprs) {
		return false
	}
	e := ep.exprs[idx]
	if e.kind != rtgExprIdent {
		return false
	}
	if rtgTokTextIs(p, e.tok, text) {
		return true
	}
	return rtgBytesEqualText(p.src, e.nameStart, e.nameEnd, text)
}

func rtgFuncInfoFromCall(g *rtgLinearGen, ep rtgExprParse, idx int) int {
	if idx < 0 || idx >= len(ep.exprs) {
		return -1
	}
	e := ep.exprs[idx]
	if e.kind != rtgExprIdent {
		return -1
	}
	for i := 0; i < len(g.meta.funcs); i++ {
		f := g.meta.funcs[i]
		if rtgBytesEqualRange(g.prog.src, f.nameStart, f.nameEnd, e.nameStart, e.nameEnd) {
			return i
		}
	}
	return -1
}

func rtgAppendMovRaxImm32(out []byte, imm int) []byte {
	out = append(out, 0x48)
	out = append(out, 0xc7)
	out = append(out, 0xc0)
	return rtgAppend32(out, imm)
}

func rtgAppendMovRdiImm32(out []byte, imm int) []byte {
	out = append(out, 0x48)
	out = append(out, 0xc7)
	out = append(out, 0xc7)
	return rtgAppend32(out, imm)
}

func rtgAppendMovRdxImm32(out []byte, imm int) []byte {
	out = append(out, 0x48)
	out = append(out, 0xc7)
	out = append(out, 0xc2)
	return rtgAppend32(out, imm)
}

func rtgAsmInit(a *rtgAsm) {
	a.imageBase = 0x400000
	a.codeOffset = 64 + 56
}

func rtgAsmNewLabel(a *rtgAsm) int {
	a.labelPos = append(a.labelPos, 0)
	a.labelSet = append(a.labelSet, false)
	return len(a.labelPos) - 1
}

func rtgAsmMarkLabel(a *rtgAsm, label int) {
	if label >= 0 && label < len(a.labelPos) {
		a.labelPos[label] = len(a.code)
		a.labelSet[label] = true
	}
}

func rtgAsmEmit8(a *rtgAsm, v int) {
	a.code = append(a.code, byte(v))
}

func rtgAsmEmit32(a *rtgAsm, v int) {
	a.code = rtgAppend32(a.code, v)
}

func rtgAsmEmit64(a *rtgAsm, v int) {
	a.code = rtgAppend64(a.code, v)
}

func rtgAsmMovRaxImm(a *rtgAsm, imm int) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0xc7)
	rtgAsmEmit8(a, 0xc0)
	rtgAsmEmit32(a, imm)
}

func rtgAsmMovRaxImm64(a *rtgAsm, imm int) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0xb8)
	rtgAsmEmit64(a, imm)
}

func rtgAsmMovRaxDataAddr(a *rtgAsm, dataOff int) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0xb8)
	at := len(a.code)
	rtgAsmEmit64(a, 0)
	a.absRelocs = append(a.absRelocs, rtgAbsRef{at: at, off: dataOff, kind: 1})
}

func rtgAsmMovRaxBssAddr(a *rtgAsm, bssOff int) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0xb8)
	at := len(a.code)
	rtgAsmEmit64(a, 0)
	a.absRelocs = append(a.absRelocs, rtgAbsRef{at: at, off: bssOff, kind: 2})
}

func rtgAsmMovR10BssAddr(a *rtgAsm, bssOff int) {
	rtgAsmEmit8(a, 0x49)
	rtgAsmEmit8(a, 0xba)
	at := len(a.code)
	rtgAsmEmit64(a, 0)
	a.absRelocs = append(a.absRelocs, rtgAbsRef{at: at, off: bssOff, kind: 2})
}

func rtgAsmBuildArgvSlice(a *rtgAsm, bssOff int) {
	loopLabel := rtgAsmNewLabel(a)
	strlenLabel := rtgAsmNewLabel(a)
	afterLenLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x8b)
	rtgAsmEmit8(a, 0x04)
	rtgAsmEmit8(a, 0x24)
	rtgAsmEmit8(a, 0x49)
	rtgAsmEmit8(a, 0x89)
	rtgAsmEmit8(a, 0xc0)
	rtgAsmEmit8(a, 0x4c)
	rtgAsmEmit8(a, 0x8d)
	rtgAsmEmit8(a, 0x4c)
	rtgAsmEmit8(a, 0x24)
	rtgAsmEmit8(a, 0x08)
	rtgAsmMovR10BssAddr(a, bssOff)
	rtgAsmEmit8(a, 0x4d)
	rtgAsmEmit8(a, 0x89)
	rtgAsmEmit8(a, 0xd4)
	rtgAsmEmit8(a, 0x4d)
	rtgAsmEmit8(a, 0x31)
	rtgAsmEmit8(a, 0xdb)
	rtgAsmMarkLabel(a, loopLabel)
	rtgAsmEmit8(a, 0x4d)
	rtgAsmEmit8(a, 0x39)
	rtgAsmEmit8(a, 0xc3)
	rtgAsmJgeLabel(a, doneLabel)
	rtgAsmEmit8(a, 0x4b)
	rtgAsmEmit8(a, 0x8b)
	rtgAsmEmit8(a, 0x3c)
	rtgAsmEmit8(a, 0xd9)
	rtgAsmEmit8(a, 0x49)
	rtgAsmEmit8(a, 0x89)
	rtgAsmEmit8(a, 0x3a)
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x31)
	rtgAsmEmit8(a, 0xc0)
	rtgAsmMarkLabel(a, strlenLabel)
	rtgAsmEmit8(a, 0x80)
	rtgAsmEmit8(a, 0x3c)
	rtgAsmEmit8(a, 0x07)
	rtgAsmEmit8(a, 0x00)
	rtgAsmJzLabel(a, afterLenLabel)
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0xff)
	rtgAsmEmit8(a, 0xc0)
	rtgAsmJmpLabel(a, strlenLabel)
	rtgAsmMarkLabel(a, afterLenLabel)
	rtgAsmEmit8(a, 0x49)
	rtgAsmEmit8(a, 0x89)
	rtgAsmEmit8(a, 0x42)
	rtgAsmEmit8(a, 0x08)
	rtgAsmEmit8(a, 0x49)
	rtgAsmEmit8(a, 0x83)
	rtgAsmEmit8(a, 0xc2)
	rtgAsmEmit8(a, 0x10)
	rtgAsmEmit8(a, 0x49)
	rtgAsmEmit8(a, 0xff)
	rtgAsmEmit8(a, 0xc3)
	rtgAsmJmpLabel(a, loopLabel)
	rtgAsmMarkLabel(a, doneLabel)
	rtgAsmEmit8(a, 0x4c)
	rtgAsmEmit8(a, 0x89)
	rtgAsmEmit8(a, 0xe7)
	rtgAsmEmit8(a, 0x4c)
	rtgAsmEmit8(a, 0x89)
	rtgAsmEmit8(a, 0xc6)
	rtgAsmEmit8(a, 0x4c)
	rtgAsmEmit8(a, 0x89)
	rtgAsmEmit8(a, 0xc2)
}

func rtgAsmLoadRaxBss(a *rtgAsm, bssOff int) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0xa1)
	at := len(a.code)
	rtgAsmEmit64(a, 0)
	a.absRelocs = append(a.absRelocs, rtgAbsRef{at: at, off: bssOff, kind: 2})
}

func rtgAsmStoreRaxBss(a *rtgAsm, bssOff int) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0xa3)
	at := len(a.code)
	rtgAsmEmit64(a, 0)
	a.absRelocs = append(a.absRelocs, rtgAbsRef{at: at, off: bssOff, kind: 2})
}

func rtgAsmMovRdiImm(a *rtgAsm, imm int) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0xc7)
	rtgAsmEmit8(a, 0xc7)
	rtgAsmEmit32(a, imm)
}

func rtgAsmMovRdiRax(a *rtgAsm) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x89)
	rtgAsmEmit8(a, 0xc7)
}

func rtgAsmMovRdxRax(a *rtgAsm) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x89)
	rtgAsmEmit8(a, 0xc2)
}

func rtgAsmMovRaxRdx(a *rtgAsm) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x89)
	rtgAsmEmit8(a, 0xd0)
}

func rtgAsmMovRsiRax(a *rtgAsm) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x89)
	rtgAsmEmit8(a, 0xc6)
}

func rtgAsmMovRcxRax(a *rtgAsm) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x89)
	rtgAsmEmit8(a, 0xc1)
}

func rtgAsmMovRdxRcx(a *rtgAsm) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x89)
	rtgAsmEmit8(a, 0xca)
}

func rtgAsmAddRdxRcx(a *rtgAsm) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x01)
	rtgAsmEmit8(a, 0xca)
}

func rtgAsmMovRaxRcx(a *rtgAsm) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x89)
	rtgAsmEmit8(a, 0xc8)
}

func rtgAsmPushRbp(a *rtgAsm) {
	rtgAsmEmit8(a, 0x55)
}

func rtgAsmMovRbpRsp(a *rtgAsm) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x89)
	rtgAsmEmit8(a, 0xe5)
}

func rtgAsmSubRspImm(a *rtgAsm, imm int) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x81)
	rtgAsmEmit8(a, 0xec)
	rtgAsmEmit32(a, imm)
}

func rtgAsmAddRspImm(a *rtgAsm, imm int) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x81)
	rtgAsmEmit8(a, 0xc4)
	rtgAsmEmit32(a, imm)
}

func rtgAsmMovRdxImm(a *rtgAsm, imm int) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0xc7)
	rtgAsmEmit8(a, 0xc2)
	rtgAsmEmit32(a, imm)
}

func rtgAsmMovR10Imm(a *rtgAsm, imm int) {
	rtgAsmEmit8(a, 0x49)
	rtgAsmEmit8(a, 0xc7)
	rtgAsmEmit8(a, 0xc2)
	rtgAsmEmit32(a, imm)
}

func rtgAsmLeaRsiData(a *rtgAsm, dataOff int) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x8d)
	rtgAsmEmit8(a, 0x35)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	a.dataRelocs = append(a.dataRelocs, rtgDataRef{at: at, off: dataOff})
}

func rtgAsmLeaRdiData(a *rtgAsm, dataOff int) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x8d)
	rtgAsmEmit8(a, 0x3d)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	a.dataRelocs = append(a.dataRelocs, rtgDataRef{at: at, off: dataOff})
}

func rtgAsmSyscall(a *rtgAsm) {
	rtgAsmEmit8(a, 0x0f)
	rtgAsmEmit8(a, 0x05)
}

func rtgAsmPushRax(a *rtgAsm) {
	rtgAsmEmit8(a, 0x50)
}

func rtgAsmPushRcx(a *rtgAsm) {
	rtgAsmEmit8(a, 0x51)
}

func rtgAsmPushRdx(a *rtgAsm) {
	rtgAsmEmit8(a, 0x52)
}

func rtgAsmPopRax(a *rtgAsm) {
	rtgAsmEmit8(a, 0x58)
}

func rtgAsmPopRcx(a *rtgAsm) {
	rtgAsmEmit8(a, 0x59)
}

func rtgAsmPopRdx(a *rtgAsm) {
	rtgAsmEmit8(a, 0x5a)
}

func rtgAsmPopRsi(a *rtgAsm) {
	rtgAsmEmit8(a, 0x5e)
}

func rtgAsmPopRdi(a *rtgAsm) {
	rtgAsmEmit8(a, 0x5f)
}

func rtgAsmPopR8(a *rtgAsm) {
	rtgAsmEmit8(a, 0x41)
	rtgAsmEmit8(a, 0x58)
}

func rtgAsmPopR9(a *rtgAsm) {
	rtgAsmEmit8(a, 0x41)
	rtgAsmEmit8(a, 0x59)
}

func rtgAsmStoreRaxStack(a *rtgAsm, offset int) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x89)
	rtgAsmEmit8(a, 0x85)
	rtgAsmEmit32(a, -offset)
}

func rtgAsmStoreRdiStack(a *rtgAsm, offset int) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x89)
	rtgAsmEmit8(a, 0xbd)
	rtgAsmEmit32(a, -offset)
}

func rtgAsmStoreRsiStack(a *rtgAsm, offset int) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x89)
	rtgAsmEmit8(a, 0xb5)
	rtgAsmEmit32(a, -offset)
}

func rtgAsmStoreRdxStack(a *rtgAsm, offset int) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x89)
	rtgAsmEmit8(a, 0x95)
	rtgAsmEmit32(a, -offset)
}

func rtgAsmStoreRcxStack(a *rtgAsm, offset int) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x89)
	rtgAsmEmit8(a, 0x8d)
	rtgAsmEmit32(a, -offset)
}

func rtgAsmStoreR8Stack(a *rtgAsm, offset int) {
	rtgAsmEmit8(a, 0x4c)
	rtgAsmEmit8(a, 0x89)
	rtgAsmEmit8(a, 0x85)
	rtgAsmEmit32(a, -offset)
}

func rtgAsmStoreR9Stack(a *rtgAsm, offset int) {
	rtgAsmEmit8(a, 0x4c)
	rtgAsmEmit8(a, 0x89)
	rtgAsmEmit8(a, 0x8d)
	rtgAsmEmit32(a, -offset)
}

func rtgAsmLoadRaxStack(a *rtgAsm, offset int) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x8b)
	rtgAsmEmit8(a, 0x85)
	rtgAsmEmit32(a, -offset)
}

func rtgAsmLoadRaxFrameArg(a *rtgAsm, offset int) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x8b)
	rtgAsmEmit8(a, 0x85)
	rtgAsmEmit32(a, offset)
}

func rtgAsmLeaRaxStack(a *rtgAsm, offset int) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x8d)
	rtgAsmEmit8(a, 0x85)
	rtgAsmEmit32(a, -offset)
}

func rtgAsmLeaRdxStack(a *rtgAsm, offset int) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x8d)
	rtgAsmEmit8(a, 0x95)
	rtgAsmEmit32(a, -offset)
}

func rtgAsmLoadRdxStack(a *rtgAsm, offset int) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x8b)
	rtgAsmEmit8(a, 0x95)
	rtgAsmEmit32(a, -offset)
}

func rtgAsmAddRdxImm(a *rtgAsm, imm int) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x81)
	rtgAsmEmit8(a, 0xc2)
	rtgAsmEmit32(a, imm)
}

func rtgAsmLoadRdxMemRdxDisp(a *rtgAsm, disp int) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x8b)
	rtgAsmEmit8(a, 0x92)
	rtgAsmEmit32(a, disp)
}

func rtgAsmLoadRcxStack(a *rtgAsm, offset int) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x8b)
	rtgAsmEmit8(a, 0x8d)
	rtgAsmEmit32(a, -offset)
}

func rtgAsmLoadRcxMemRdxDisp(a *rtgAsm, disp int) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x8b)
	rtgAsmEmit8(a, 0x8a)
	rtgAsmEmit32(a, disp)
}

func rtgAsmLoadQwordRaxIndexRcx8(a *rtgAsm) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x8b)
	rtgAsmEmit8(a, 0x04)
	rtgAsmEmit8(a, 0xc8)
}

func rtgAsmLoadQwordRaxIndexRcx1(a *rtgAsm) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x8b)
	rtgAsmEmit8(a, 0x04)
	rtgAsmEmit8(a, 0x08)
}

func rtgAsmLoadQwordRaxIndexRcxDisp(a *rtgAsm, disp int) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x8b)
	rtgAsmEmit8(a, 0x84)
	rtgAsmEmit8(a, 0x08)
	rtgAsmEmit32(a, disp)
}

func rtgAsmLoadRaxMemRdxDisp(a *rtgAsm, disp int) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x8b)
	rtgAsmEmit8(a, 0x82)
	rtgAsmEmit32(a, disp)
}

func rtgAsmLoadByteRaxIndexRcx(a *rtgAsm) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x0f)
	rtgAsmEmit8(a, 0xb6)
	rtgAsmEmit8(a, 0x04)
	rtgAsmEmit8(a, 0x08)
}

func rtgAsmStoreRaxMemRdxRcx8(a *rtgAsm) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x89)
	rtgAsmEmit8(a, 0x04)
	rtgAsmEmit8(a, 0xca)
}

func rtgAsmStoreRaxMemRdxDisp(a *rtgAsm, disp int) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x89)
	rtgAsmEmit8(a, 0x82)
	rtgAsmEmit32(a, disp)
}

func rtgAsmStoreAlMemRdxRcx1(a *rtgAsm) {
	rtgAsmEmit8(a, 0x88)
	rtgAsmEmit8(a, 0x04)
	rtgAsmEmit8(a, 0x0a)
}

func rtgAsmIncRcx(a *rtgAsm) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0xff)
	rtgAsmEmit8(a, 0xc1)
}

func rtgAsmNegRax(a *rtgAsm) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0xf7)
	rtgAsmEmit8(a, 0xd8)
}

func rtgAsmBoolNotRax(a *rtgAsm) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x83)
	rtgAsmEmit8(a, 0xf8)
	rtgAsmEmit8(a, 0)
	rtgAsmEmit8(a, 0x0f)
	rtgAsmEmit8(a, 0x94)
	rtgAsmEmit8(a, 0xc0)
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x0f)
	rtgAsmEmit8(a, 0xb6)
	rtgAsmEmit8(a, 0xc0)
}

func rtgAsmCmpRaxImm8(a *rtgAsm, imm int) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x83)
	rtgAsmEmit8(a, 0xf8)
	rtgAsmEmit8(a, imm)
}

func rtgAsmAddRaxRcx(a *rtgAsm) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x01)
	rtgAsmEmit8(a, 0xc8)
}

func rtgAsmAndRaxRcx(a *rtgAsm) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x21)
	rtgAsmEmit8(a, 0xc8)
}

func rtgAsmAndRaxImm(a *rtgAsm, imm int) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x25)
	rtgAsmEmit32(a, imm)
}

func rtgAsmOrRaxRcx(a *rtgAsm) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x09)
	rtgAsmEmit8(a, 0xc8)
}

func rtgAsmXorRaxRcx(a *rtgAsm) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x31)
	rtgAsmEmit8(a, 0xc8)
}

func rtgAsmAndNotLeftRcxRightRax(a *rtgAsm) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0xf7)
	rtgAsmEmit8(a, 0xd0)
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x21)
	rtgAsmEmit8(a, 0xc8)
}

func rtgAsmShiftLeftRcxByRax(a *rtgAsm) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x89)
	rtgAsmEmit8(a, 0xca)
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x89)
	rtgAsmEmit8(a, 0xc1)
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x89)
	rtgAsmEmit8(a, 0xd0)
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0xd3)
	rtgAsmEmit8(a, 0xe0)
}

func rtgAsmShiftRightRcxByRax(a *rtgAsm) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x89)
	rtgAsmEmit8(a, 0xca)
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x89)
	rtgAsmEmit8(a, 0xc1)
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x89)
	rtgAsmEmit8(a, 0xd0)
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0xd3)
	rtgAsmEmit8(a, 0xf8)
}

func rtgAsmShlRcxImm(a *rtgAsm, imm int) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0xc1)
	rtgAsmEmit8(a, 0xe1)
	rtgAsmEmit8(a, imm)
}

func rtgAsmImulRcxImm(a *rtgAsm, imm int) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x69)
	rtgAsmEmit8(a, 0xc9)
	rtgAsmEmit32(a, imm)
}

func rtgAsmSubLeftRcxRightRax(a *rtgAsm) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x29)
	rtgAsmEmit8(a, 0xc1)
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x89)
	rtgAsmEmit8(a, 0xc8)
}

func rtgAsmMulRaxRcx(a *rtgAsm) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x0f)
	rtgAsmEmit8(a, 0xaf)
	rtgAsmEmit8(a, 0xc1)
}

func rtgAsmDivLeftRcxRightRax(a *rtgAsm, mod bool) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x89)
	rtgAsmEmit8(a, 0xc3)
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x89)
	rtgAsmEmit8(a, 0xc8)
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x99)
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0xf7)
	rtgAsmEmit8(a, 0xfb)
	if mod {
		rtgAsmEmit8(a, 0x48)
		rtgAsmEmit8(a, 0x89)
		rtgAsmEmit8(a, 0xd0)
	}
}

func rtgAsmCmpRcxRaxSet(a *rtgAsm, setcc int) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x39)
	rtgAsmEmit8(a, 0xc1)
	rtgAsmEmit8(a, 0x0f)
	rtgAsmEmit8(a, setcc)
	rtgAsmEmit8(a, 0xc0)
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x0f)
	rtgAsmEmit8(a, 0xb6)
	rtgAsmEmit8(a, 0xc0)
}

func rtgAsmRet(a *rtgAsm) {
	rtgAsmEmit8(a, 0xc3)
}

func rtgAsmLeave(a *rtgAsm) {
	rtgAsmEmit8(a, 0xc9)
}

func rtgAsmCallLabel(a *rtgAsm, label int) {
	rtgAsmEmit8(a, 0xe8)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	a.relocs = append(a.relocs, rtgLabelRef{at: at, label: label, kind: rtgRel32})
}

func rtgAsmJmpLabel(a *rtgAsm, label int) {
	rtgAsmEmit8(a, 0xe9)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	a.relocs = append(a.relocs, rtgLabelRef{at: at, label: label, kind: rtgRel32})
}

func rtgAsmJzLabel(a *rtgAsm, label int) {
	rtgAsmEmit8(a, 0x0f)
	rtgAsmEmit8(a, 0x84)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	a.relocs = append(a.relocs, rtgLabelRef{at: at, label: label, kind: rtgRel32})
}

func rtgAsmJnzLabel(a *rtgAsm, label int) {
	rtgAsmEmit8(a, 0x0f)
	rtgAsmEmit8(a, 0x85)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	a.relocs = append(a.relocs, rtgLabelRef{at: at, label: label, kind: rtgRel32})
}

func rtgAsmJgeLabel(a *rtgAsm, label int) {
	rtgAsmEmit8(a, 0x0f)
	rtgAsmEmit8(a, 0x8d)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	a.relocs = append(a.relocs, rtgLabelRef{at: at, label: label, kind: rtgRel32})
}

func rtgAsmPatch(a *rtgAsm) {
	for i := 0; i < len(a.relocs); i++ {
		r := a.relocs[i]
		if r.label >= 0 && r.label < len(a.labelPos) && a.labelSet[r.label] {
			target := a.labelPos[r.label]
			disp := target - (r.at + 4)
			rtgPut32At(a.code, r.at, disp)
		}
	}
	a.dataOffset = a.codeOffset + len(a.code)
	for i := 0; i < len(a.dataRelocs); i++ {
		r := a.dataRelocs[i]
		target := a.dataOffset + r.off
		next := a.codeOffset + r.at + 4
		disp := target - next
		rtgPut32At(a.code, r.at, disp)
	}
	for i := 0; i < len(a.absRelocs); i++ {
		r := a.absRelocs[i]
		target := a.dataOffset + r.off
		if r.kind == 2 {
			target = a.dataOffset + len(a.data) + r.off
		}
		rtgPut64At(a.code, r.at, a.imageBase+target)
	}
}

func rtgAsmImage(a *rtgAsm) []byte {
	rtgAsmPatch(a)
	var out []byte
	fileSize := a.codeOffset + len(a.code) + len(a.data)
	memSize := fileSize + a.bssSize
	out = rtgAppendElfHeader(out, a.codeOffset, fileSize, memSize)
	for i := 0; i < len(a.code); i++ {
		out = append(out, a.code[i])
	}
	for i := 0; i < len(a.data); i++ {
		out = append(out, a.data[i])
	}
	return out
}

func rtgAppendElfHeader(out []byte, entryOff int, fileSize int, memSize int) []byte {
	base := 0x400000

	out = append(out, 0x7f)
	out = append(out, 'E')
	out = append(out, 'L')
	out = append(out, 'F')
	out = append(out, 2)
	out = append(out, 1)
	out = append(out, 1)
	out = append(out, 0)
	for i := 0; i < 8; i++ {
		out = append(out, 0)
	}
	out = rtgAppend16(out, 2)
	out = rtgAppend16(out, 0x3e)
	out = rtgAppend32(out, 1)
	out = rtgAppend64(out, base+entryOff)
	out = rtgAppend64(out, 64)
	out = rtgAppend64(out, 0)
	out = rtgAppend32(out, 0)
	out = rtgAppend16(out, 64)
	out = rtgAppend16(out, 56)
	out = rtgAppend16(out, 1)
	out = rtgAppend16(out, 0)
	out = rtgAppend16(out, 0)
	out = rtgAppend16(out, 0)

	out = rtgAppend32(out, 1)
	out = rtgAppend32(out, 7)
	out = rtgAppend64(out, 0)
	out = rtgAppend64(out, base)
	out = rtgAppend64(out, base)
	out = rtgAppend64(out, fileSize)
	out = rtgAppend64(out, memSize)
	out = rtgAppend64(out, 0x1000)
	return out
}

func rtgAppend16(out []byte, v int) []byte {
	out = append(out, byte(v))
	out = append(out, byte(v>>8))
	return out
}

func rtgAppend32(out []byte, v int) []byte {
	out = append(out, byte(v))
	out = append(out, byte(v>>8))
	out = append(out, byte(v>>16))
	out = append(out, byte(v>>24))
	return out
}

func rtgAppend64(out []byte, v int) []byte {
	out = append(out, byte(v))
	out = append(out, byte(v>>8))
	out = append(out, byte(v>>16))
	out = append(out, byte(v>>24))
	out = append(out, byte(v>>32))
	out = append(out, byte(v>>40))
	out = append(out, byte(v>>48))
	out = append(out, byte(v>>56))
	return out
}

func rtgPut32At(out []byte, at int, v int) {
	out[at] = byte(v)
	out[at+1] = byte(v >> 8)
	out[at+2] = byte(v >> 16)
	out[at+3] = byte(v >> 24)
}

func rtgPut64At(out []byte, at int, v int) {
	out[at] = byte(v)
	out[at+1] = byte(v >> 8)
	out[at+2] = byte(v >> 16)
	out[at+3] = byte(v >> 24)
	out[at+4] = byte(v >> 32)
	out[at+5] = byte(v >> 40)
	out[at+6] = byte(v >> 48)
	out[at+7] = byte(v >> 56)
}

func rtgStringBytes(s string) []byte {
	var out []byte
	for i := 0; i < len(s); i++ {
		out = append(out, s[i])
	}
	return out
}

func rtgParseExpression(p rtgProgram, start int, end int) rtgExprParse {
	var ep rtgExprParse
	ep.prog = p
	ep.pos = start
	ep.end = end
	ep.ok = true
	rtgParseBinaryExpr(&ep, 1)
	if ep.pos < ep.end {
		ep.ok = false
	}
	return ep
}

func rtgParseBinaryExpr(ep *rtgExprParse, minPrec int) int {
	left := rtgParseUnaryExpr(ep)
	for ep.ok && ep.pos < ep.end {
		prec := rtgTokenPrecedence(ep.prog, ep.pos)
		if prec < minPrec {
			break
		}
		opTok := ep.pos
		ep.pos++
		right := rtgParseBinaryExpr(ep, prec+1)
		left = rtgAddExpr(ep, rtgExpr{kind: rtgExprBinary, tok: opTok, left: left, right: right})
	}
	return left
}

func rtgParseUnaryExpr(ep *rtgExprParse) int {
	if ep.pos >= ep.end {
		ep.ok = false
		return 0
	}
	if rtgTokTextIs(ep.prog, ep.pos, "+") || rtgTokTextIs(ep.prog, ep.pos, "-") || rtgTokTextIs(ep.prog, ep.pos, "!") || rtgTokTextIs(ep.prog, ep.pos, "&") || rtgTokTextIs(ep.prog, ep.pos, "*") {
		opTok := ep.pos
		ep.pos++
		inner := rtgParseUnaryExpr(ep)
		return rtgAddExpr(ep, rtgExpr{kind: rtgExprUnary, tok: opTok, left: inner})
	}
	return rtgParsePostfixExpr(ep)
}

func rtgParsePostfixExpr(ep *rtgExprParse) int {
	left := rtgParsePrimaryExpr(ep)
	for ep.ok && ep.pos < ep.end {
		if rtgTokTextIs(ep.prog, ep.pos, "{") {
			base := ep.exprs[left]
			if base.kind != rtgExprIdent {
				ep.ok = false
				return left
			}
			first := len(ep.fields)
			count := 0
			ep.pos++
			for ep.ok && ep.pos < ep.end && !rtgTokTextIs(ep.prog, ep.pos, "}") {
				if !rtgTokIsKind(ep.prog, ep.pos, rtgTokIdent) || !rtgTokTextIs(ep.prog, ep.pos+1, ":") {
					ep.ok = false
					return left
				}
				nameTok := ep.prog.toks[ep.pos]
				ep.pos += 2
				fieldEnd := rtgFindExprBoundary(ep.prog, ep.pos, ep.end)
				oldEnd := ep.end
				ep.end = fieldEnd
				fieldRoot := rtgParseBinaryExpr(ep, 1)
				ep.end = oldEnd
				ep.fields = append(ep.fields, rtgCompositeField{nameStart: nameTok.start, nameEnd: nameTok.end, expr: fieldRoot})
				count++
				ep.pos = fieldEnd
				if rtgTokTextIs(ep.prog, ep.pos, ",") {
					ep.pos++
				}
			}
			if !rtgTokTextIs(ep.prog, ep.pos, "}") {
				ep.ok = false
				return left
			}
			ep.pos++
			left = rtgAddExpr(ep, rtgExpr{kind: rtgExprComposite, tok: base.tok, firstArg: first, argCount: count, nameStart: base.nameStart, nameEnd: base.nameEnd})
			continue
		}
		if rtgTokTextIs(ep.prog, ep.pos, "(") {
			callTok := ep.pos
			var callArgs []int
			ep.pos++
			for ep.ok && ep.pos < ep.end && !rtgTokTextIs(ep.prog, ep.pos, ")") {
				argEnd := rtgFindExprBoundary(ep.prog, ep.pos, ep.end)
				if rtgTokTextIs(ep.prog, argEnd, "{") {
					closeTok := rtgSkipBalanced(ep.prog, argEnd, "{", "}")
					if closeTok > argEnd {
						argEnd = closeTok
					}
				}
				oldEnd := ep.end
				ep.end = argEnd
				argRoot := rtgParseBinaryExpr(ep, 1)
				ep.end = oldEnd
				callArgs = append(callArgs, argRoot)
				ep.pos = argEnd
				if rtgTokTextIs(ep.prog, ep.pos, ",") {
					ep.pos++
				}
			}
			if !rtgTokTextIs(ep.prog, ep.pos, ")") {
				ep.ok = false
				return left
			}
			ep.pos++
			first := len(ep.args)
			for i := 0; i < len(callArgs); i++ {
				ep.args = append(ep.args, callArgs[i])
			}
			count := len(callArgs)
			left = rtgAddExpr(ep, rtgExpr{kind: rtgExprCall, tok: callTok, left: left, firstArg: first, argCount: count})
			continue
		}
		if rtgTokTextIs(ep.prog, ep.pos, "[") {
			indexTok := ep.pos
			ep.pos++
			indexEnd := rtgFindMatchingExprClose(ep.prog, ep.pos, ep.end, "[", "]")
			if indexEnd <= ep.pos {
				ep.ok = false
				return left
			}
			oldEnd := ep.end
			ep.end = indexEnd
			right := rtgParseBinaryExpr(ep, 1)
			ep.end = oldEnd
			ep.pos = indexEnd + 1
			left = rtgAddExpr(ep, rtgExpr{kind: rtgExprIndex, tok: indexTok, left: left, right: right})
			continue
		}
		if rtgTokTextIs(ep.prog, ep.pos, ".") && rtgTokIsKind(ep.prog, ep.pos+1, rtgTokIdent) {
			dotTok := ep.pos
			nameTok := ep.prog.toks[ep.pos+1]
			ep.pos += 2
			left = rtgAddExpr(ep, rtgExpr{kind: rtgExprSelector, tok: dotTok, left: left, nameStart: nameTok.start, nameEnd: nameTok.end})
			continue
		}
		break
	}
	return left
}

func rtgParsePrimaryExpr(ep *rtgExprParse) int {
	if ep.pos >= ep.end {
		ep.ok = false
		return 0
	}
	tok := ep.prog.toks[ep.pos]
	if tok.kind == rtgTokIdent {
		ep.pos++
		if rtgBytesEqualText(ep.prog.src, tok.start, tok.end, "true") {
			return rtgAddExpr(ep, rtgExpr{kind: rtgExprBool, tok: ep.pos - 1})
		}
		if rtgBytesEqualText(ep.prog.src, tok.start, tok.end, "false") {
			return rtgAddExpr(ep, rtgExpr{kind: rtgExprBool, tok: ep.pos - 1})
		}
		return rtgAddExpr(ep, rtgExpr{kind: rtgExprIdent, tok: ep.pos - 1, nameStart: tok.start, nameEnd: tok.end})
	}
	if tok.kind == rtgTokNumber {
		ep.pos++
		return rtgAddExpr(ep, rtgExpr{kind: rtgExprInt, tok: ep.pos - 1})
	}
	if tok.kind == rtgTokFloat {
		ep.pos++
		return rtgAddExpr(ep, rtgExpr{kind: rtgExprFloat, tok: ep.pos - 1})
	}
	if tok.kind == rtgTokString {
		ep.pos++
		return rtgAddExpr(ep, rtgExpr{kind: rtgExprString, tok: ep.pos - 1})
	}
	if tok.kind == rtgTokChar {
		ep.pos++
		return rtgAddExpr(ep, rtgExpr{kind: rtgExprChar, tok: ep.pos - 1})
	}
	if rtgTokTextIs(ep.prog, ep.pos, "(") {
		ep.pos++
		inner := rtgParseBinaryExpr(ep, 1)
		if !rtgTokTextIs(ep.prog, ep.pos, ")") {
			ep.ok = false
			return inner
		}
		ep.pos++
		return inner
	}
	ep.ok = false
	return 0
}

func rtgAddExpr(ep *rtgExprParse, e rtgExpr) int {
	ep.exprs = append(ep.exprs, e)
	return len(ep.exprs) - 1
}

func rtgTokenPrecedence(p rtgProgram, pos int) int {
	if pos < 0 || pos >= len(p.toks) {
		return 0
	}
	if rtgTokTextIs(p, pos, "||") {
		return 1
	}
	if rtgTokTextIs(p, pos, "&&") {
		return 2
	}
	if rtgTokTextIs(p, pos, "==") || rtgTokTextIs(p, pos, "!=") || rtgTokTextIs(p, pos, "<") || rtgTokTextIs(p, pos, "<=") || rtgTokTextIs(p, pos, ">") || rtgTokTextIs(p, pos, ">=") {
		return 3
	}
	if rtgTokTextIs(p, pos, "+") || rtgTokTextIs(p, pos, "-") || rtgTokTextIs(p, pos, "|") || rtgTokTextIs(p, pos, "^") {
		return 4
	}
	if rtgTokTextIs(p, pos, "*") || rtgTokTextIs(p, pos, "/") || rtgTokTextIs(p, pos, "%") || rtgTokTextIs(p, pos, "<<") || rtgTokTextIs(p, pos, ">>") || rtgTokTextIs(p, pos, "&") || rtgTokTextIs(p, pos, "&^") {
		return 5
	}
	return 0
}

func rtgFindExprBoundary(p rtgProgram, start int, end int) int {
	i := start
	paren := 0
	brack := 0
	brace := 0
	for i < end {
		if paren == 0 && brack == 0 && brace == 0 && rtgTokTextIs(p, i, "{") {
			closeTok := rtgSkipBalanced(p, i, "{", "}")
			if closeTok > i {
				i = closeTok
				continue
			}
		}
		if paren == 0 && brack == 0 && brace == 0 && (rtgTokTextIs(p, i, ",") || rtgTokTextIs(p, i, ")") || rtgTokTextIs(p, i, "]") || rtgTokTextIs(p, i, "}")) {
			return i
		}
		if rtgTokTextIs(p, i, "(") {
			paren++
		} else if rtgTokTextIs(p, i, ")") {
			if paren == 0 {
				return i
			}
			paren--
		} else if rtgTokTextIs(p, i, "[") {
			brack++
		} else if rtgTokTextIs(p, i, "]") {
			if brack == 0 {
				return i
			}
			brack--
		} else if rtgTokTextIs(p, i, "{") {
			brace++
		} else if rtgTokTextIs(p, i, "}") {
			if brace == 0 {
				return i
			}
			brace--
		}
		i++
	}
	return i
}

func rtgFindMatchingExprClose(p rtgProgram, start int, end int, open string, close string) int {
	depth := 0
	i := start
	for i < end {
		if rtgTokTextIs(p, i, open) {
			depth++
		} else if rtgTokTextIs(p, i, close) {
			if depth == 0 {
				return i
			}
			depth--
		}
		i++
	}
	return start
}

func rtgParseFunctionBody(p rtgProgram, fn rtgFuncDecl) rtgBodyParse {
	var bp rtgBodyParse
	bp.prog = p
	bp.ok = true
	rtgParseStatements(&bp, fn.bodyStart+1, fn.bodyEnd)
	return bp
}

func rtgParseStatements(bp *rtgBodyParse, start int, end int) int {
	i := start
	for bp.ok && i < end {
		if rtgTokTextIs(bp.prog, i, "}") {
			return i
		}
		next := rtgParseOneStatement(bp, i, end)
		if next <= i {
			bp.ok = false
			return i
		}
		i = next
	}
	return i
}

func rtgParseOneStatement(bp *rtgBodyParse, start int, end int) int {
	p := bp.prog
	if start >= end {
		return end
	}
	if rtgTokIsKeyword(p, start, rtgTokReturn) {
		exprEnd := rtgStatementLineEnd(p, start+1, end)
		bp.stmts = append(bp.stmts, rtgStmt{kind: rtgStmtReturn, startTok: start, endTok: exprEnd, exprStart: start + 1, exprEnd: exprEnd})
		return exprEnd
	}
	if rtgTokIsKeyword(p, start, rtgTokIf) {
		bodyStart := rtgFindNextTokenText(p, start+1, end, "{")
		if bodyStart <= start {
			return start
		}
		bodyEnd := rtgFindMatchingBrace(p, bodyStart, end)
		if bodyEnd <= bodyStart {
			return start
		}
		stmt := rtgStmt{kind: rtgStmtIf, startTok: start, endTok: bodyEnd + 1, exprStart: start + 1, exprEnd: bodyStart, bodyStart: bodyStart + 1, bodyEnd: bodyEnd}
		next := bodyEnd + 1
		if rtgTokIsKeyword(p, next, rtgTokElse) {
			if rtgTokIsKeyword(p, next+1, rtgTokIf) {
				elseEnd := rtgFindIfStatementEnd(p, next+1, end)
				if elseEnd <= next+1 {
					return start
				}
				stmt.elseStart = next + 1
				stmt.elseEnd = elseEnd
				stmt.endTok = elseEnd
				next = elseEnd
			} else if rtgTokTextIs(p, next+1, "{") {
				elseBodyEnd := rtgFindMatchingBrace(p, next+1, end)
				if elseBodyEnd <= next+1 {
					return start
				}
				stmt.elseStart = next + 2
				stmt.elseEnd = elseBodyEnd
				stmt.endTok = elseBodyEnd + 1
				next = elseBodyEnd + 1
			}
		}
		bp.stmts = append(bp.stmts, stmt)
		return next
	}
	if rtgTokIsKeyword(p, start, rtgTokFor) {
		bodyStart := rtgFindNextTokenText(p, start+1, end, "{")
		if bodyStart <= start {
			return start
		}
		bodyEnd := rtgFindMatchingBrace(p, bodyStart, end)
		if bodyEnd <= bodyStart {
			return start
		}
		bp.stmts = append(bp.stmts, rtgStmt{kind: rtgStmtFor, startTok: start, endTok: bodyEnd + 1, exprStart: start + 1, exprEnd: bodyStart, bodyStart: bodyStart + 1, bodyEnd: bodyEnd})
		return bodyEnd + 1
	}
	if rtgTokIsKeyword(p, start, rtgTokBreak) {
		endTok := rtgStatementLineEnd(p, start+1, end)
		bp.stmts = append(bp.stmts, rtgStmt{kind: rtgStmtBreak, startTok: start, endTok: endTok})
		return endTok
	}
	if rtgTokIsKeyword(p, start, rtgTokContinue) {
		endTok := rtgStatementLineEnd(p, start+1, end)
		bp.stmts = append(bp.stmts, rtgStmt{kind: rtgStmtContinue, startTok: start, endTok: endTok})
		return endTok
	}
	if rtgTokIsKeyword(p, start, rtgTokGoto) {
		endTok := rtgStatementLineEnd(p, start+1, end)
		nameStart := 0
		nameEnd := 0
		if rtgTokIsKind(p, start+1, rtgTokIdent) {
			nameStart = p.toks[start+1].start
			nameEnd = p.toks[start+1].end
		}
		bp.stmts = append(bp.stmts, rtgStmt{kind: rtgStmtGoto, startTok: start, endTok: endTok, nameStart: nameStart, nameEnd: nameEnd})
		return endTok
	}
	if rtgTokIsKind(p, start, rtgTokIdent) && rtgTokTextIs(p, start+1, ":") {
		name := p.toks[start]
		bp.stmts = append(bp.stmts, rtgStmt{kind: rtgStmtLabel, startTok: start, endTok: start + 2, nameStart: name.start, nameEnd: name.end})
		return start + 2
	}
	if rtgTokIsKeyword(p, start, rtgTokVar) {
		endTok := rtgStatementLineEnd(p, start+1, end)
		nameStart := 0
		nameEnd := 0
		if rtgTokIsKind(p, start+1, rtgTokIdent) {
			nameStart = p.toks[start+1].start
			nameEnd = p.toks[start+1].end
		}
		bp.stmts = append(bp.stmts, rtgStmt{kind: rtgStmtVar, startTok: start, endTok: endTok, nameStart: nameStart, nameEnd: nameEnd})
		return endTok
	}
	lineEnd := rtgStatementLineEnd(p, start, end)
	assignTok := rtgFindAssignmentToken(p, start, lineEnd)
	if assignTok > start {
		kind := rtgStmtAssign
		if rtgTokTextIs(p, assignTok, ":=") {
			kind = rtgStmtShort
		}
		nameStart := 0
		nameEnd := 0
		if rtgTokIsKind(p, start, rtgTokIdent) {
			nameStart = p.toks[start].start
			nameEnd = p.toks[start].end
		}
		bp.stmts = append(bp.stmts, rtgStmt{kind: kind, startTok: start, endTok: lineEnd, exprStart: assignTok + 1, exprEnd: lineEnd, nameStart: nameStart, nameEnd: nameEnd})
		return lineEnd
	}
	bp.stmts = append(bp.stmts, rtgStmt{kind: rtgStmtExpr, startTok: start, endTok: lineEnd, exprStart: start, exprEnd: lineEnd})
	return lineEnd
}

func rtgFindIfStatementEnd(p rtgProgram, start int, end int) int {
	if !rtgTokIsKeyword(p, start, rtgTokIf) {
		return start
	}
	bodyStart := rtgFindNextTokenText(p, start+1, end, "{")
	if bodyStart <= start {
		return start
	}
	bodyEnd := rtgFindMatchingBrace(p, bodyStart, end)
	if bodyEnd <= bodyStart {
		return start
	}
	next := bodyEnd + 1
	if rtgTokIsKeyword(p, next, rtgTokElse) {
		if rtgTokIsKeyword(p, next+1, rtgTokIf) {
			return rtgFindIfStatementEnd(p, next+1, end)
		}
		if rtgTokTextIs(p, next+1, "{") {
			elseEnd := rtgFindMatchingBrace(p, next+1, end)
			if elseEnd <= next+1 {
				return start
			}
			return elseEnd + 1
		}
	}
	return next
}

func rtgStatementLineEnd(p rtgProgram, start int, end int) int {
	if start >= end {
		return end
	}
	line := p.toks[start].line
	i := start
	paren := 0
	brack := 0
	brace := 0
	for i < end {
		if i > start && p.toks[i].line != line && paren == 0 && brack == 0 && brace == 0 {
			return i
		}
		if rtgTokTextIs(p, i, "(") {
			paren++
		} else if rtgTokTextIs(p, i, ")") {
			paren--
		} else if rtgTokTextIs(p, i, "[") {
			brack++
		} else if rtgTokTextIs(p, i, "]") {
			brack--
		} else if rtgTokTextIs(p, i, "{") {
			brace++
		} else if rtgTokTextIs(p, i, "}") {
			if brace == 0 {
				return i
			}
			brace--
		}
		i++
	}
	return i
}

func rtgFindNextTokenText(p rtgProgram, start int, end int, text string) int {
	i := start
	for i < end {
		if rtgTokTextIs(p, i, text) {
			return i
		}
		i++
	}
	return start
}

func rtgFindMatchingBrace(p rtgProgram, openTok int, end int) int {
	if !rtgTokTextIs(p, openTok, "{") {
		return openTok
	}
	depth := 1
	i := openTok + 1
	for i < end {
		if rtgTokTextIs(p, i, "{") {
			depth++
		} else if rtgTokTextIs(p, i, "}") {
			depth--
			if depth == 0 {
				return i
			}
		}
		i++
	}
	return openTok
}

func rtgFindAssignmentToken(p rtgProgram, start int, end int) int {
	i := start
	paren := 0
	brack := 0
	for i < end {
		if rtgTokTextIs(p, i, "(") {
			paren++
		} else if rtgTokTextIs(p, i, ")") {
			paren--
		} else if rtgTokTextIs(p, i, "[") {
			brack++
		} else if rtgTokTextIs(p, i, "]") {
			brack--
		} else if paren == 0 && brack == 0 {
			if rtgTokTextIs(p, i, "=") || rtgTokTextIs(p, i, ":=") || rtgTokTextIs(p, i, "+=") || rtgTokTextIs(p, i, "-=") || rtgTokTextIs(p, i, "*=") || rtgTokTextIs(p, i, "/=") || rtgTokTextIs(p, i, "%=") {
				return i
			}
		}
		i++
	}
	return start
}

func rtgBuildMeta(p rtgProgram) rtgMeta {
	var m rtgMeta
	m.prog = p
	m.ok = true
	rtgInitBuiltinTypes(&m)

	i := 0
	for i < len(p.toks) && p.toks[i].kind != rtgTokEOF {
		if rtgTokIsKeyword(p, i, rtgTokType) || rtgTokIsKeyword(p, i, rtgTokVar) || rtgTokIsKeyword(p, i, rtgTokConst) {
			kind := p.toks[i].kind
			i++
			if rtgTokTextIs(p, i, "(") {
				groupEnd := rtgSkipBalanced(p, i, "(", ")")
				if groupEnd <= i {
					m.ok = false
					return m
				}
				j := i + 1
				for j < groupEnd-1 {
					if rtgTokIsKind(p, j, rtgTokIdent) {
						entryEnd := rtgStatementLineEnd(p, j, groupEnd-1)
						rtgParseTopDeclEntry(&m, kind, j, entryEnd)
						j = entryEnd
					} else {
						j++
					}
				}
				i = groupEnd
				continue
			}
			entryEnd := rtgStatementLineEnd(p, i, len(p.toks))
			rtgParseTopDeclEntry(&m, kind, i, entryEnd)
			i = entryEnd
			continue
		}
		if rtgTokIsKeyword(p, i, rtgTokFunc) {
			fnIndex := rtgFindFuncDeclByStart(p, i)
			if fnIndex >= 0 {
				rtgParseFuncInfo(&m, fnIndex)
				i = p.funcs[fnIndex].endTok
				continue
			}
		}
		i++
	}

	return m
}

func rtgInitBuiltinTypes(m *rtgMeta) {
	m.types = append(m.types, rtgTypeInfo{kind: rtgTypeInvalid, size: 0})
	m.types = append(m.types, rtgTypeInfo{kind: rtgTypeInt, size: 8})
	m.types = append(m.types, rtgTypeInfo{kind: rtgTypeInt64, size: 8})
	m.types = append(m.types, rtgTypeInfo{kind: rtgTypeByte, size: 1})
	m.types = append(m.types, rtgTypeInfo{kind: rtgTypeBool, size: 1})
	m.types = append(m.types, rtgTypeInfo{kind: rtgTypeString, size: 16})
	m.types = append(m.types, rtgTypeInfo{kind: rtgTypeFloat64, size: 8})
}

func rtgParseTopDeclEntry(m *rtgMeta, kind int, start int, end int) {
	p := m.prog
	if start >= end || !rtgTokIsKind(p, start, rtgTokIdent) {
		m.ok = false
		return
	}
	name := p.toks[start]
	if kind == rtgTokType {
		typeResult := rtgParseType(m, start+1, end)
		if typeResult.typ == 0 || typeResult.next > end {
			m.ok = false
			return
		}
		if m.types[typeResult.typ].kind == rtgTypeStruct || m.types[typeResult.typ].kind == rtgTypePointer || m.types[typeResult.typ].kind == rtgTypeSlice {
			m.types[typeResult.typ].nameStart = name.start
			m.types[typeResult.typ].nameEnd = name.end
		} else {
			rtgAddType(m, rtgTypeInfo{kind: rtgTypeNamed, elem: typeResult.typ, size: rtgTypeSize(m, typeResult.typ), nameStart: name.start, nameEnd: name.end})
		}
		return
	}
	eq := rtgFindTokenTextInRange(p, start+1, end, "=")
	typeEnd := end
	initStart := end
	initEnd := end
	if eq > start {
		typeEnd = eq
		initStart = eq + 1
		initEnd = end
	}
	typ := 0
	if start+1 < typeEnd {
		typeResult := rtgParseType(m, start+1, typeEnd)
		typ = typeResult.typ
	}
	m.globals = append(m.globals, rtgSymbolInfo{nameStart: name.start, nameEnd: name.end, kind: kind, typ: typ, initStart: initStart, initEnd: initEnd})
}

func rtgParseFuncInfo(m *rtgMeta, fnIndex int) {
	p := m.prog
	fn := p.funcs[fnIndex]
	nameStart := fn.nameStart
	nameEnd := fn.nameEnd
	nameTok := fn.startTok + 1
	lparen := rtgFindNextTokenText(p, nameTok+1, fn.bodyStart, "(")
	if lparen <= nameTok {
		m.ok = false
		return
	}
	rparen := rtgFindMatchingExprClose(p, lparen+1, fn.bodyStart, "(", ")")
	if rparen <= lparen {
		m.ok = false
		return
	}
	firstParam := len(m.params)
	paramCount := 0
	rtgParseParamList(m, lparen+1, rparen, &paramCount)
	resultType := 0
	if rparen+1 < fn.bodyStart {
		typeResult := rtgParseType(m, rparen+1, fn.bodyStart)
		resultType = typeResult.typ
	}
	m.funcs = append(m.funcs, rtgFuncInfo{declIndex: fnIndex, nameStart: nameStart, nameEnd: nameEnd, firstParam: firstParam, paramCount: paramCount, resultType: resultType, bodyStart: fn.bodyStart + 1, bodyEnd: fn.bodyEnd})
}

func rtgParseParamList(m *rtgMeta, start int, end int, count *int) {
	p := m.prog
	i := start
	for i < end {
		for i < end && rtgTokTextIs(p, i, ",") {
			i++
		}
		if i >= end {
			return
		}
		if !rtgTokIsKind(p, i, rtgTokIdent) {
			m.ok = false
			return
		}
		name := p.toks[i]
		typeStart := i + 1
		entryEnd := rtgFindParamEntryEnd(p, typeStart, end)
		typeResult := rtgParseType(m, typeStart, entryEnd)
		if typeResult.typ == 0 {
			m.ok = false
			return
		}
		m.params = append(m.params, rtgSymbolInfo{nameStart: name.start, nameEnd: name.end, typ: typeResult.typ})
		*count = *count + 1
		i = entryEnd
		if rtgTokTextIs(p, i, ",") {
			i++
		}
	}
}

func rtgFindParamEntryEnd(p rtgProgram, start int, end int) int {
	i := start
	depth := 0
	for i < end {
		if depth == 0 && rtgTokTextIs(p, i, ",") {
			return i
		}
		if rtgTokTextIs(p, i, "[") || rtgTokTextIs(p, i, "{") || rtgTokTextIs(p, i, "(") {
			depth++
		} else if rtgTokTextIs(p, i, "]") || rtgTokTextIs(p, i, "}") || rtgTokTextIs(p, i, ")") {
			depth--
		}
		i++
	}
	return end
}

func rtgParseType(m *rtgMeta, start int, end int) rtgTypeResult {
	p := m.prog
	if start >= end {
		return rtgTypeResult{next: start}
	}
	if rtgTokTextIs(p, start, "*") {
		elem := rtgParseType(m, start+1, end)
		if elem.typ == 0 {
			return rtgTypeResult{next: start}
		}
		return rtgTypeResult{typ: rtgAddType(m, rtgTypeInfo{kind: rtgTypePointer, elem: elem.typ, size: 8}), next: elem.next}
	}
	if rtgTokTextIs(p, start, "[") && rtgTokTextIs(p, start+1, "]") {
		elem := rtgParseType(m, start+2, end)
		if elem.typ == 0 {
			return rtgTypeResult{next: start}
		}
		return rtgTypeResult{typ: rtgAddType(m, rtgTypeInfo{kind: rtgTypeSlice, elem: elem.typ, size: 24}), next: elem.next}
	}
	if rtgTokIsKeyword(p, start, rtgTokStruct) && rtgTokTextIs(p, start+1, "{") {
		closeTok := rtgFindMatchingBrace(p, start+1, end)
		if closeTok <= start+1 {
			return rtgTypeResult{next: start}
		}
		firstField := len(m.fields)
		count := 0
		offset := 0
		i := start + 2
		for i < closeTok {
			if rtgTokIsKind(p, i, rtgTokIdent) {
				name := p.toks[i]
				lineEnd := rtgStatementLineEnd(p, i, closeTok)
				fieldType := rtgParseType(m, i+1, lineEnd)
				if fieldType.typ == 0 {
					return rtgTypeResult{next: start}
				}
				offset = rtgAlignTo8(offset)
				m.fields = append(m.fields, rtgFieldInfo{nameStart: name.start, nameEnd: name.end, typ: fieldType.typ, offset: offset})
				offset += rtgTypeSize(m, fieldType.typ)
				count++
				i = lineEnd
			} else {
				i++
			}
		}
		return rtgTypeResult{typ: rtgAddType(m, rtgTypeInfo{kind: rtgTypeStruct, first: firstField, count: count, size: rtgAlignTo8(offset)}), next: closeTok + 1}
	}
	if rtgTokIsKind(p, start, rtgTokIdent) {
		tok := p.toks[start]
		builtin := rtgBuiltinTypeFromToken(p, tok)
		if builtin != 0 {
			return rtgTypeResult{typ: builtin, next: start + 1}
		}
		return rtgTypeResult{typ: rtgNamedTypeFromToken(m, tok), next: start + 1}
	}
	return rtgTypeResult{next: start}
}

func rtgBuiltinTypeFromToken(p rtgProgram, tok rtgToken) int {
	if rtgBytesEqualText(p.src, tok.start, tok.end, "int") {
		return rtgTypeInt
	}
	if rtgBytesEqualText(p.src, tok.start, tok.end, "int64") {
		return rtgTypeInt64
	}
	if rtgBytesEqualText(p.src, tok.start, tok.end, "byte") {
		return rtgTypeByte
	}
	if rtgBytesEqualText(p.src, tok.start, tok.end, "bool") {
		return rtgTypeBool
	}
	if rtgBytesEqualText(p.src, tok.start, tok.end, "string") {
		return rtgTypeString
	}
	if rtgBytesEqualText(p.src, tok.start, tok.end, "float64") {
		return rtgTypeFloat64
	}
	return 0
}

func rtgNamedTypeFromToken(m *rtgMeta, tok rtgToken) int {
	for i := 0; i < len(m.types); i++ {
		if m.types[i].nameEnd > m.types[i].nameStart && rtgBytesEqualRange(m.prog.src, m.types[i].nameStart, m.types[i].nameEnd, tok.start, tok.end) {
			return i
		}
	}
	return rtgAddType(m, rtgTypeInfo{kind: rtgTypeNamed, size: 8, nameStart: tok.start, nameEnd: tok.end})
}

func rtgAddType(m *rtgMeta, typ rtgTypeInfo) int {
	m.types = append(m.types, typ)
	return len(m.types) - 1
}

func rtgTypeSize(m *rtgMeta, typ int) int {
	if typ >= 0 && typ < len(m.types) {
		size := m.types[typ].size
		if size > 0 {
			return size
		}
	}
	return 8
}

func rtgResolveType(m *rtgMeta, typ int) rtgTypeInfo {
	if typ >= 0 && typ < len(m.types) {
		t := m.types[typ]
		if t.kind == rtgTypeNamed && t.elem > 0 && t.elem < len(m.types) {
			return m.types[t.elem]
		}
		return t
	}
	var t rtgTypeInfo
	return t
}

func rtgTypeIsSlice(m *rtgMeta, typ int) bool {
	t := rtgResolveType(m, typ)
	return t.kind == rtgTypeSlice
}

func rtgTypeIsString(m *rtgMeta, typ int) bool {
	t := rtgResolveType(m, typ)
	return t.kind == rtgTypeString
}

func rtgTypeIsStruct(m *rtgMeta, typ int) bool {
	t := rtgResolveType(m, typ)
	return t.kind == rtgTypeStruct
}

func rtgAlignTo8(v int) int {
	rem := v % 8
	if rem == 0 {
		return v
	}
	return v + 8 - rem
}

func rtgFindFuncDeclByStart(p rtgProgram, start int) int {
	for i := 0; i < len(p.funcs); i++ {
		if p.funcs[i].startTok == start {
			return i
		}
	}
	return -1
}

func rtgFindFuncByName(p rtgProgram, name string) int {
	for i := 0; i < len(p.funcs); i++ {
		if rtgBytesEqualText(p.src, p.funcs[i].nameStart, p.funcs[i].nameEnd, name) {
			return i
		}
	}
	return -1
}

func rtgFindFuncInfoByName(meta rtgMeta, name string) int {
	for i := 0; i < len(meta.funcs); i++ {
		if rtgBytesEqualText(meta.prog.src, meta.funcs[i].nameStart, meta.funcs[i].nameEnd, name) {
			return i
		}
	}
	return -1
}

func rtgFindTokenByStart(p rtgProgram, start int) int {
	for i := 0; i < len(p.toks); i++ {
		if p.toks[i].start == start {
			return i
		}
	}
	return 0
}

func rtgFindTokenTextInRange(p rtgProgram, start int, end int, text string) int {
	i := start
	for i < end {
		if rtgTokTextIs(p, i, text) {
			return i
		}
		i++
	}
	return start
}

func rtgFindTokenTextAfter(p rtgProgram, start int, end int, text string) int {
	i := start
	for i < end {
		if rtgTokTextIs(p, i, text) {
			return i
		}
		i++
	}
	return start - 1
}

func rtgBytesEqualRange(src []byte, aStart int, aEnd int, bStart int, bEnd int) bool {
	if aEnd-aStart != bEnd-bStart {
		return false
	}
	for i := 0; i < aEnd-aStart; i++ {
		if src[aStart+i] != src[bStart+i] {
			return false
		}
	}
	return true
}

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
const rtgTokSwitch = 19
const rtgTokCase = 20
const rtgTokDefault = 21
const rtgTokOp = 22

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
const rtgStmtSwitch = 12

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
	src     []byte
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
	expr   int
	mem    bool
	global bool
	ok     bool
}

type rtgCompileResult struct {
	data      []byte
	ok        bool
	errorCode int
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
	gotoLabels     []rtgGlobalInfo
	breakLabels    []int
	continueLabels []int
	breakDepth     int
	continueDepth  int
	stackSize      int
	errorCode      int
	envDataOff     int
	envLenOff      int
}

const rtgErrNone = 0
const rtgErrNoAppMain = 1
const rtgErrGlobals = 2
const rtgErrEntryArgs = 3
const rtgErrScalarFunction = 4
const rtgErrBindParams = 5
const rtgErrFunctionBody = 6
const rtgErrBodyParse = 7
const rtgErrStmt = 8
const rtgErrExprParse = 9
const rtgErrExprNonCall = 10
const rtgErrExprEmit = 11
const rtgErrReturnExprParse = 12
const rtgErrReturnStruct = 13
const rtgErrReturnSlice = 14
const rtgErrReturnInt = 15
const rtgErrLinearNoAppMain = 16
const rtgErrLinearNoMetaAppMain = 17
const rtgErrLinearBodyParse = 18

func rtgCompilerError(g *rtgLinearGen, code int) bool {
	return false
}

func rtgCompileFail(code int) rtgCompileResult {
	var r rtgCompileResult
	return r
}

func rtgCompileFailFromGen(g *rtgLinearGen, fallback int) rtgCompileResult {
	var r rtgCompileResult
	return r
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
	var prog rtgProgram
	prog = rtgParseProgram(src)
	if !prog.ok {
		return 1
	}
	var meta rtgMeta
	meta = rtgBuildMeta(&prog)
	if !meta.ok {
		return 1
	}
	var result rtgCompileResult
	result = rtgTryCompileScalarProgram(prog, &meta)
	if result.ok {
		write(output, result.data, 0)
		return 0
	}
	result = rtgTryCompileLinearAppMain(prog, &meta)
	if result.ok {
		write(output, result.data, 0)
		return 0
	}
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
				var decl rtgDecl
				decl.kind = kind
				decl.nameStart = p.toks[start].start
				decl.nameEnd = p.toks[start].end
				decl.startTok = start
				decl.endTok = end
				p.decls = append(p.decls, decl)
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
			var decl rtgDecl
			decl.kind = kind
			decl.nameStart = name.start
			decl.nameEnd = name.end
			decl.startTok = start
			decl.endTok = end
			p.decls = append(p.decls, decl)
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
			i++
			start := i - 1
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
	if end-start == 6 && rtgBytesEqualTextAt(src, start, "switch") {
		return rtgTokSwitch
	}
	if end-start == 4 && rtgBytesEqualTextAt(src, start, "case") {
		return rtgTokCase
	}
	if end-start == 7 && rtgBytesEqualTextAt(src, start, "default") {
		return rtgTokDefault
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
	if c >= 'a' {
		if c <= 'z' {
			return true
		}
	}
	if c >= 'A' {
		if c <= 'Z' {
			return true
		}
	}
	if c == '_' {
		return true
	}
	return false
}

func rtgIsIdentPart(c byte) bool {
	if rtgIsIdentStart(c) {
		return true
	}
	if rtgIsDigit(c) {
		return true
	}
	return false
}

func rtgIsDigit(c byte) bool {
	if c >= '0' {
		if c <= '9' {
			return true
		}
	}
	return false
}

func rtgIsBasedDigit(c byte) bool {
	if c >= '0' {
		if c <= '9' {
			return true
		}
	}
	if c >= 'a' {
		if c <= 'f' {
			return true
		}
	}
	if c >= 'A' {
		if c <= 'F' {
			return true
		}
	}
	return false
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
			} else if src[i] == 'r' {
				out = append(out, '\r')
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
	} else if tok.end-start > 1 && src[start] == '0' {
		base = 8
		start++
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
	if src[i] == 'r' {
		return 13
	}
	if src[i] == '\\' {
		return 92
	}
	if src[i] == '\'' {
		return 39
	}
	return int(src[i])
}

func rtgTryCompileScalarProgram(p rtgProgram, meta *rtgMeta) rtgCompileResult {
	appIndex := rtgFindFuncInfoByName(meta, "appMain")
	if appIndex < 0 {
		result := rtgCompileFail(rtgErrNoAppMain)
		return result
	}
	var g rtgLinearGen
	g.prog = p
	g.meta.prog = meta.prog
	g.meta.src = meta.src
	g.meta.types = meta.types
	g.meta.fields = meta.fields
	g.meta.globals = meta.globals
	g.meta.params = meta.params
	g.meta.funcs = meta.funcs
	g.meta.ok = meta.ok
	g.stackSize = 65536
	rtgAsmInit(&g.asm)
	for i := 0; i < len(meta.funcs); i++ {
		g.funcLabels = append(g.funcLabels, rtgAsmNewLabel(&g.asm))
	}
	if !rtgLinearInitGlobals(&g) {
		result := rtgCompileFailFromGen(&g, rtgErrGlobals)
		return result
	}
	if !rtgEmitProgramEntryArgs(&g, meta.funcs[appIndex]) {
		result := rtgCompileFailFromGen(&g, rtgErrEntryArgs)
		return result
	}
	rtgAsmCallLabel(&g.asm, g.funcLabels[appIndex])
	rtgAsmMovRdiRax(&g.asm)
	rtgAsmMovRaxImm(&g.asm, 60)
	rtgAsmSyscall(&g.asm)
	for i := 0; i < len(meta.funcs); i++ {
		if !rtgEmitScalarFunction(&g, i) {
			result := rtgCompileFailFromGen(&g, rtgErrScalarFunction)
			return result
		}
	}
	data := rtgAsmImage(&g.asm)
	var result rtgCompileResult
	result.data = data
	result.ok = true
	return result
}

func rtgEmitProgramEntryArgs(g *rtgLinearGen, app rtgFuncInfo) bool {
	argsOff := g.asm.bssSize
	g.asm.bssSize += 32768
	g.envDataOff = g.asm.bssSize
	g.asm.bssSize += 32768
	g.envLenOff = g.asm.bssSize
	g.asm.bssSize += 8
	rtgAsmBuildArgvEnvSlices(&g.asm, argsOff, g.envDataOff, g.envLenOff)
	if app.paramCount == 0 {
		return true
	}
	first := g.meta.params[app.firstParam]
	if !rtgTypeIsSlice(&g.meta, first.typ) {
		return false
	}
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
	oldStackUsed := g.stackUsed
	oldGotoLabels := g.gotoLabels
	var locals []rtgLocalInfo
	var gotoLabels []rtgGlobalInfo
	g.locals = locals
	g.gotoLabels = gotoLabels
	g.breakDepth = 0
	g.continueDepth = 0
	g.currentFunc = fnInfoIndex
	g.returnStruct = 0
	g.stackUsed = 0
	rtgAsmMarkLabel(&g.asm, g.funcLabels[fnInfoIndex])
	rtgAsmPushRbp(&g.asm)
	rtgAsmMovRbpRsp(&g.asm)
	rtgAsmSubRspImm(&g.asm, g.stackSize)
	if rtgTypeIsStruct(&g.meta, metaFn.resultType) {
		g.returnStruct = rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
		rtgAsmStoreRdiStack(&g.asm, g.returnStruct)
	}
	if !rtgBindFunctionParams(g, metaFn) {
		return rtgCompilerError(g, rtgErrBindParams)
	}
	if !rtgEmitLinearRange(g, fn.bodyStart+1, fn.bodyEnd) {
		return rtgCompilerError(g, rtgErrFunctionBody)
	}
	rtgAsmMovRaxImm(&g.asm, 0)
	rtgAsmLeave(&g.asm)
	rtgAsmRet(&g.asm)
	g.locals = oldLocals
	g.breakDepth = oldBreak
	g.continueDepth = oldContinue
	g.currentFunc = oldCurrent
	g.returnStruct = oldReturnStruct
	g.stackUsed = oldStackUsed
	g.gotoLabels = oldGotoLabels
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

func rtgTryCompileLinearAppMain(p rtgProgram, meta *rtgMeta) rtgCompileResult {
	fnIndex := rtgFindFuncByName(p, "appMain")
	if fnIndex < 0 {
		result := rtgCompileFail(rtgErrLinearNoAppMain)
		return result
	}
	metaIndex := rtgFindFuncInfoByName(meta, "appMain")
	if metaIndex < 0 {
		result := rtgCompileFail(rtgErrLinearNoMetaAppMain)
		return result
	}
	fn := p.funcs[fnIndex]
	body := rtgParseFunctionBody(p, fn)
	if !body.ok {
		result := rtgCompileFail(rtgErrLinearBodyParse)
		return result
	}
	var g rtgLinearGen
	g.prog = p
	g.meta.prog = meta.prog
	g.meta.src = meta.src
	g.meta.types = meta.types
	g.meta.fields = meta.fields
	g.meta.globals = meta.globals
	g.meta.params = meta.params
	g.meta.funcs = meta.funcs
	g.meta.ok = meta.ok
	g.stackSize = 65536
	rtgAsmInit(&g.asm)
	rtgAsmPushRbp(&g.asm)
	rtgAsmMovRbpRsp(&g.asm)
	rtgAsmSubRspImm(&g.asm, g.stackSize)
	if !rtgLinearInitGlobals(&g) {
		result := rtgCompileFailFromGen(&g, rtgErrGlobals)
		return result
	}
	app := meta.funcs[metaIndex]
	if app.paramCount > 0 {
		first := meta.params[app.firstParam]
		if !rtgTypeIsSlice(meta, first.typ) {
			result := rtgCompileFailFromGen(&g, rtgErrEntryArgs)
			return result
		}
		offset := rtgAddTypedLocal(&g, first.nameStart, first.nameEnd, first.typ)
		argsOff := g.asm.bssSize
		g.asm.bssSize += 32768
		g.envDataOff = g.asm.bssSize
		g.asm.bssSize += 32768
		g.envLenOff = g.asm.bssSize
		g.asm.bssSize += 8
		rtgAsmBuildArgvEnvSlices(&g.asm, argsOff, g.envDataOff, g.envLenOff)
		rtgAsmStoreRdiStack(&g.asm, offset)
		rtgAsmStoreRsiStack(&g.asm, offset-8)
		rtgAsmStoreRdxStack(&g.asm, offset-16)
	}
	if !rtgEmitLinearRange(&g, fn.bodyStart+1, fn.bodyEnd) {
		result := rtgCompileFailFromGen(&g, rtgErrFunctionBody)
		return result
	}
	rtgAsmMovRaxImm(&g.asm, 60)
	rtgAsmMovRdiImm(&g.asm, 0)
	rtgAsmSyscall(&g.asm)
	data := rtgAsmImage(&g.asm)
	var result rtgCompileResult
	result.data = data
	result.ok = true
	return result
}

func rtgEmitLinearRange(g *rtgLinearGen, start int, end int) bool {
	var bp rtgBodyParse
	var stmts []rtgStmt
	bp.prog = g.prog
	bp.stmts = stmts
	bp.ok = true
	i := start
	for bp.ok && i < end {
		if i < 0 || i >= len(bp.prog.toks) {
			return true
		}
		if rtgTokTextIs(bp.prog, i, ";") {
			i++
			continue
		}
		if bp.prog.toks[i].start < 0 || bp.prog.toks[i].start >= len(bp.prog.src) {
			return true
		}
		if rtgTokIsKind(bp.prog, i, rtgTokEOF) {
			return true
		}
		if rtgTokTextIs(bp.prog, i, "}") {
			return true
		}
		before := len(bp.stmts)
		next := rtgParseOneStatement(&bp, i, end)
		if !bp.ok || next <= i || len(bp.stmts) <= before {
			return rtgCompilerError(g, rtgErrBodyParse)
		}
		stmt := bp.stmts[len(bp.stmts)-1]
		i = next
		if !rtgEmitLinearStmt(g, stmt) {
			return rtgCompilerError(g, rtgErrStmt)
		}
	}
	if !bp.ok {
		return rtgCompilerError(g, rtgErrBodyParse)
	}
	return true
}

func rtgEmitLinearStmt(g *rtgLinearGen, stmt rtgStmt) bool {
	p := g.prog
	if stmt.kind == rtgStmtExpr {
		if rtgEmitLinearPrintStmt(g, stmt) {
			return true
		}
		if rtgEmitLinearIncDec(g, stmt.startTok, stmt.endTok) {
			return true
		}
		ep := rtgParseExpression(p, stmt.exprStart, stmt.exprEnd)
		if !ep.ok || len(ep.exprs) == 0 {
			return rtgCompilerError(g, rtgErrExprParse)
		}
		rootIndex := len(ep.exprs) - 1
		root := ep.exprs[rootIndex]
		if root.kind != rtgExprCall {
			return rtgCompilerError(g, rtgErrExprNonCall)
		}
		if !rtgEmitIntExpr(g, ep, rootIndex) {
			return rtgCompilerError(g, rtgErrExprEmit)
		}
		return true
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
			return rtgCompilerError(g, rtgErrReturnExprParse)
		}
		rootIndex := len(ep.exprs) - 1
		resultType := rtgCurrentResultType(g)
		if rtgTypeIsStruct(&g.meta, resultType) {
			if !rtgEmitStructReturnExpr(g, ep, rootIndex) {
				return rtgCompilerError(g, rtgErrReturnStruct)
			}
		} else if rtgTypeIsSlice(&g.meta, resultType) {
			if !rtgEmitSliceReturnExpr(g, ep, rootIndex) {
				return rtgCompilerError(g, rtgErrReturnSlice)
			}
		} else if rtgTypeIsString(&g.meta, resultType) {
			if !rtgEmitStringReturnExpr(g, ep, rootIndex) {
				return rtgCompilerError(g, rtgErrReturnInt)
			}
		} else {
			if !rtgEmitIntExpr(g, ep, rootIndex) {
				return rtgCompilerError(g, rtgErrReturnInt)
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
	if stmt.kind == rtgStmtSwitch {
		return rtgEmitLinearSwitch(g, stmt)
	}
	if stmt.kind == rtgStmtGoto {
		label := rtgFindOrCreateGotoLabel(g, stmt.nameStart, stmt.nameEnd)
		rtgAsmJmpLabel(&g.asm, label)
		return true
	}
	if stmt.kind == rtgStmtLabel {
		label := rtgFindOrCreateGotoLabel(g, stmt.nameStart, stmt.nameEnd)
		rtgAsmMarkLabel(&g.asm, label)
		return true
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

func rtgFindOrCreateGotoLabel(g *rtgLinearGen, nameStart int, nameEnd int) int {
	for i := 0; i < len(g.gotoLabels); i++ {
		info := g.gotoLabels[i]
		if rtgBytesEqualRange(g.prog.src, info.nameStart, info.nameEnd, nameStart, nameEnd) {
			return info.offset
		}
	}
	label := rtgAsmNewLabel(&g.asm)
	g.gotoLabels = append(g.gotoLabels, rtgGlobalInfo{nameStart: nameStart, nameEnd: nameEnd, offset: label})
	return label
}

func rtgEmitLinearIf(g *rtgLinearGen, stmt rtgStmt) bool {
	p := g.prog
	ep := rtgParseExpression(p, stmt.exprStart, stmt.exprEnd)
	if !ep.ok || len(ep.exprs) == 0 {
		return false
	}
	rootIndex := len(ep.exprs) - 1
	if !rtgEmitIntExpr(g, ep, rootIndex) {
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
			var stmts []rtgStmt
			nested.prog = p
			nested.stmts = stmts
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
	oldBreakDepth := g.breakDepth
	oldContinueDepth := g.continueDepth
	g.breakLabels = append(g.breakLabels, endLabel)
	g.continueLabels = append(g.continueLabels, startLabel)
	g.breakDepth = len(g.breakLabels)
	g.continueDepth = len(g.continueLabels)
	rtgAsmMarkLabel(&g.asm, startLabel)
	if stmt.exprStart < stmt.exprEnd {
		ep := rtgParseExpression(p, stmt.exprStart, stmt.exprEnd)
		if !ep.ok || len(ep.exprs) == 0 {
			return false
		}
		rootIndex := len(ep.exprs) - 1
		if !rtgEmitIntExpr(g, ep, rootIndex) {
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
	g.breakDepth = oldBreakDepth
	g.continueDepth = oldContinueDepth
	return true
}

func rtgEmitLinearSwitch(g *rtgLinearGen, stmt rtgStmt) bool {
	p := g.prog
	if stmt.exprStart >= stmt.exprEnd {
		return false
	}
	ep := rtgParseExpression(p, stmt.exprStart, stmt.exprEnd)
	if !ep.ok || len(ep.exprs) == 0 {
		return false
	}
	rootIndex := len(ep.exprs) - 1
	switchType := rtgInferParsedExprType(g, ep, rootIndex)
	stringSwitch := rtgTypeIsString(&g.meta, switchType)
	valueOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	lenOffset := 0
	if stringSwitch {
		lenOffset = rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
		if !rtgEmitStringValueRegs(g, ep, rootIndex) {
			return false
		}
		rtgAsmStoreRaxStack(&g.asm, valueOffset)
		rtgAsmStoreRdxStack(&g.asm, lenOffset)
	} else {
		if !rtgEmitIntExpr(g, ep, rootIndex) {
			return false
		}
		rtgAsmStoreRaxStack(&g.asm, valueOffset)
	}

	endLabel := rtgAsmNewLabel(&g.asm)
	oldBreakDepth := g.breakDepth
	g.breakLabels = append(g.breakLabels, endLabel)
	g.breakDepth = len(g.breakLabels)

	var clauseStarts []int
	var clauseLabels []int
	defaultLabel := endLabel
	hasDefault := false
	i := stmt.bodyStart
	for i < stmt.bodyEnd {
		clause := rtgFindNextSwitchClause(p, i, stmt.bodyEnd)
		if clause >= stmt.bodyEnd {
			break
		}
		label := rtgAsmNewLabel(&g.asm)
		clauseStarts = append(clauseStarts, clause)
		clauseLabels = append(clauseLabels, label)
		if rtgTokIsKeyword(p, clause, rtgTokDefault) {
			defaultLabel = label
			hasDefault = true
		}
		i = clause + 1
	}
	for i := 0; i < len(clauseStarts); i++ {
		clause := clauseStarts[i]
		if rtgTokIsKeyword(p, clause, rtgTokCase) {
			if !rtgEmitSwitchCaseTests(g, stmt, clause, valueOffset, lenOffset, stringSwitch, clauseLabels[i]) {
				return false
			}
		}
	}
	if hasDefault {
		rtgAsmJmpLabel(&g.asm, defaultLabel)
	} else {
		rtgAsmJmpLabel(&g.asm, endLabel)
	}
	for i := 0; i < len(clauseStarts); i++ {
		clause := clauseStarts[i]
		colon := rtgFindSwitchClauseColon(p, clause+1, stmt.bodyEnd)
		if colon <= clause {
			return false
		}
		bodyEnd := rtgFindNextSwitchClause(p, colon+1, stmt.bodyEnd)
		rtgAsmMarkLabel(&g.asm, clauseLabels[i])
		if !rtgEmitLinearRange(g, colon+1, bodyEnd) {
			return false
		}
		rtgAsmJmpLabel(&g.asm, endLabel)
	}
	rtgAsmMarkLabel(&g.asm, endLabel)
	g.breakDepth = oldBreakDepth
	return true
}

func rtgEmitSwitchCaseTests(g *rtgLinearGen, stmt rtgStmt, clause int, valueOffset int, lenOffset int, stringSwitch bool, matchLabel int) bool {
	p := g.prog
	colon := rtgFindSwitchClauseColon(p, clause+1, stmt.bodyEnd)
	if colon <= clause+1 {
		return false
	}
	i := clause + 1
	for i < colon {
		valueEnd := rtgFindExprBoundary(p, i, colon)
		if valueEnd <= i {
			return false
		}
		ep := rtgParseExpression(p, i, valueEnd)
		if !ep.ok || len(ep.exprs) == 0 {
			return false
		}
		rootIndex := len(ep.exprs) - 1
		if stringSwitch {
			if !rtgEmitSwitchStringCaseTest(g, valueOffset, lenOffset, ep, rootIndex, matchLabel) {
				return false
			}
		} else {
			rtgAsmLoadRaxStack(&g.asm, valueOffset)
			rtgAsmPushRax(&g.asm)
			if !rtgEmitIntExpr(g, ep, rootIndex) {
				return false
			}
			rtgAsmPopRcx(&g.asm)
			rtgAsmCmpRcxRaxSet(&g.asm, 0x94)
			rtgAsmCmpRaxImm8(&g.asm, 0)
			rtgAsmJnzLabel(&g.asm, matchLabel)
		}
		i = valueEnd
		if rtgTokTextIs(p, i, ",") {
			i++
		}
	}
	return true
}

func rtgEmitSwitchStringCaseTest(g *rtgLinearGen, valueOffset int, lenOffset int, ep rtgExprParse, idx int, matchLabel int) bool {
	rightPtr := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	rightLen := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	index := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	if !rtgEmitStringValueRegs(g, ep, idx) {
		return false
	}
	rtgAsmStoreRaxStack(&g.asm, rightPtr)
	rtgAsmStoreRdxStack(&g.asm, rightLen)
	noMatch := rtgAsmNewLabel(&g.asm)
	loopLabel := rtgAsmNewLabel(&g.asm)

	rtgAsmLoadRaxStack(&g.asm, lenOffset)
	rtgAsmPushRax(&g.asm)
	rtgAsmLoadRaxStack(&g.asm, rightLen)
	rtgAsmPopRcx(&g.asm)
	rtgAsmCmpRcxRaxSet(&g.asm, 0x94)
	rtgAsmCmpRaxImm8(&g.asm, 0)
	rtgAsmJzLabel(&g.asm, noMatch)

	rtgAsmMovRaxImm(&g.asm, 0)
	rtgAsmStoreRaxStack(&g.asm, index)
	rtgAsmMarkLabel(&g.asm, loopLabel)
	rtgAsmLoadRaxStack(&g.asm, index)
	rtgAsmPushRax(&g.asm)
	rtgAsmLoadRaxStack(&g.asm, lenOffset)
	rtgAsmPopRcx(&g.asm)
	rtgAsmCmpRcxRaxSet(&g.asm, 0x9d)
	rtgAsmCmpRaxImm8(&g.asm, 0)
	rtgAsmJnzLabel(&g.asm, matchLabel)

	rtgAsmLoadRaxStack(&g.asm, index)
	rtgAsmPushRax(&g.asm)
	rtgAsmLoadRaxStack(&g.asm, valueOffset)
	rtgAsmPopRcx(&g.asm)
	rtgAsmLoadByteRaxIndexRcx(&g.asm)
	rtgAsmPushRax(&g.asm)
	rtgAsmLoadRaxStack(&g.asm, index)
	rtgAsmPushRax(&g.asm)
	rtgAsmLoadRaxStack(&g.asm, rightPtr)
	rtgAsmPopRcx(&g.asm)
	rtgAsmLoadByteRaxIndexRcx(&g.asm)
	rtgAsmPopRcx(&g.asm)
	rtgAsmCmpRcxRaxSet(&g.asm, 0x94)
	rtgAsmCmpRaxImm8(&g.asm, 0)
	rtgAsmJzLabel(&g.asm, noMatch)
	rtgAsmLoadRaxStack(&g.asm, index)
	rtgAsmIncRax(&g.asm)
	rtgAsmStoreRaxStack(&g.asm, index)
	rtgAsmJmpLabel(&g.asm, loopLabel)
	rtgAsmMarkLabel(&g.asm, noMatch)
	return true
}

func rtgFindNextSwitchClause(p rtgProgram, start int, end int) int {
	depth := 0
	i := start
	for i < end {
		if depth == 0 && (rtgTokIsKeyword(p, i, rtgTokCase) || rtgTokIsKeyword(p, i, rtgTokDefault)) {
			return i
		}
		if rtgTokTextIs(p, i, "{") {
			depth++
		} else if rtgTokTextIs(p, i, "}") {
			if depth > 0 {
				depth--
			}
		}
		i++
	}
	return end
}

func rtgFindSwitchClauseColon(p rtgProgram, start int, end int) int {
	paren := 0
	brack := 0
	brace := 0
	i := start
	for i < end {
		if paren == 0 && brack == 0 && brace == 0 && rtgTokTextIs(p, i, ":") {
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
				return end
			}
			brace--
		}
		i++
	}
	return end
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
	oldBreakDepth := g.breakDepth
	oldContinueDepth := g.continueDepth
	g.breakLabels = append(g.breakLabels, endLabel)
	g.continueLabels = append(g.continueLabels, postLabel)
	g.breakDepth = len(g.breakLabels)
	g.continueDepth = len(g.continueLabels)
	rtgAsmMarkLabel(&g.asm, startLabel)
	if semi1+1 < semi2 {
		ep := rtgParseExpression(p, semi1+1, semi2)
		if !ep.ok || len(ep.exprs) == 0 {
			return false
		}
		rootIndex := len(ep.exprs) - 1
		if !rtgEmitIntExpr(g, ep, rootIndex) {
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
	g.breakDepth = oldBreakDepth
	g.continueDepth = oldContinueDepth
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
	if start+2 > end {
		return false
	}
	opTok := end - 1
	if !rtgTokTextIs(p, opTok, "++") && !rtgTokTextIs(p, opTok, "--") {
		return false
	}
	ep := rtgParseExpression(p, start, opTok)
	if !ep.ok || len(ep.exprs) == 0 {
		return false
	}
	rootIndex := len(ep.exprs) - 1
	root := ep.exprs[rootIndex]
	localOffset := -1
	globalOffset := -1
	addrOffset := 0
	if root.kind == rtgExprIdent {
		localOffset = rtgFindLocalOffset(g, root.nameStart, root.nameEnd)
		if localOffset >= 0 {
			rtgAsmLoadRaxStack(&g.asm, localOffset)
		} else {
			globalOffset = rtgFindGlobalOffset(g, root.nameStart, root.nameEnd)
			if globalOffset < 0 {
				return false
			}
			rtgAsmLoadRaxBss(&g.asm, globalOffset)
		}
	} else if root.kind == rtgExprSelector {
		if !rtgEmitSelectorAddressRdx(g, ep, root) {
			return false
		}
		addrOffset = rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
		rtgAsmStoreRdxStack(&g.asm, addrOffset)
		rtgAsmLoadRaxMemRdxDisp(&g.asm, 0)
	} else if root.kind == rtgExprIndex {
		if !rtgEmitIndexAddressRax(g, ep, root) {
			return false
		}
		addrOffset = rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
		rtgAsmStoreRaxStack(&g.asm, addrOffset)
		rtgAsmMovRdxRax(&g.asm)
		rtgAsmLoadRaxMemRdxDisp(&g.asm, 0)
	} else if root.kind == rtgExprUnary && rtgTokTextIs(p, root.tok, "*") {
		if !rtgEmitIntExpr(g, ep, root.left) {
			return false
		}
		addrOffset = rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
		rtgAsmStoreRaxStack(&g.asm, addrOffset)
		rtgAsmMovRdxRax(&g.asm)
		rtgAsmLoadRaxMemRdxDisp(&g.asm, 0)
	} else {
		return false
	}
	rtgAsmPushRax(&g.asm)
	rtgAsmMovRaxImm(&g.asm, 1)
	rtgAsmPopRcx(&g.asm)
	if rtgTokTextIs(p, opTok, "++") {
		rtgAsmAddRaxRcx(&g.asm)
	} else {
		rtgAsmSubLeftRcxRightRax(&g.asm)
	}
	if localOffset >= 0 {
		rtgAsmStoreRaxStack(&g.asm, localOffset)
		return true
	}
	if globalOffset >= 0 {
		rtgAsmStoreRaxBss(&g.asm, globalOffset)
		return true
	}
	rtgAsmLoadRdxStack(&g.asm, addrOffset)
	rtgAsmStoreRaxMemRdxDisp(&g.asm, 0)
	return true
}

func rtgEmitLinearPrintStmt(g *rtgLinearGen, stmt rtgStmt) bool {
	p := g.prog
	a := &g.asm
	if !rtgTokTextIs(p, stmt.exprStart, "print") {
		return false
	}
	ep := rtgParseExpression(p, stmt.exprStart, stmt.exprEnd)
	if !ep.ok || len(ep.exprs) == 0 {
		return false
	}
	root := ep.exprs[len(ep.exprs)-1]
	if root.kind != rtgExprCall || root.argCount != 1 || !rtgExprIsIdentText(p, ep, root.left, "print") {
		return false
	}
	if !rtgEmitStringValueRegs(g, ep, ep.args[root.firstArg]) {
		return false
	}
	rtgAsmMovRdiImm(a, 1)
	rtgAsmMovRsiRax(a)
	rtgAsmMovRaxImm(a, 1)
	rtgAsmSyscall(a)
	return true
}

func rtgLinearInitGlobals(g *rtgLinearGen) bool {
	for i := 0; i < len(g.meta.globals); i++ {
		s := g.meta.globals[i]
		if s.kind != rtgTokVar {
			continue
		}
		off := g.asm.bssSize
		g.globals = append(g.globals, rtgGlobalInfo{nameStart: s.nameStart, nameEnd: s.nameEnd, offset: off})
		size := rtgTypeSize(&g.meta, s.typ)
		if size < 8 {
			size = 8
		}
		g.asm.bssSize += rtgAlignTo8(size)
		if s.initStart < s.initEnd {
			ep := rtgParseExpression(g.prog, s.initStart, s.initEnd)
			if !ep.ok || len(ep.exprs) == 0 {
				return false
			}
			rootIndex := len(ep.exprs) - 1
			if rtgTypeIsString(&g.meta, s.typ) {
				if !rtgEmitStringValueRegs(g, ep, rootIndex) {
					return false
				}
				rtgAsmPushRdx(&g.asm)
				rtgAsmStoreRaxBss(&g.asm, off)
				rtgAsmPopRax(&g.asm)
				rtgAsmStoreRaxBss(&g.asm, off+8)
				continue
			}
			if rtgTypeIsSlice(&g.meta, s.typ) {
				root := ep.exprs[rootIndex]
				if root.kind != rtgExprComposite {
					return false
				}
				if !rtgEmitSliceLiteralRegs(g, ep, root, s.typ) {
					return false
				}
				rtgAsmPushRcx(&g.asm)
				rtgAsmPushRdx(&g.asm)
				rtgAsmStoreRaxBss(&g.asm, off)
				rtgAsmPopRax(&g.asm)
				rtgAsmStoreRaxBss(&g.asm, off+8)
				rtgAsmPopRax(&g.asm)
				rtgAsmStoreRaxBss(&g.asm, off+16)
				continue
			}
			constResult := rtgEvalConstExpr(g, ep, rootIndex)
			if !constResult.ok {
				return false
			}
			rtgAsmMovRaxImm(&g.asm, constResult.value)
			rtgAsmStoreRaxBss(&g.asm, off)
		} else if rtgTypeIsSlice(&g.meta, s.typ) {
			rtgEmitInitEmptySliceBss(g, s.typ, off)
		}
	}
	return true
}

func rtgEmitInitEmptySliceBss(g *rtgLinearGen, sliceType int, off int) {
	t := rtgResolveType(&g.meta, sliceType)
	elemSize := rtgTypeSize(&g.meta, t.elem)
	if elemSize < 1 {
		elemSize = 8
	}
	backingSize := 32768
	backingOff := g.asm.bssSize
	g.asm.bssSize += backingSize
	rtgAsmMovRaxBssAddr(&g.asm, backingOff)
	rtgAsmStoreRaxBss(&g.asm, off)
	rtgAsmMovRaxImm(&g.asm, 0)
	rtgAsmStoreRaxBss(&g.asm, off+8)
	rtgAsmMovRaxImm(&g.asm, backingSize/elemSize)
	rtgAsmStoreRaxBss(&g.asm, off+16)
}

func rtgEmitLinearAssign(g *rtgLinearGen, stmt rtgStmt) bool {
	p := g.prog
	nameStart := stmt.nameStart
	nameEnd := stmt.nameEnd
	if (stmt.kind == rtgStmtVar || rtgTokTextIs(p, stmt.startTok, "var")) && rtgTokIsKind(p, stmt.startTok+1, rtgTokIdent) {
		nameStart = p.toks[stmt.startTok+1].start
		nameEnd = p.toks[stmt.startTok+1].end
	} else if rtgTokIsKind(p, stmt.startTok, rtgTokIdent) {
		nameStart = p.toks[stmt.startTok].start
		nameEnd = p.toks[stmt.startTok].end
	}
	assignTok := rtgFindAssignmentToken(p, stmt.startTok, stmt.endTok)
	if assignTok > stmt.startTok && (rtgTokTextIs(p, assignTok, "+=") || rtgTokTextIs(p, assignTok, "-=") || rtgTokTextIs(p, assignTok, "*=") || rtgTokTextIs(p, assignTok, "/=") || rtgTokTextIs(p, assignTok, "%=")) {
		lhs := rtgParseExpression(p, stmt.startTok, assignTok)
		if lhs.ok && len(lhs.exprs) > 0 {
			lhsRoot := lhs.exprs[len(lhs.exprs)-1]
			if lhsRoot.kind == rtgExprIndex {
				baseEnd := rtgFindTokenTextInRange(p, stmt.startTok, assignTok, "[")
				if baseEnd <= stmt.startTok {
					return false
				}
				baseEp := rtgParseExpression(p, stmt.startTok, baseEnd)
				if !baseEp.ok || len(baseEp.exprs) == 0 {
					return false
				}
				baseIndex := len(baseEp.exprs) - 1
				leftType := rtgInferParsedExprType(g, baseEp, baseIndex)
				sliceType := rtgResolveType(&g.meta, leftType)
				elemType := rtgResolveType(&g.meta, sliceType.elem)
				if sliceType.kind != rtgTypeSlice || (elemType.kind != rtgTypeInt && elemType.kind != rtgTypeByte && elemType.kind != rtgTypeBool) {
					return false
				}
				indexOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
				ptrOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
				if !rtgEmitIntExpr(g, lhs, lhsRoot.right) {
					return false
				}
				rtgAsmStoreRaxStack(&g.asm, indexOffset)
				if !rtgEmitSliceBasePtrLenTokens(g, p, stmt.startTok, baseEnd, baseEp, baseIndex) {
					return false
				}
				rtgAsmStoreRaxStack(&g.asm, ptrOffset)
				rtgAsmLoadRaxStack(&g.asm, ptrOffset)
				rtgAsmMovRdxRax(&g.asm)
				rtgAsmLoadRcxStack(&g.asm, indexOffset)
				if elemType.kind == rtgTypeByte || elemType.kind == rtgTypeBool {
					rtgAsmLoadByteRaxIndexRcx(&g.asm)
				} else {
					rtgAsmLoadQwordRaxIndexRcx8(&g.asm)
				}
				rtgAsmPushRax(&g.asm)
				rhs := rtgParseExpression(p, assignTok+1, stmt.endTok)
				if !rhs.ok || len(rhs.exprs) == 0 {
					return false
				}
				rhsIndex := len(rhs.exprs) - 1
				if !rtgEmitIntExpr(g, rhs, rhsIndex) {
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
				rtgAsmLoadRdxStack(&g.asm, ptrOffset)
				rtgAsmLoadRcxStack(&g.asm, indexOffset)
				if elemType.kind == rtgTypeByte || elemType.kind == rtgTypeBool {
					rtgAsmStoreAlMemRdxRcx1(&g.asm)
				} else {
					rtgAsmStoreRaxMemRdxRcx8(&g.asm)
				}
				return true
			}
			if lhsRoot.kind == rtgExprSelector {
				if !rtgEmitSelectorAddressRdx(g, lhs, lhsRoot) {
					return false
				}
				addrOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
				rtgAsmStoreRdxStack(&g.asm, addrOffset)
				rhs := rtgParseExpression(p, assignTok+1, stmt.endTok)
				if !rhs.ok || len(rhs.exprs) == 0 {
					return false
				}
				rtgAsmLoadRdxStack(&g.asm, addrOffset)
				rtgAsmLoadRaxMemRdxDisp(&g.asm, 0)
				rtgAsmPushRax(&g.asm)
				rhsIndex := len(rhs.exprs) - 1
				if !rtgEmitIntExpr(g, rhs, rhsIndex) {
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
				rtgAsmLoadRdxStack(&g.asm, addrOffset)
				rtgAsmStoreRaxMemRdxDisp(&g.asm, 0)
				return true
			}
		}
	}
	if assignTok > stmt.startTok && rtgTokTextIs(p, assignTok, "=") {
		lhs := rtgParseExpression(p, stmt.startTok, assignTok)
		if lhs.ok && len(lhs.exprs) > 0 {
			lhsIndex := len(lhs.exprs) - 1
			lhsRoot := lhs.exprs[lhsIndex]
			lhsType := rtgInferParsedExprType(g, lhs, lhsIndex)
			if lhsRoot.kind == rtgExprIndex {
				baseEnd := rtgFindTokenTextInRange(p, stmt.startTok, assignTok, "[")
				if baseEnd <= stmt.startTok {
					return false
				}
				baseEp := rtgParseExpression(p, stmt.startTok, baseEnd)
				if !baseEp.ok || len(baseEp.exprs) == 0 {
					return false
				}
				baseIndex := len(baseEp.exprs) - 1
				leftType := rtgInferParsedExprType(g, baseEp, baseIndex)
				sliceType := rtgResolveType(&g.meta, leftType)
				elemType := rtgResolveType(&g.meta, sliceType.elem)
				if sliceType.kind != rtgTypeSlice {
					return false
				}
				rhs := rtgParseExpression(p, assignTok+1, stmt.endTok)
				if !rhs.ok || len(rhs.exprs) == 0 {
					return false
				}
				rhsIndex := len(rhs.exprs) - 1
				if elemType.kind == rtgTypeInt || elemType.kind == rtgTypeByte || elemType.kind == rtgTypeBool {
					if !rtgEmitIntExpr(g, lhs, lhsRoot.right) {
						return false
					}
					rtgAsmPushRax(&g.asm)
					if !rtgEmitSliceBasePtrLenTokens(g, p, stmt.startTok, baseEnd, baseEp, baseIndex) {
						return false
					}
					rtgAsmPushRax(&g.asm)
					if !rtgEmitIntExpr(g, rhs, rhsIndex) {
						return false
					}
					rtgAsmPopRdx(&g.asm)
					rtgAsmPopRcx(&g.asm)
					if elemType.kind == rtgTypeByte || elemType.kind == rtgTypeBool {
						rtgAsmStoreAlMemRdxRcx1(&g.asm)
					} else {
						rtgAsmStoreRaxMemRdxRcx8(&g.asm)
					}
					return true
				}
				if elemType.kind == rtgTypeString {
					indexOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
					ptrOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
					if !rtgEmitIntExpr(g, lhs, lhsRoot.right) {
						return false
					}
					rtgAsmStoreRaxStack(&g.asm, indexOffset)
					if !rtgEmitSliceBasePtrLenTokens(g, p, stmt.startTok, baseEnd, baseEp, baseIndex) {
						return false
					}
					rtgAsmStoreRaxStack(&g.asm, ptrOffset)
					if !rtgEmitStringValueRegs(g, rhs, rhsIndex) {
						return false
					}
					rtgAsmPushRdx(&g.asm)
					rtgAsmPushRax(&g.asm)
					rtgAsmLoadRaxStack(&g.asm, ptrOffset)
					rtgAsmMovRdxRax(&g.asm)
					rtgAsmLoadRcxStack(&g.asm, indexOffset)
					rtgAsmShlRcxImm(&g.asm, 4)
					rtgAsmAddRdxRcx(&g.asm)
					rtgAsmPopRax(&g.asm)
					rtgAsmStoreRaxMemRdxDisp(&g.asm, 0)
					rtgAsmPopRax(&g.asm)
					rtgAsmStoreRaxMemRdxDisp(&g.asm, 8)
					return true
				}
				if elemType.kind == rtgTypeStruct {
					indexOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
					ptrOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
					destOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
					if !rtgEmitIntExpr(g, lhs, lhsRoot.right) {
						return false
					}
					rtgAsmStoreRaxStack(&g.asm, indexOffset)
					if !rtgEmitSliceBasePtrLenTokens(g, p, stmt.startTok, baseEnd, baseEp, baseIndex) {
						return false
					}
					rtgAsmStoreRaxStack(&g.asm, ptrOffset)
					rtgAsmLoadRaxStack(&g.asm, ptrOffset)
					rtgAsmMovRdxRax(&g.asm)
					rtgAsmLoadRcxStack(&g.asm, indexOffset)
					elemSize := rtgTypeSize(&g.meta, sliceType.elem)
					rtgAsmImulRcxImm(&g.asm, elemSize)
					rtgAsmAddRdxRcx(&g.asm)
					rtgAsmStoreRdxStack(&g.asm, destOffset)
					if !rtgEmitCompositeFieldToMem(g, rhs, rhsIndex, sliceType.elem, destOffset, 0) {
						return false
					}
					return true
				}
				return false
			}
			if lhsRoot.kind == rtgExprSelector && rtgTypeIsSlice(&g.meta, lhsType) {
				if !rtgEmitSelectorAddressRdx(g, lhs, lhsRoot) {
					return false
				}
				addrOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
				rtgAsmStoreRdxStack(&g.asm, addrOffset)
				rhs := rtgParseExpression(p, assignTok+1, stmt.endTok)
				if !rhs.ok || len(rhs.exprs) == 0 {
					return false
				}
				if rtgEmitAppendAssignGeneral(g, stmt, rhs) {
					return true
				}
				rhsIndex := len(rhs.exprs) - 1
				rhsRoot := rhs.exprs[rhsIndex]
				if rhsRoot.kind == rtgExprSelector {
					if !rtgEmitSelectorAddressRdx(g, rhs, rhsRoot) {
						return false
					}
					rtgAsmLoadRaxMemRdxDisp(&g.asm, 16)
					rtgAsmPushRax(&g.asm)
					rtgAsmLoadRaxMemRdxDisp(&g.asm, 8)
					rtgAsmPushRax(&g.asm)
					rtgAsmLoadRaxMemRdxDisp(&g.asm, 0)
					rtgAsmPushRax(&g.asm)
				} else if rhsRoot.kind == rtgExprIdent {
					localIndex := rtgFindLocalIndex(g, rhsRoot.nameStart, rhsRoot.nameEnd)
					if localIndex < 0 || !rtgTypeIsSlice(&g.meta, g.locals[localIndex].typ) {
						return false
					}
					rtgAsmLoadRaxStack(&g.asm, g.locals[localIndex].offset-16)
					rtgAsmPushRax(&g.asm)
					rtgAsmLoadRaxStack(&g.asm, g.locals[localIndex].offset-8)
					rtgAsmPushRax(&g.asm)
					rtgAsmLoadRaxStack(&g.asm, g.locals[localIndex].offset)
					rtgAsmPushRax(&g.asm)
				} else if rhsRoot.kind == rtgExprCall {
					fnIndex := rtgFuncInfoFromCall(g, rhs, rhsRoot.left)
					if fnIndex < 0 || !rtgTypeIsSlice(&g.meta, g.meta.funcs[fnIndex].resultType) {
						return false
					}
					if !rtgEmitUserCall(g, rhs, rhsRoot) {
						return false
					}
					rtgAsmPushRcx(&g.asm)
					rtgAsmPushRdx(&g.asm)
					rtgAsmPushRax(&g.asm)
				} else {
					return false
				}
				rtgAsmLoadRdxStack(&g.asm, addrOffset)
				rtgAsmPopRax(&g.asm)
				rtgAsmStoreRaxMemRdxDisp(&g.asm, 0)
				rtgAsmPopRax(&g.asm)
				rtgAsmStoreRaxMemRdxDisp(&g.asm, 8)
				rtgAsmPopRax(&g.asm)
				rtgAsmStoreRaxMemRdxDisp(&g.asm, 16)
				return true
			}
			if lhsRoot.kind == rtgExprSelector && rtgTypeIsStruct(&g.meta, lhsType) {
				if !rtgEmitSelectorAddressRdx(g, lhs, lhsRoot) {
					return false
				}
				addrOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
				rtgAsmStoreRdxStack(&g.asm, addrOffset)
				rhs := rtgParseExpression(p, assignTok+1, stmt.endTok)
				if !rhs.ok || len(rhs.exprs) == 0 {
					return false
				}
				rhsIndex := len(rhs.exprs) - 1
				rhsRoot := rhs.exprs[rhsIndex]
				size := rtgTypeSize(&g.meta, lhsType)
				if rhsRoot.kind == rtgExprIdent {
					localIndex := rtgFindLocalIndex(g, rhsRoot.nameStart, rhsRoot.nameEnd)
					if localIndex < 0 || rtgTypeSize(&g.meta, g.locals[localIndex].typ) != size {
						return false
					}
					rtgAsmLoadRdxStack(&g.asm, addrOffset)
					for at := 0; at < size; at += 8 {
						rtgAsmLoadRaxStack(&g.asm, g.locals[localIndex].offset-at)
						rtgAsmStoreRaxMemRdxDisp(&g.asm, at)
					}
					return true
				}
				if rhsRoot.kind == rtgExprSelector {
					fieldType := rtgInferParsedExprType(g, rhs, rhsIndex)
					if !rtgTypeIsStruct(&g.meta, fieldType) || rtgTypeSize(&g.meta, fieldType) != size {
						return false
					}
					if !rtgEmitSelectorAddressRdx(g, rhs, rhsRoot) {
						return false
					}
					for at := size - 8; at >= 0; at -= 8 {
						rtgAsmLoadRaxMemRdxDisp(&g.asm, at)
						rtgAsmPushRax(&g.asm)
					}
					rtgAsmLoadRdxStack(&g.asm, addrOffset)
					for at := 0; at < size; at += 8 {
						rtgAsmPopRax(&g.asm)
						rtgAsmStoreRaxMemRdxDisp(&g.asm, at)
					}
					return true
				}
				return false
			}
			if lhsRoot.kind == rtgExprSelector {
				if !rtgEmitSelectorAddressRdx(g, lhs, lhsRoot) {
					return false
				}
				rtgAsmPushRdx(&g.asm)
				rhs := rtgParseExpression(p, assignTok+1, stmt.endTok)
				if !rhs.ok || len(rhs.exprs) == 0 {
					return false
				}
				rhsIndex := len(rhs.exprs) - 1
				if !rtgEmitIntExpr(g, rhs, rhsIndex) {
					return false
				}
				rtgAsmPopRdx(&g.asm)
				rtgAsmStoreRaxMemRdxDisp(&g.asm, 0)
				return true
			}
		}
	}
	if nameEnd <= nameStart {
		if rtgTokTextIs(p, stmt.startTok, "*") && assignTok > stmt.startTok && (rtgTokTextIs(p, assignTok, "+=") || rtgTokTextIs(p, assignTok, "-=") || rtgTokTextIs(p, assignTok, "*=") || rtgTokTextIs(p, assignTok, "/=") || rtgTokTextIs(p, assignTok, "%=")) {
			left := rtgParseExpression(p, stmt.startTok+1, assignTok)
			if !left.ok || len(left.exprs) == 0 {
				return false
			}
			leftIndex := len(left.exprs) - 1
			if !rtgEmitIntExpr(g, left, leftIndex) {
				return false
			}
			rtgAsmPushRax(&g.asm)
			rtgAsmMovRdxRax(&g.asm)
			rtgAsmLoadRaxMemRdxDisp(&g.asm, 0)
			rtgAsmPushRax(&g.asm)
			right := rtgParseExpression(p, assignTok+1, stmt.endTok)
			if !right.ok || len(right.exprs) == 0 {
				return false
			}
			rightIndex := len(right.exprs) - 1
			if !rtgEmitIntExpr(g, right, rightIndex) {
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
			} else if rtgTokTextIs(p, assignTok, "%=") {
				rtgAsmDivLeftRcxRightRax(&g.asm, true)
			} else {
				return false
			}
			rtgAsmPopRdx(&g.asm)
			rtgAsmStoreRaxMemRdxDisp(&g.asm, 0)
			return true
		}
		if rtgTokTextIs(p, stmt.startTok, "*") && assignTok > stmt.startTok && rtgTokTextIs(p, assignTok, "=") {
			left := rtgParseExpression(p, stmt.startTok+1, assignTok)
			if !left.ok || len(left.exprs) == 0 {
				return false
			}
			leftIndex := len(left.exprs) - 1
			if !rtgEmitIntExpr(g, left, leftIndex) {
				return false
			}
			rtgAsmPushRax(&g.asm)
			right := rtgParseExpression(p, assignTok+1, stmt.endTok)
			if !right.ok || len(right.exprs) == 0 {
				return false
			}
			rightIndex := len(right.exprs) - 1
			if !rtgEmitIntExpr(g, right, rightIndex) {
				return false
			}
			rtgAsmPopRdx(&g.asm)
			rtgAsmStoreRaxMemRdxDisp(&g.asm, 0)
			return true
		}
		return false
	}
	offset := rtgFindLocalOffset(g, nameStart, nameEnd)
	if stmt.kind == rtgStmtShort && offset >= 0 && assignTok > stmt.startTok {
		localIndex := rtgFindLocalIndex(g, nameStart, nameEnd)
		inferredType := rtgInferExprType(g, assignTok+1, stmt.endTok)
		if localIndex >= 0 && inferredType != 0 {
			oldType := rtgResolveType(&g.meta, g.locals[localIndex].typ)
			newType := rtgResolveType(&g.meta, inferredType)
			if oldType.kind != newType.kind || rtgTypeSize(&g.meta, g.locals[localIndex].typ) != rtgTypeSize(&g.meta, inferredType) {
				offset = -1
			}
		}
	}
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
		if stmt.kind == rtgStmtAssign && !rtgTokTextIs(p, stmt.startTok, "var") {
			globalOffset = rtgFindGlobalOffset(g, nameStart, nameEnd)
			if globalOffset < 0 {
				return false
			}
		} else {
			localType := rtgTypeInt
			if stmt.kind == rtgStmtVar || rtgTokTextIs(p, stmt.startTok, "var") {
				typeEnd := assignTok
				if assignTok <= stmt.startTok {
					typeEnd = stmt.endTok
				}
				if stmt.startTok+2 < typeEnd {
					typeResult := rtgParseType(&g.meta, g.prog, stmt.startTok+2, typeEnd)
					if typeResult.typ != 0 {
						localType = typeResult.typ
					}
				}
			}
			if stmt.kind == rtgStmtShort {
				inferredType := rtgInferExprType(g, assignTok+1, stmt.endTok)
				if assignTok+2 < stmt.endTok && rtgTokIsKind(p, assignTok+1, rtgTokIdent) && rtgTokTextIs(p, assignTok+2, "(") {
					fnIndex := rtgFindFuncInfoByRange(g, p.toks[assignTok+1].start, p.toks[assignTok+1].end)
					if fnIndex >= 0 {
						inferredType = g.meta.funcs[fnIndex].resultType
					}
				}
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
	rootIndex := len(ep.exprs) - 1
	if rtgTokTextIs(p, assignTok, "+=") || rtgTokTextIs(p, assignTok, "-=") || rtgTokTextIs(p, assignTok, "*=") || rtgTokTextIs(p, assignTok, "/=") || rtgTokTextIs(p, assignTok, "%=") {
		if globalOffset >= 0 {
			rtgAsmLoadRaxBss(&g.asm, globalOffset)
		} else {
			rtgAsmLoadRaxStack(&g.asm, offset)
		}
		rtgAsmPushRax(&g.asm)
		if !rtgEmitIntExpr(g, ep, rootIndex) {
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
	if globalOffset < 0 && rtgEmitTypedAssign(g, ep, rootIndex, offset) {
		return true
	}
	if !rtgEmitIntExpr(g, ep, rootIndex) {
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
	rootIndex := len(ep.exprs) - 1
	return rtgInferParsedExprType(g, ep, rootIndex)
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
		constStringTok := rtgFindConstStringToken(g, e.nameStart, e.nameEnd)
		if constStringTok >= 0 {
			return rtgTypeString
		}
		return rtgTypeInt
	}
	if e.kind == rtgExprCall {
		if rtgExprIsIdentText(g.prog, ep, e.left, "append") && e.argCount == 2 {
			return rtgInferParsedExprType(g, ep, ep.args[e.firstArg])
		}
		if rtgExprIsIdentText(g.prog, ep, e.left, "[]byte") && e.argCount == 1 {
			return rtgByteSliceType(g)
		}
		if e.argCount == 2 || e.argCount == 3 {
			if rtgExprIsIdentText(g.prog, ep, e.left, "make") {
				return rtgTypeFromExpr(g, ep, ep.args[e.firstArg])
			}
		}
		if rtgExprIsIdentText(g.prog, ep, e.left, "rtgEnv") && e.argCount == 0 {
			fnIndex := rtgFuncInfoFromCall(g, ep, e.left)
			if fnIndex >= 0 {
				return g.meta.funcs[fnIndex].resultType
			}
		}
		if rtgExprIsIdentText(g.prog, ep, e.left, "rtgParseProgram") {
			named := rtgFindTypeByText(g, "rtgProgram")
			if named > 0 {
				return named
			}
		}
		if rtgExprIsIdentText(g.prog, ep, e.left, "int") || rtgExprIsIdentText(g.prog, ep, e.left, "int64") || rtgExprIsIdentText(g.prog, ep, e.left, "byte") || rtgExprIsIdentText(g.prog, ep, e.left, "len") || rtgExprIsIdentText(g.prog, ep, e.left, "open") || rtgExprIsIdentText(g.prog, ep, e.left, "close") || rtgExprIsIdentText(g.prog, ep, e.left, "read") || rtgExprIsIdentText(g.prog, ep, e.left, "write") || rtgExprIsIdentText(g.prog, ep, e.left, "chmod") {
			return rtgTypeInt
		}
		fnIndex := rtgFuncInfoFromCall(g, ep, e.left)
		if fnIndex >= 0 {
			if rtgBytesEqualText(g.prog.src, g.meta.funcs[fnIndex].nameStart, g.meta.funcs[fnIndex].nameEnd, "rtgParseProgram") {
				named := rtgFindTypeByText(g, "rtgProgram")
				if named > 0 {
					return named
				}
			}
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
		return rtgCompositeExprType(g, ep, e)
	}
	if e.kind == rtgExprUnary {
		if rtgTokTextIs(g.prog, e.tok, "&") {
			elemType := rtgInferParsedExprType(g, ep, e.left)
			if elemType == 0 {
				return 0
			}
			info := rtgTypeInfo{kind: rtgTypePointer, elem: elemType, size: 8}
			return rtgAddType(&g.meta, info)
		}
		if rtgTokTextIs(g.prog, e.tok, "*") {
			innerType := rtgInferParsedExprType(g, ep, e.left)
			inner := rtgResolveType(&g.meta, innerType)
			if inner.kind == rtgTypePointer {
				return inner.elem
			}
		}
	}
	return rtgTypeInt
}

func rtgCompositeExprType(g *rtgLinearGen, ep rtgExprParse, e rtgExpr) int {
	return rtgTypeFromExprValue(g, e)
}

func rtgTypeFromExpr(g *rtgLinearGen, ep rtgExprParse, idx int) int {
	if idx < 0 || idx >= len(ep.exprs) {
		return 0
	}
	e := ep.exprs[idx]
	return rtgTypeFromExprValue(g, e)
}

func rtgTypeFromExprValue(g *rtgLinearGen, e rtgExpr) int {
	if e.tok < 0 || e.tok >= len(g.prog.toks) {
		return 0
	}
	endTok := e.tok
	for endTok < len(g.prog.toks) && g.prog.toks[endTok].end <= e.nameEnd {
		endTok++
	}
	typeResult := rtgParseType(&g.meta, g.prog, e.tok, endTok)
	return typeResult.typ
}

func rtgFindTypeByText(g *rtgLinearGen, name string) int {
	for i := 0; i < len(g.meta.types); i++ {
		t := g.meta.types[i]
		if t.nameEnd > t.nameStart && rtgBytesEqualText(g.prog.src, t.nameStart, t.nameEnd, name) {
			return i
		}
	}
	return 0
}

func rtgByteSliceType(g *rtgLinearGen) int {
	for i := 0; i < len(g.meta.types); i++ {
		t := g.meta.types[i]
		if t.kind == rtgTypeSlice {
			elem := rtgResolveType(&g.meta, t.elem)
			if elem.kind == rtgTypeByte {
				return i
			}
		}
	}
	info := rtgTypeInfo{kind: rtgTypeSlice, elem: rtgTypeByte, size: 24}
	typ := rtgAddType(&g.meta, info)
	return typ
}

func rtgFindFuncInfoByRange(g *rtgLinearGen, nameStart int, nameEnd int) int {
	for i := 0; i < len(g.meta.funcs); i++ {
		f := g.meta.funcs[i]
		if rtgBytesEqualRange(g.prog.src, f.nameStart, f.nameEnd, nameStart, nameEnd) {
			return i
		}
	}
	return -1
}

func rtgLocalTypeAtOffset(g *rtgLinearGen, offset int) int {
	for i := 0; i < len(g.locals); i++ {
		if g.locals[i].offset == offset {
			return g.locals[i].typ
		}
	}
	for i := 0; i < len(g.locals); i++ {
		t := rtgResolveType(&g.meta, g.locals[i].typ)
		if t.kind == rtgTypeStruct {
			for j := 0; j < t.count; j++ {
				field := g.meta.fields[t.first+j]
				if g.locals[i].offset-field.offset == offset {
					return field.typ
				}
			}
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
		if e.kind == rtgExprIdent {
			localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
			if localIndex < 0 || rtgTypeSize(&g.meta, g.locals[localIndex].typ) != rtgTypeSize(&g.meta, destType) {
				return false
			}
			size := rtgTypeSize(&g.meta, destType)
			for at := 0; at < size; at += 8 {
				rtgAsmLoadRaxStack(&g.asm, g.locals[localIndex].offset-at)
				rtgAsmStoreRaxStack(&g.asm, offset-at)
			}
			return true
		}
		if e.kind == rtgExprCall {
			return rtgEmitStructCallToLocal(g, ep, e, destType, offset)
		}
		if e.kind == rtgExprIndex {
			return rtgEmitIndexedStructToLocal(g, ep, e, destType, offset)
		}
		if e.kind == rtgExprSelector {
			fieldType := rtgInferParsedExprType(g, ep, idx)
			if !rtgTypeIsStruct(&g.meta, fieldType) || rtgTypeSize(&g.meta, fieldType) != rtgTypeSize(&g.meta, destType) {
				return false
			}
			if !rtgEmitSelectorAddressRdx(g, ep, e) {
				return false
			}
			size := rtgTypeSize(&g.meta, destType)
			for at := 0; at < size; at += 8 {
				rtgAsmLoadRaxMemRdxDisp(&g.asm, at)
				rtgAsmStoreRaxStack(&g.asm, offset-at)
			}
			return true
		}
		if e.kind == rtgExprComposite {
			rtgZeroLocalAtOffset(g, offset)
			for i := 0; i < e.argCount; i++ {
				field := ep.fields[e.firstArg+i]
				fieldOffset := rtgStructFieldOffset(g, destType, field.nameStart, field.nameEnd)
				if fieldOffset < 0 {
					return false
				}
				fieldType := rtgStructFieldType(g, destType, field.nameStart, field.nameEnd)
				if fieldType == 0 {
					return false
				}
				if !rtgEmitCompositeFieldToStack(g, ep, field.expr, fieldType, offset-fieldOffset) {
					return false
				}
			}
			return true
		}
		return false
	}
	if destResolved.kind == rtgTypeString {
		if !rtgEmitStringValueRegs(g, ep, idx) {
			return false
		}
		rtgAsmStoreRaxStack(&g.asm, offset)
		rtgAsmStoreRdxStack(&g.asm, offset-8)
		return true
	}
	if !rtgTypeIsSlice(&g.meta, destType) {
		return false
	}
	if e.kind == rtgExprComposite {
		if !rtgEmitSliceLiteralRegs(g, ep, e, destType) {
			return false
		}
		rtgAsmStoreRaxStack(&g.asm, offset)
		rtgAsmStoreRdxStack(&g.asm, offset-8)
		rtgAsmStoreRcxStack(&g.asm, offset-16)
		return true
	}
	if e.kind == rtgExprCall {
		if e.argCount == 2 || e.argCount == 3 {
			if rtgExprIsIdentText(g.prog, ep, e.left, "make") {
				if !rtgEmitMakeSliceRegs(g, ep, e) {
					return false
				}
				rtgAsmStoreRaxStack(&g.asm, offset)
				rtgAsmStoreRdxStack(&g.asm, offset-8)
				rtgAsmStoreRcxStack(&g.asm, offset-16)
				return true
			}
		}
		if e.argCount == 1 && rtgExprIsIdentText(g.prog, ep, e.left, "[]byte") {
			if !rtgEmitByteSliceConversionRegs(g, ep, e) {
				return false
			}
			rtgAsmStoreRaxStack(&g.asm, offset)
			rtgAsmStoreRdxStack(&g.asm, offset-8)
			rtgAsmStoreRcxStack(&g.asm, offset-16)
			return true
		}
		if e.argCount == 0 && rtgExprIsIdentText(g.prog, ep, e.left, "rtgEnv") {
			return rtgEmitEnvSliceToLocal(g, offset)
		}
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
	if e.kind == rtgExprSelector {
		fieldType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsSlice(&g.meta, fieldType) {
			return false
		}
		if !rtgEmitSelectorAddressRdx(g, ep, e) {
			return false
		}
		rtgAsmLoadRaxMemRdxDisp(&g.asm, 0)
		rtgAsmStoreRaxStack(&g.asm, offset)
		rtgAsmLoadRaxMemRdxDisp(&g.asm, 8)
		rtgAsmStoreRaxStack(&g.asm, offset-8)
		rtgAsmLoadRaxMemRdxDisp(&g.asm, 16)
		rtgAsmStoreRaxStack(&g.asm, offset-16)
		return true
	}
	return false
}

func rtgEmitSliceValueRegs(g *rtgLinearGen, ep rtgExprParse, idx int) bool {
	if idx < 0 || idx >= len(ep.exprs) {
		return false
	}
	e := ep.exprs[idx]
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 {
			globalOffset := rtgFindGlobalOffset(g, e.nameStart, e.nameEnd)
			globalType := rtgFindGlobalType(g, e.nameStart, e.nameEnd)
			if globalOffset < 0 || !rtgTypeIsSlice(&g.meta, globalType) {
				return false
			}
			rtgAsmLoadRaxBss(&g.asm, globalOffset+16)
			rtgAsmPushRax(&g.asm)
			rtgAsmLoadRaxBss(&g.asm, globalOffset+8)
			rtgAsmPushRax(&g.asm)
			rtgAsmLoadRaxBss(&g.asm, globalOffset)
			rtgAsmPopRdx(&g.asm)
			rtgAsmPopRcx(&g.asm)
			return true
		}
		if !rtgTypeIsSlice(&g.meta, g.locals[localIndex].typ) {
			return false
		}
		rtgAsmLoadRaxStack(&g.asm, g.locals[localIndex].offset)
		rtgAsmLoadRdxStack(&g.asm, g.locals[localIndex].offset-8)
		rtgAsmLoadRcxStack(&g.asm, g.locals[localIndex].offset-16)
		return true
	}
	if e.kind == rtgExprSelector {
		valueType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsSlice(&g.meta, valueType) {
			return false
		}
		if !rtgEmitSelectorAddressRdx(g, ep, e) {
			return false
		}
		addrOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
		rtgAsmStoreRdxStack(&g.asm, addrOffset)
		rtgAsmLoadRaxMemRdxDisp(&g.asm, 0)
		rtgAsmPushRax(&g.asm)
		rtgAsmLoadRdxStack(&g.asm, addrOffset)
		rtgAsmLoadRaxMemRdxDisp(&g.asm, 8)
		rtgAsmPushRax(&g.asm)
		rtgAsmLoadRdxStack(&g.asm, addrOffset)
		rtgAsmLoadRaxMemRdxDisp(&g.asm, 16)
		rtgAsmMovRcxRax(&g.asm)
		rtgAsmPopRdx(&g.asm)
		rtgAsmPopRax(&g.asm)
		return true
	}
	if e.kind == rtgExprComposite {
		sliceType := rtgCompositeExprType(g, ep, e)
		if !rtgTypeIsSlice(&g.meta, sliceType) {
			return false
		}
		return rtgEmitSliceLiteralRegs(g, ep, e, sliceType)
	}
	if e.kind == rtgExprCall {
		if e.argCount == 2 && rtgExprIsIdentText(g.prog, ep, e.left, "append") {
			var stmt rtgStmt
			if !rtgEmitAppendAssignGeneral(g, stmt, ep) {
				return false
			}
			return rtgEmitSliceValueRegs(g, ep, ep.args[e.firstArg])
		}
		if e.argCount == 2 || e.argCount == 3 {
			if rtgExprIsIdentText(g.prog, ep, e.left, "make") {
				return rtgEmitMakeSliceRegs(g, ep, e)
			}
		}
		if e.argCount == 1 && rtgExprIsIdentText(g.prog, ep, e.left, "[]byte") {
			return rtgEmitByteSliceConversionRegs(g, ep, e)
		}
		if e.argCount == 0 && rtgExprIsIdentText(g.prog, ep, e.left, "rtgEnv") {
			return rtgEmitEnvSliceRegs(g)
		}
		callType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsSlice(&g.meta, callType) {
			return false
		}
		if !rtgEmitIntExpr(g, ep, idx) {
			return false
		}
		return true
	}
	return false
}

func rtgEmitSliceLiteralRegs(g *rtgLinearGen, ep rtgExprParse, e rtgExpr, sliceType int) bool {
	t := rtgResolveType(&g.meta, sliceType)
	if t.kind != rtgTypeSlice {
		return false
	}
	elemSize := rtgTypeSize(&g.meta, t.elem)
	if elemSize < 1 {
		elemSize = 8
	}
	backingSize := 32768
	needSize := e.argCount * elemSize
	if needSize > backingSize {
		backingSize = rtgAlignTo8(needSize)
	}
	if backingSize < elemSize {
		backingSize = elemSize
	}
	backingOff := g.asm.bssSize
	g.asm.bssSize += backingSize
	if !rtgEmitSliceLiteralBacking(g, ep, e, sliceType, backingOff) {
		return false
	}
	capacity := backingSize / elemSize
	rtgAsmMovRaxImm(&g.asm, capacity)
	rtgAsmPushRax(&g.asm)
	rtgAsmMovRaxBssAddr(&g.asm, backingOff)
	rtgAsmMovRdxImm(&g.asm, e.argCount)
	rtgAsmPopRcx(&g.asm)
	return true
}

func rtgEmitSliceLiteralBacking(g *rtgLinearGen, ep rtgExprParse, e rtgExpr, sliceType int, backingOff int) bool {
	t := rtgResolveType(&g.meta, sliceType)
	if t.kind != rtgTypeSlice {
		return false
	}
	elemType := t.elem
	elemResolved := rtgResolveType(&g.meta, elemType)
	elemSize := rtgTypeSize(&g.meta, elemType)
	if elemSize < 1 {
		elemSize = 8
	}
	for i := 0; i < e.argCount; i++ {
		field := ep.fields[e.firstArg+i]
		if field.nameEnd > field.nameStart {
			return false
		}
		disp := i * elemSize
		if elemResolved.kind == rtgTypeString {
			if !rtgEmitStringValueRegs(g, ep, field.expr) {
				return false
			}
			rtgAsmPushRdx(&g.asm)
			rtgAsmPushRax(&g.asm)
			rtgAsmMovRaxBssAddr(&g.asm, backingOff)
			rtgAsmMovRdxRax(&g.asm)
			rtgAsmPopRax(&g.asm)
			rtgAsmStoreRaxMemRdxDisp(&g.asm, disp)
			rtgAsmPopRax(&g.asm)
			rtgAsmStoreRaxMemRdxDisp(&g.asm, disp+8)
			continue
		}
		if elemResolved.kind == rtgTypeStruct {
			addrOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
			rtgAsmMovRaxBssAddr(&g.asm, backingOff)
			rtgAsmMovRdxRax(&g.asm)
			if disp != 0 {
				rtgAsmAddRdxImm(&g.asm, disp)
			}
			rtgAsmStoreRdxStack(&g.asm, addrOffset)
			if !rtgEmitCompositeFieldToMem(g, ep, field.expr, elemType, addrOffset, 0) {
				return false
			}
			continue
		}
		if elemResolved.kind != rtgTypeInt && elemResolved.kind != rtgTypeInt64 && elemResolved.kind != rtgTypeByte && elemResolved.kind != rtgTypeBool {
			return false
		}
		if !rtgEmitIntExpr(g, ep, field.expr) {
			return false
		}
		rtgAsmPushRax(&g.asm)
		rtgAsmMovRaxBssAddr(&g.asm, backingOff)
		rtgAsmMovRdxRax(&g.asm)
		rtgAsmPopRax(&g.asm)
		if elemSize == 1 {
			rtgAsmStoreAlMemRdxDisp(&g.asm, disp)
		} else {
			rtgAsmStoreRaxMemRdxDisp(&g.asm, disp)
		}
	}
	return true
}

func rtgEmitMakeSliceRegs(g *rtgLinearGen, ep rtgExprParse, e rtgExpr) bool {
	if e.argCount != 2 && e.argCount != 3 {
		return false
	}
	sliceType := rtgTypeFromExpr(g, ep, ep.args[e.firstArg])
	t := rtgResolveType(&g.meta, sliceType)
	if t.kind != rtgTypeSlice {
		return false
	}
	elemSize := rtgTypeSize(&g.meta, t.elem)
	if elemSize < 1 {
		elemSize = 8
	}
	lenOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	capOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg+1]) {
		return false
	}
	rtgAsmStoreRaxStack(&g.asm, lenOffset)
	if e.argCount == 3 {
		if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg+2]) {
			return false
		}
		rtgAsmStoreRaxStack(&g.asm, capOffset)
	} else {
		rtgAsmLoadRaxStack(&g.asm, lenOffset)
		rtgAsmStoreRaxStack(&g.asm, capOffset)
	}
	backingSize := 32768
	backingOff := g.asm.bssSize
	g.asm.bssSize += backingSize
	rtgAsmMovRaxBssAddr(&g.asm, backingOff)
	rtgAsmLoadRdxStack(&g.asm, lenOffset)
	rtgAsmLoadRcxStack(&g.asm, capOffset)
	return true
}

func rtgEmitEnvSliceRegs(g *rtgLinearGen) bool {
	if g.envDataOff == 0 && g.envLenOff == 0 {
		return false
	}
	rtgAsmLoadRaxBss(&g.asm, g.envLenOff)
	rtgAsmMovRdxRax(&g.asm)
	rtgAsmPushRdx(&g.asm)
	rtgAsmPopRcx(&g.asm)
	rtgAsmMovRaxBssAddr(&g.asm, g.envDataOff)
	return true
}

func rtgEmitEnvSliceToLocal(g *rtgLinearGen, offset int) bool {
	if !rtgEmitEnvSliceRegs(g) {
		return false
	}
	rtgAsmStoreRaxStack(&g.asm, offset)
	rtgAsmStoreRdxStack(&g.asm, offset-8)
	rtgAsmStoreRcxStack(&g.asm, offset-16)
	return true
}

func rtgEmitByteSliceConversionRegs(g *rtgLinearGen, ep rtgExprParse, e rtgExpr) bool {
	if e.argCount != 1 {
		return false
	}
	srcOff := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	lenOff := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	idxOff := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	backingOff := g.asm.bssSize
	backingSize := 32768
	g.asm.bssSize += backingSize
	argIndex := ep.args[e.firstArg]
	if !rtgEmitStringValueRegs(g, ep, argIndex) {
		return false
	}
	rtgAsmStoreRaxStack(&g.asm, srcOff)
	rtgAsmStoreRdxStack(&g.asm, lenOff)
	rtgAsmMovRaxImm(&g.asm, 0)
	rtgAsmStoreRaxStack(&g.asm, idxOff)
	loopLabel := rtgAsmNewLabel(&g.asm)
	doneLabel := rtgAsmNewLabel(&g.asm)
	rtgAsmMarkLabel(&g.asm, loopLabel)
	rtgAsmLoadRaxStack(&g.asm, idxOff)
	rtgAsmPushRax(&g.asm)
	rtgAsmLoadRaxStack(&g.asm, lenOff)
	rtgAsmPopRcx(&g.asm)
	rtgAsmCmpRcxRaxSet(&g.asm, 0x9d)
	rtgAsmCmpRaxImm8(&g.asm, 0)
	rtgAsmJnzLabel(&g.asm, doneLabel)
	rtgAsmLoadRaxStack(&g.asm, idxOff)
	rtgAsmPushRax(&g.asm)
	rtgAsmLoadRaxStack(&g.asm, srcOff)
	rtgAsmPopRcx(&g.asm)
	rtgAsmLoadByteRaxIndexRcx(&g.asm)
	rtgAsmPushRax(&g.asm)
	rtgAsmLoadRaxStack(&g.asm, idxOff)
	rtgAsmPushRax(&g.asm)
	rtgAsmMovRaxBssAddr(&g.asm, backingOff)
	rtgAsmMovRdxRax(&g.asm)
	rtgAsmPopRcx(&g.asm)
	rtgAsmPopRax(&g.asm)
	rtgAsmStoreAlMemRdxRcx1(&g.asm)
	rtgAsmLoadRaxStack(&g.asm, idxOff)
	rtgAsmIncRax(&g.asm)
	rtgAsmStoreRaxStack(&g.asm, idxOff)
	rtgAsmJmpLabel(&g.asm, loopLabel)
	rtgAsmMarkLabel(&g.asm, doneLabel)
	rtgAsmMovRaxBssAddr(&g.asm, backingOff)
	rtgAsmLoadRdxStack(&g.asm, lenOff)
	rtgAsmMovRcxRdx(&g.asm)
	return true
}

func rtgEmitStringValueRegs(g *rtgLinearGen, ep rtgExprParse, idx int) bool {
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
		msgLen := len(msg)
		rtgAsmMovRaxDataAddr(&g.asm, msgOff)
		rtgAsmMovRdxImm(&g.asm, msgLen)
		return true
	}
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			if !rtgTypeIsString(&g.meta, g.locals[localIndex].typ) {
				return false
			}
			rtgAsmLoadRaxStack(&g.asm, g.locals[localIndex].offset)
			rtgAsmLoadRdxStack(&g.asm, g.locals[localIndex].offset-8)
			return true
		}
		globalOffset := rtgFindGlobalOffset(g, e.nameStart, e.nameEnd)
		globalType := rtgFindGlobalType(g, e.nameStart, e.nameEnd)
		if globalOffset >= 0 && rtgTypeIsString(&g.meta, globalType) {
			rtgAsmLoadRaxBss(&g.asm, globalOffset)
			rtgAsmPushRax(&g.asm)
			rtgAsmLoadRaxBss(&g.asm, globalOffset+8)
			rtgAsmMovRdxRax(&g.asm)
			rtgAsmPopRax(&g.asm)
			return true
		}
		constTok := rtgFindConstStringToken(g, e.nameStart, e.nameEnd)
		if constTok >= 0 {
			msg := rtgDecodeStringToken(g.prog.src, g.prog.toks[constTok])
			msgOff := len(g.asm.data)
			for i := 0; i < len(msg); i++ {
				g.asm.data = append(g.asm.data, msg[i])
			}
			g.asm.data = append(g.asm.data, 0)
			msgLen := len(msg)
			rtgAsmMovRaxDataAddr(&g.asm, msgOff)
			rtgAsmMovRdxImm(&g.asm, msgLen)
			return true
		}
		return false
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
		rtgAsmMovRdxRax(&g.asm)
		rtgAsmLoadQwordRaxIndexRcx1(&g.asm)
		rtgAsmAddRdxRcx(&g.asm)
		rtgAsmLoadRdxMemRdxDisp(&g.asm, 8)
		return true
	}
	if e.kind == rtgExprSelector {
		valueType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsString(&g.meta, valueType) {
			return false
		}
		if !rtgEmitSelectorAddressRdx(g, ep, e) {
			return false
		}
		addrOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
		rtgAsmStoreRdxStack(&g.asm, addrOffset)
		rtgAsmLoadRaxMemRdxDisp(&g.asm, 0)
		rtgAsmPushRax(&g.asm)
		rtgAsmLoadRdxStack(&g.asm, addrOffset)
		rtgAsmLoadRaxMemRdxDisp(&g.asm, 8)
		rtgAsmMovRdxRax(&g.asm)
		rtgAsmPopRax(&g.asm)
		return true
	}
	if e.kind == rtgExprCall {
		callType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsString(&g.meta, callType) {
			return false
		}
		if !rtgEmitUserCall(g, ep, e) {
			return false
		}
		return true
	}
	return false
}

func rtgEmitCompositeFieldToStack(g *rtgLinearGen, ep rtgExprParse, idx int, fieldType int, destOffset int) bool {
	fieldResolved := rtgResolveType(&g.meta, fieldType)
	if fieldResolved.kind == rtgTypeSlice {
		if !rtgEmitSliceValueRegs(g, ep, idx) {
			return false
		}
		rtgAsmStoreRaxStack(&g.asm, destOffset)
		rtgAsmStoreRdxStack(&g.asm, destOffset-8)
		rtgAsmStoreRcxStack(&g.asm, destOffset-16)
		return true
	}
	if fieldResolved.kind == rtgTypeString {
		if !rtgEmitStringValueRegs(g, ep, idx) {
			return false
		}
		rtgAsmStoreRaxStack(&g.asm, destOffset)
		rtgAsmStoreRdxStack(&g.asm, destOffset-8)
		return true
	}
	if fieldResolved.kind == rtgTypeStruct {
		tempOffset := rtgAddTypedLocal(g, 0, 0, fieldType)
		if !rtgEmitTypedAssign(g, ep, idx, tempOffset) {
			return false
		}
		size := rtgTypeSize(&g.meta, fieldType)
		for at := 0; at < size; at += 8 {
			rtgAsmLoadRaxStack(&g.asm, tempOffset-at)
			rtgAsmStoreRaxStack(&g.asm, destOffset-at)
		}
		return true
	}
	if !rtgEmitIntExpr(g, ep, idx) {
		return false
	}
	rtgAsmStoreRaxStack(&g.asm, destOffset)
	return true
}

func rtgEmitCompositeFieldToMem(g *rtgLinearGen, ep rtgExprParse, idx int, fieldType int, addrOffset int, fieldOffset int) bool {
	fieldResolved := rtgResolveType(&g.meta, fieldType)
	if fieldResolved.kind == rtgTypeSlice {
		if !rtgEmitSliceValueRegs(g, ep, idx) {
			return false
		}
		rtgAsmPushRcx(&g.asm)
		rtgAsmPushRdx(&g.asm)
		rtgAsmPushRax(&g.asm)
		rtgAsmLoadRdxStack(&g.asm, addrOffset)
		rtgAsmPopRax(&g.asm)
		rtgAsmStoreRaxMemRdxDisp(&g.asm, fieldOffset)
		rtgAsmPopRax(&g.asm)
		rtgAsmStoreRaxMemRdxDisp(&g.asm, fieldOffset+8)
		rtgAsmPopRax(&g.asm)
		rtgAsmStoreRaxMemRdxDisp(&g.asm, fieldOffset+16)
		return true
	}
	if fieldResolved.kind == rtgTypeString {
		if !rtgEmitStringValueRegs(g, ep, idx) {
			return false
		}
		rtgAsmPushRdx(&g.asm)
		rtgAsmPushRax(&g.asm)
		rtgAsmLoadRdxStack(&g.asm, addrOffset)
		rtgAsmPopRax(&g.asm)
		rtgAsmStoreRaxMemRdxDisp(&g.asm, fieldOffset)
		rtgAsmPopRax(&g.asm)
		rtgAsmStoreRaxMemRdxDisp(&g.asm, fieldOffset+8)
		return true
	}
	if fieldResolved.kind == rtgTypeStruct {
		tempOffset := rtgAddTypedLocal(g, 0, 0, fieldType)
		if !rtgEmitTypedAssign(g, ep, idx, tempOffset) {
			return false
		}
		size := rtgTypeSize(&g.meta, fieldType)
		for at := 0; at < size; at += 8 {
			rtgAsmLoadRdxStack(&g.asm, addrOffset)
			rtgAsmLoadRaxStack(&g.asm, tempOffset-at)
			rtgAsmStoreRaxMemRdxDisp(&g.asm, fieldOffset+at)
		}
		return true
	}
	if !rtgEmitIntExpr(g, ep, idx) {
		return false
	}
	rtgAsmLoadRdxStack(&g.asm, addrOffset)
	rtgAsmStoreRaxMemRdxDisp(&g.asm, fieldOffset)
	return true
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
	if e.kind == rtgExprIndex {
		leftType := rtgInferParsedExprType(g, ep, e.left)
		sliceType := rtgResolveType(&g.meta, leftType)
		elemType := rtgResolveType(&g.meta, sliceType.elem)
		if sliceType.kind != rtgTypeSlice || elemType.kind != rtgTypeStruct || rtgTypeSize(&g.meta, sliceType.elem) != size {
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
		rtgAsmImulRcxImm(&g.asm, size)
		rtgAsmMovRdxRax(&g.asm)
		rtgAsmAddRdxRcx(&g.asm)
		rtgAsmLoadRcxStack(&g.asm, g.returnStruct)
		for at := 0; at < size; at += 8 {
			rtgAsmLoadRaxMemRdxDisp(&g.asm, at)
			rtgAsmStoreRaxMemRcxDisp(&g.asm, at)
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
			fieldType := rtgStructFieldType(g, resultType, field.nameStart, field.nameEnd)
			if fieldType == 0 {
				return false
			}
			if !rtgEmitCompositeFieldToMem(g, ep, field.expr, fieldType, g.returnStruct, fieldOffset) {
				return false
			}
		}
		return true
	}
	if e.kind == rtgExprCall {
		fnIndex := rtgFuncInfoFromCall(g, ep, e.left)
		if fnIndex < 0 || !rtgTypeIsStruct(&g.meta, g.meta.funcs[fnIndex].resultType) {
			return false
		}
		if rtgTypeSize(&g.meta, g.meta.funcs[fnIndex].resultType) != size {
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
		rtgAsmLoadRaxStack(&g.asm, g.returnStruct)
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
	if e.kind == rtgExprComposite {
		sliceType := rtgCompositeExprType(g, ep, e)
		if !rtgTypeIsSlice(&g.meta, sliceType) {
			return false
		}
		return rtgEmitSliceLiteralRegs(g, ep, e, sliceType)
	}
	if e.kind == rtgExprCall {
		if e.argCount == 2 && rtgExprIsIdentText(g.prog, ep, e.left, "append") {
			var stmt rtgStmt
			if !rtgEmitAppendAssignGeneral(g, stmt, ep) {
				return false
			}
			return rtgEmitSliceValueRegs(g, ep, ep.args[e.firstArg])
		}
		if e.argCount == 2 || e.argCount == 3 {
			if rtgExprIsIdentText(g.prog, ep, e.left, "make") {
				return rtgEmitMakeSliceRegs(g, ep, e)
			}
		}
		if e.argCount == 1 && rtgExprIsIdentText(g.prog, ep, e.left, "[]byte") {
			return rtgEmitByteSliceConversionRegs(g, ep, e)
		}
		if e.argCount == 0 && rtgExprIsIdentText(g.prog, ep, e.left, "rtgEnv") {
			return rtgEmitEnvSliceRegs(g)
		}
		fnIndex := rtgFuncInfoFromCall(g, ep, e.left)
		if fnIndex < 0 || !rtgTypeIsSlice(&g.meta, g.meta.funcs[fnIndex].resultType) {
			return false
		}
		return rtgEmitUserCall(g, ep, e)
	}
	return false
}

func rtgEmitStringReturnExpr(g *rtgLinearGen, ep rtgExprParse, idx int) bool {
	if !rtgEmitStringValueRegs(g, ep, idx) {
		return false
	}
	return true
}

func rtgEmitUserCall(g *rtgLinearGen, ep rtgExprParse, e rtgExpr) bool {
	a := &g.asm
	fnIndex := rtgFuncInfoFromCall(g, ep, e.left)
	if fnIndex < 0 {
		return false
	}
	if fnIndex >= len(g.funcLabels) {
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
		fieldType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsStruct(&g.meta, fieldType) || rtgTypeSize(&g.meta, fieldType) != size {
			return -1
		}
		if !rtgEmitSelectorAddressRdx(g, ep, e) {
			return -1
		}
		for at := size - 8; at >= 0; at -= 8 {
			rtgAsmLoadRaxMemRdxDisp(&g.asm, at)
			rtgAsmPushRax(&g.asm)
		}
		return size / 8
	}
	if e.kind == rtgExprComposite {
		offset := rtgAddTypedLocal(g, 0, 0, typ)
		rtgZeroLocalAtOffset(g, offset)
		for i := 0; i < e.argCount; i++ {
			field := ep.fields[e.firstArg+i]
			fieldOffset := rtgStructFieldOffset(g, typ, field.nameStart, field.nameEnd)
			if fieldOffset < 0 {
				return -1
			}
			if !rtgEmitIntExpr(g, ep, field.expr) {
				return -1
			}
			rtgAsmStoreRaxStack(&g.asm, offset-fieldOffset)
		}
		for at := size - 8; at >= 0; at -= 8 {
			rtgAsmLoadRaxStack(&g.asm, offset-at)
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
	if e.kind == rtgExprComposite {
		sliceType := rtgCompositeExprType(g, ep, e)
		if !rtgTypeIsSlice(&g.meta, sliceType) {
			return false
		}
		if !rtgEmitSliceLiteralRegs(g, ep, e, sliceType) {
			return false
		}
		rtgAsmPushRcx(&g.asm)
		rtgAsmPushRdx(&g.asm)
		rtgAsmPushRax(&g.asm)
		return true
	}
	if e.kind == rtgExprCall {
		if e.argCount == 2 || e.argCount == 3 {
			if rtgExprIsIdentText(g.prog, ep, e.left, "make") {
				if !rtgEmitMakeSliceRegs(g, ep, e) {
					return false
				}
				rtgAsmPushRcx(&g.asm)
				rtgAsmPushRdx(&g.asm)
				rtgAsmPushRax(&g.asm)
				return true
			}
		}
		if e.argCount == 1 && rtgExprIsIdentText(g.prog, ep, e.left, "[]byte") {
			if !rtgEmitByteSliceConversionRegs(g, ep, e) {
				return false
			}
			rtgAsmPushRcx(&g.asm)
			rtgAsmPushRdx(&g.asm)
			rtgAsmPushRax(&g.asm)
			return true
		}
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
		msgLen := len(msg)
		rtgAsmMovRaxImm(&g.asm, msgLen)
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
	if e.kind == rtgExprIndex {
		argType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsString(&g.meta, argType) {
			return false
		}
		if !rtgEmitStringValueRegs(g, ep, idx) {
			return false
		}
		rtgAsmPushRdx(&g.asm)
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
		value := rtgParseIntToken(p.src, p.toks[e.tok])
		rtgAsmMovRaxImm(a, value)
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
		value := rtgParseCharToken(p.src, p.toks[e.tok])
		rtgAsmMovRaxImm(a, value)
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
				msgLen := len(msg)
				rtgAsmMovRaxImm(a, msgLen)
				return true
			}
			if arg.kind == rtgExprIdent {
				localIndex := rtgFindLocalIndex(g, arg.nameStart, arg.nameEnd)
				if localIndex >= 0 && (rtgTypeIsSlice(&g.meta, g.locals[localIndex].typ) || rtgTypeIsString(&g.meta, g.locals[localIndex].typ)) {
					rtgAsmLoadRaxStack(a, g.locals[localIndex].offset-8)
					return true
				}
				globalOffset := rtgFindGlobalOffset(g, arg.nameStart, arg.nameEnd)
				globalType := rtgFindGlobalType(g, arg.nameStart, arg.nameEnd)
				if globalOffset >= 0 && (rtgTypeIsString(&g.meta, globalType) || rtgTypeIsSlice(&g.meta, globalType)) {
					rtgAsmLoadRaxBss(a, globalOffset+8)
					return true
				}
				constTok := rtgFindConstStringToken(g, arg.nameStart, arg.nameEnd)
				if constTok >= 0 {
					msg := rtgDecodeStringToken(p.src, p.toks[constTok])
					msgLen := len(msg)
					rtgAsmMovRaxImm(a, msgLen)
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
				if localIndex >= 0 {
					rtgAsmLeaRaxStack(a, g.locals[localIndex].offset)
					return true
				}
				globalOffset := rtgFindGlobalOffset(g, inner.nameStart, inner.nameEnd)
				if globalOffset >= 0 {
					rtgAsmMovRaxBssAddr(a, globalOffset)
					return true
				}
				return false
			}
			if inner.kind == rtgExprSelector {
				if !rtgEmitSelectorAddressRdx(g, ep, inner) {
					return false
				}
				rtgAsmMovRaxRdx(a)
				return true
			}
			if inner.kind == rtgExprIndex {
				return rtgEmitIndexAddressRax(g, ep, inner)
			}
			return false
		}
		if rtgTokTextIs(p, e.tok, "*") {
			if !rtgEmitIntExpr(g, ep, e.left) {
				return false
			}
			rtgAsmMovRdxRax(a)
			rtgAsmLoadRaxMemRdxDisp(a, 0)
			return true
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
		if rtgTokTextIs(p, e.tok, "==") || rtgTokTextIs(p, e.tok, "!=") {
			leftType := rtgInferParsedExprType(g, ep, e.left)
			rightType := rtgInferParsedExprType(g, ep, e.right)
			if rtgTypeIsString(&g.meta, leftType) || rtgTypeIsString(&g.meta, rightType) {
				notEqual := rtgTokTextIs(p, e.tok, "!=")
				return rtgEmitStringCompare(g, ep, e.left, e.right, notEqual)
			}
		}
		rightExpr := ep.exprs[e.right]
		rightKind := rightExpr.kind
		rightTok := rightExpr.tok
		if !rtgEmitIntExpr(g, ep, e.left) {
			return false
		}
		rtgAsmPushRax(a)
		if rightKind == rtgExprInt {
			value := rtgParseIntToken(p.src, p.toks[rightTok])
			rtgAsmMovRaxImm(a, value)
		} else if rightKind == rtgExprChar {
			value := rtgParseCharToken(p.src, p.toks[rightTok])
			rtgAsmMovRaxImm(a, value)
		} else if rightKind == rtgExprBool {
			if rtgBytesEqualText(p.src, p.toks[rightTok].start, p.toks[rightTok].end, "true") {
				rtgAsmMovRaxImm(a, 1)
			} else {
				rtgAsmMovRaxImm(a, 0)
			}
		} else {
			if !rtgEmitIntExpr(g, ep, e.right) {
				return false
			}
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
	var loc rtgSliceLocation
	locEp := ep
	assignTok := rtgFindAssignmentToken(g.prog, stmt.startTok, stmt.endTok)
	if assignTok > stmt.startTok {
		lhs := rtgParseExpression(g.prog, stmt.startTok, assignTok)
		if lhs.ok && len(lhs.exprs) > 0 {
			lhsIndex := len(lhs.exprs) - 1
			loc = rtgSliceLocationFromExpr(g, lhs, lhsIndex)
			locEp = lhs
		}
	}
	if !loc.ok {
		loc = rtgSliceLocationFromExpr(g, ep, ep.args[root.firstArg])
		locEp = ep
	}
	if !loc.ok {
		return false
	}
	t := rtgResolveType(&g.meta, loc.typ)
	if t.kind != rtgTypeSlice {
		return false
	}
	elem := rtgResolveType(&g.meta, t.elem)
	valueIndex := ep.args[root.firstArg+1]
	if root.nameStart == 1 {
		return rtgEmitAppendExpansionToLocation(g, ep, locEp, loc, t.elem, valueIndex)
	}
	if elem.kind == rtgTypeStruct {
		value := ep.exprs[valueIndex]
		if value.kind != rtgExprComposite {
			if value.kind == rtgExprIdent {
				typeTok := value.tok
				if !rtgTokTextIs(g.prog, typeTok+1, "{") {
					typeTok = rtgFindTokenByStart(g.prog, value.nameStart)
				}
				if rtgTokTextIs(g.prog, typeTok+1, "{") {
					return rtgEmitAppendStructCompositeTokens(g, locEp, loc, t.elem, typeTok)
				}
				return rtgEmitAppendStructLocal(g, ep, locEp, loc, t.elem, value)
			}
			typeTok := rtgFindAppendCompositeTypeToken(g.prog, root.tok, stmt.endTok)
			if typeTok >= 0 {
				return rtgEmitAppendStructCompositeTokens(g, locEp, loc, t.elem, typeTok)
			}
			return false
		}
		if !rtgEmitAppendStructComposite(g, ep, locEp, loc, t.elem, value) {
			return false
		}
		return true
	}
	if elem.kind == rtgTypeInt || elem.kind == rtgTypeByte || elem.kind == rtgTypeBool {
		if !rtgEmitAppendScalarToLocation(g, ep, locEp, loc, elem.kind, valueIndex) {
			return false
		}
		return true
	}
	if elem.kind == rtgTypeString {
		if !rtgEmitAppendStringToLocation(g, ep, locEp, loc, valueIndex) {
			return false
		}
		return true
	}
	return false
}

func rtgEmitAppendExpansionToLocation(g *rtgLinearGen, ep rtgExprParse, locEp rtgExprParse, loc rtgSliceLocation, elemType int, valueIndex int) bool {
	elemSize := rtgTypeSize(&g.meta, elemType)
	if elemSize < 1 {
		elemSize = 8
	}
	if elemSize != 1 && elemSize%8 != 0 {
		return false
	}
	sourceType := rtgInferParsedExprType(g, ep, valueIndex)
	source := rtgResolveType(&g.meta, sourceType)
	if source.kind != rtgTypeSlice {
		return false
	}
	if rtgTypeSize(&g.meta, source.elem) != elemSize {
		return false
	}
	srcPtr := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	srcLen := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	srcIndex := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	destPtr := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	destLen := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	headerOffset := 0
	if !rtgEmitSliceValueRegs(g, ep, valueIndex) {
		return false
	}
	rtgAsmStoreRaxStack(&g.asm, srcPtr)
	rtgAsmStoreRdxStack(&g.asm, srcLen)
	rtgAsmMovRaxImm(&g.asm, 0)
	rtgAsmStoreRaxStack(&g.asm, srcIndex)
	if loc.mem {
		if loc.expr < 0 || loc.expr >= len(locEp.exprs) {
			return false
		}
		if !rtgEmitSelectorAddressRdx(g, locEp, locEp.exprs[loc.expr]) {
			return false
		}
		headerOffset = rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
		rtgAsmStoreRdxStack(&g.asm, headerOffset)
		rtgEmitEnsureMemSlice(g, elemSize)
		rtgAsmLoadRaxMemRdxDisp(&g.asm, 0)
		rtgAsmStoreRaxStack(&g.asm, destPtr)
		rtgAsmLoadRdxStack(&g.asm, headerOffset)
		rtgAsmLoadRaxMemRdxDisp(&g.asm, 8)
		rtgAsmStoreRaxStack(&g.asm, destLen)
	} else if loc.global {
		rtgAsmLoadRaxBss(&g.asm, loc.offset)
		rtgAsmStoreRaxStack(&g.asm, destPtr)
		rtgAsmLoadRaxBss(&g.asm, loc.offset+8)
		rtgAsmStoreRaxStack(&g.asm, destLen)
	} else {
		rtgAsmLoadRaxStack(&g.asm, loc.offset)
		rtgAsmStoreRaxStack(&g.asm, destPtr)
		rtgAsmLoadRaxStack(&g.asm, loc.offset-8)
		rtgAsmStoreRaxStack(&g.asm, destLen)
	}
	loopLabel := rtgAsmNewLabel(&g.asm)
	doneLabel := rtgAsmNewLabel(&g.asm)
	rtgAsmMarkLabel(&g.asm, loopLabel)
	rtgAsmLoadRaxStack(&g.asm, srcIndex)
	rtgAsmPushRax(&g.asm)
	rtgAsmLoadRaxStack(&g.asm, srcLen)
	rtgAsmPopRcx(&g.asm)
	rtgAsmCmpRcxRaxSet(&g.asm, 0x9d)
	rtgAsmCmpRaxImm8(&g.asm, 0)
	rtgAsmJnzLabel(&g.asm, doneLabel)
	rtgEmitAppendExpansionCopyElement(g, elemSize, srcPtr, srcIndex, destPtr, destLen)
	rtgAsmLoadRaxStack(&g.asm, srcIndex)
	rtgAsmIncRax(&g.asm)
	rtgAsmStoreRaxStack(&g.asm, srcIndex)
	rtgAsmLoadRaxStack(&g.asm, destLen)
	rtgAsmIncRax(&g.asm)
	rtgAsmStoreRaxStack(&g.asm, destLen)
	rtgAsmJmpLabel(&g.asm, loopLabel)
	rtgAsmMarkLabel(&g.asm, doneLabel)
	rtgAsmLoadRaxStack(&g.asm, destLen)
	if loc.mem {
		rtgAsmLoadRdxStack(&g.asm, headerOffset)
		rtgAsmStoreRaxMemRdxDisp(&g.asm, 8)
	} else if loc.global {
		rtgAsmStoreRaxBss(&g.asm, loc.offset+8)
	} else {
		rtgAsmStoreRaxStack(&g.asm, loc.offset-8)
	}
	return true
}

func rtgEmitAppendExpansionCopyElement(g *rtgLinearGen, elemSize int, srcPtr int, srcIndex int, destPtr int, destLen int) {
	if elemSize == 1 {
		rtgAsmLoadRaxStack(&g.asm, srcPtr)
		rtgAsmLoadRcxStack(&g.asm, srcIndex)
		rtgAsmLoadByteRaxIndexRcx(&g.asm)
		rtgAsmPushRax(&g.asm)
		rtgAsmLoadRdxStack(&g.asm, destPtr)
		rtgAsmLoadRcxStack(&g.asm, destLen)
		rtgAsmPopRax(&g.asm)
		rtgAsmStoreAlMemRdxRcx1(&g.asm)
		return
	}
	if elemSize == 8 {
		rtgAsmLoadRaxStack(&g.asm, srcPtr)
		rtgAsmLoadRcxStack(&g.asm, srcIndex)
		rtgAsmLoadQwordRaxIndexRcx8(&g.asm)
		rtgAsmPushRax(&g.asm)
		rtgAsmLoadRdxStack(&g.asm, destPtr)
		rtgAsmLoadRcxStack(&g.asm, destLen)
		rtgAsmPopRax(&g.asm)
		rtgAsmStoreRaxMemRdxRcx8(&g.asm)
		return
	}
	for copyOff := 0; copyOff < elemSize; copyOff += 8 {
		rtgAsmLoadRaxStack(&g.asm, srcPtr)
		rtgAsmLoadRcxStack(&g.asm, srcIndex)
		rtgAsmImulRcxImm(&g.asm, elemSize)
		rtgAsmLoadQwordRaxIndexRcxDisp(&g.asm, copyOff)
		rtgAsmPushRax(&g.asm)
		rtgAsmLoadRdxStack(&g.asm, destPtr)
		rtgAsmLoadRcxStack(&g.asm, destLen)
		rtgAsmImulRcxImm(&g.asm, elemSize)
		rtgAsmAddRdxRcx(&g.asm)
		rtgAsmPopRax(&g.asm)
		rtgAsmStoreRaxMemRdxDisp(&g.asm, copyOff)
	}
}

func rtgFindAppendCompositeTypeToken(p rtgProgram, openTok int, end int) int {
	if openTok < 0 || openTok >= end || !rtgTokTextIs(p, openTok, "(") {
		return -1
	}
	i := openTok + 1
	paren := 0
	brack := 0
	brace := 0
	for i < end {
		if paren == 0 && brack == 0 && brace == 0 && rtgTokTextIs(p, i, ",") {
			typeTok := i + 1
			if rtgTokIsKind(p, typeTok, rtgTokIdent) && rtgTokTextIs(p, typeTok+1, "{") {
				return typeTok
			}
			return -1
		}
		if rtgTokTextIs(p, i, "(") {
			paren++
		} else if rtgTokTextIs(p, i, ")") {
			if paren == 0 {
				return -1
			}
			paren--
		} else if rtgTokTextIs(p, i, "[") {
			brack++
		} else if rtgTokTextIs(p, i, "]") {
			brack--
		} else if rtgTokTextIs(p, i, "{") {
			brace++
		} else if rtgTokTextIs(p, i, "}") {
			brace--
		}
		i++
	}
	return -1
}

func rtgEmitAppendScalarToLocation(g *rtgLinearGen, ep rtgExprParse, locEp rtgExprParse, loc rtgSliceLocation, elemKind int, valueIndex int) bool {
	if !rtgEmitIntExpr(g, ep, valueIndex) {
		return false
	}
	rtgAsmPushRax(&g.asm)
	if loc.mem {
		if loc.expr < 0 || loc.expr >= len(locEp.exprs) {
			return false
		}
		if !rtgEmitSelectorAddressRdx(g, locEp, locEp.exprs[loc.expr]) {
			return false
		}
		elemSize := 8
		if elemKind == rtgTypeByte || elemKind == rtgTypeBool {
			elemSize = 1
		}
		rtgEmitEnsureMemSlice(g, elemSize)
		rtgAsmLoadRaxMemRdxDisp(&g.asm, 0)
		rtgAsmPushRax(&g.asm)
		rtgAsmLoadRaxMemRdxDisp(&g.asm, 8)
		rtgAsmMovRcxRax(&g.asm)
		rtgAsmPopRdx(&g.asm)
	} else {
		if loc.global {
			rtgAsmLoadRaxBss(&g.asm, loc.offset)
			rtgAsmPushRax(&g.asm)
			rtgAsmLoadRaxBss(&g.asm, loc.offset+8)
			rtgAsmMovRcxRax(&g.asm)
			rtgAsmPopRdx(&g.asm)
		} else {
			rtgAsmLoadRaxStack(&g.asm, loc.offset)
			rtgAsmPushRax(&g.asm)
			rtgAsmLoadRaxStack(&g.asm, loc.offset-8)
			rtgAsmMovRcxRax(&g.asm)
			rtgAsmPopRdx(&g.asm)
		}
	}
	rtgAsmPopRax(&g.asm)
	if elemKind == rtgTypeByte || elemKind == rtgTypeBool {
		rtgAsmStoreAlMemRdxRcx1(&g.asm)
	} else {
		rtgAsmStoreRaxMemRdxRcx8(&g.asm)
	}
	rtgAsmIncRcx(&g.asm)
	rtgAsmMovRaxRcx(&g.asm)
	if loc.mem {
		if !rtgEmitSelectorAddressRdx(g, locEp, locEp.exprs[loc.expr]) {
			return false
		}
		rtgAsmStoreRaxMemRdxDisp(&g.asm, 8)
	} else {
		if loc.global {
			rtgAsmStoreRaxBss(&g.asm, loc.offset+8)
		} else {
			rtgAsmStoreRaxStack(&g.asm, loc.offset-8)
		}
	}
	return true
}

func rtgEmitAppendStringToLocation(g *rtgLinearGen, ep rtgExprParse, locEp rtgExprParse, loc rtgSliceLocation, valueIndex int) bool {
	if !rtgEmitStringValueRegs(g, ep, valueIndex) {
		return false
	}
	rtgAsmPushRdx(&g.asm)
	rtgAsmPushRax(&g.asm)
	if loc.mem {
		if loc.expr < 0 || loc.expr >= len(locEp.exprs) {
			return false
		}
		if !rtgEmitSelectorAddressRdx(g, locEp, locEp.exprs[loc.expr]) {
			return false
		}
		rtgEmitEnsureMemSlice(g, 16)
		rtgAsmLoadRaxMemRdxDisp(&g.asm, 0)
		rtgAsmPushRax(&g.asm)
		rtgAsmLoadRaxMemRdxDisp(&g.asm, 8)
		rtgAsmMovRcxRax(&g.asm)
		rtgAsmPopRdx(&g.asm)
	} else {
		if loc.global {
			rtgAsmLoadRaxBss(&g.asm, loc.offset)
			rtgAsmPushRax(&g.asm)
			rtgAsmLoadRaxBss(&g.asm, loc.offset+8)
			rtgAsmMovRcxRax(&g.asm)
			rtgAsmPopRdx(&g.asm)
		} else {
			rtgAsmLoadRaxStack(&g.asm, loc.offset)
			rtgAsmPushRax(&g.asm)
			rtgAsmLoadRaxStack(&g.asm, loc.offset-8)
			rtgAsmMovRcxRax(&g.asm)
			rtgAsmPopRdx(&g.asm)
		}
	}
	rtgAsmShlRcxImm(&g.asm, 4)
	rtgAsmAddRdxRcx(&g.asm)
	rtgAsmPopRax(&g.asm)
	rtgAsmStoreRaxMemRdxDisp(&g.asm, 0)
	rtgAsmPopRax(&g.asm)
	rtgAsmStoreRaxMemRdxDisp(&g.asm, 8)
	if loc.mem {
		if !rtgEmitSelectorAddressRdx(g, locEp, locEp.exprs[loc.expr]) {
			return false
		}
		rtgAsmLoadRaxMemRdxDisp(&g.asm, 8)
	} else {
		if loc.global {
			rtgAsmLoadRaxBss(&g.asm, loc.offset+8)
		} else {
			rtgAsmLoadRaxStack(&g.asm, loc.offset-8)
		}
	}
	rtgAsmMovRcxRax(&g.asm)
	rtgAsmIncRcx(&g.asm)
	rtgAsmMovRaxRcx(&g.asm)
	if loc.mem {
		if !rtgEmitSelectorAddressRdx(g, locEp, locEp.exprs[loc.expr]) {
			return false
		}
		rtgAsmStoreRaxMemRdxDisp(&g.asm, 8)
	} else {
		if loc.global {
			rtgAsmStoreRaxBss(&g.asm, loc.offset+8)
		} else {
			rtgAsmStoreRaxStack(&g.asm, loc.offset-8)
		}
	}
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
		if localIndex < 0 {
			globalOffset := rtgFindGlobalOffset(g, e.nameStart, e.nameEnd)
			globalType := rtgFindGlobalType(g, e.nameStart, e.nameEnd)
			if globalOffset < 0 || !rtgTypeIsSlice(&g.meta, globalType) {
				return loc
			}
			loc.offset = globalOffset
			loc.typ = globalType
			loc.global = true
			loc.ok = true
			return loc
		}
		if !rtgTypeIsSlice(&g.meta, g.locals[localIndex].typ) {
			return loc
		}
		loc.offset = g.locals[localIndex].offset
		loc.typ = g.locals[localIndex].typ
		loc.ok = true
		return loc
	}
	if e.kind == rtgExprSelector {
		fieldType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsSlice(&g.meta, fieldType) {
			return loc
		}
		loc.expr = idx
		loc.typ = fieldType
		loc.mem = true
		loc.ok = true
		return loc
	}
	return loc
}

func rtgEmitEnsureMemSlice(g *rtgLinearGen, elemSize int) {
	if elemSize < 1 {
		elemSize = 8
	}
	okLabel := rtgAsmNewLabel(&g.asm)
	rtgAsmLoadRaxMemRdxDisp(&g.asm, 0)
	rtgAsmCmpRaxImm8(&g.asm, 0)
	rtgAsmJnzLabel(&g.asm, okLabel)
	backingSize := 8388608
	backingOff := g.asm.bssSize
	g.asm.bssSize += backingSize
	rtgAsmMovRaxBssAddr(&g.asm, backingOff)
	rtgAsmStoreRaxMemRdxDisp(&g.asm, 0)
	rtgAsmMovRaxImm(&g.asm, backingSize/elemSize)
	rtgAsmStoreRaxMemRdxDisp(&g.asm, 16)
	rtgAsmMarkLabel(&g.asm, okLabel)
}

func rtgEmitAppendStructCompositeTokens(g *rtgLinearGen, locEp rtgExprParse, loc rtgSliceLocation, elemType int, typeTok int) bool {
	openTok := typeTok + 1
	closeTok := rtgSkipBalanced(g.prog, openTok, "{", "}")
	if closeTok <= openTok {
		return false
	}
	elemSize := rtgTypeSize(&g.meta, elemType)
	if loc.mem {
		if loc.expr < 0 || loc.expr >= len(locEp.exprs) {
			return false
		}
		if !rtgEmitSelectorAddressRdx(g, locEp, locEp.exprs[loc.expr]) {
			return false
		}
		rtgEmitEnsureMemSlice(g, elemSize)
		rtgAsmLoadRaxMemRdxDisp(&g.asm, 0)
		rtgAsmPushRax(&g.asm)
		rtgAsmLoadRaxMemRdxDisp(&g.asm, 8)
		rtgAsmMovRcxRax(&g.asm)
		rtgAsmImulRcxImm(&g.asm, elemSize)
		rtgAsmPopRdx(&g.asm)
	} else {
		rtgAsmLoadRaxStack(&g.asm, loc.offset)
		rtgAsmPushRax(&g.asm)
		rtgAsmLoadRaxStack(&g.asm, loc.offset-8)
		rtgAsmMovRcxRax(&g.asm)
		rtgAsmImulRcxImm(&g.asm, elemSize)
		rtgAsmPopRdx(&g.asm)
	}
	rtgAsmAddRdxRcx(&g.asm)
	destOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	rtgAsmStoreRdxStack(&g.asm, destOffset)
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
		fieldType := rtgStructFieldType(g, elemType, fieldTok.start, fieldTok.end)
		if fieldType == 0 {
			return false
		}
		rootIndex := len(ep.exprs) - 1
		if !rtgEmitCompositeFieldToMem(g, ep, rootIndex, fieldType, destOffset, fieldOffset) {
			return false
		}
		i = exprEnd
		if rtgTokTextIs(g.prog, i, ",") {
			i++
		}
	}
	if loc.mem {
		if !rtgEmitSelectorAddressRdx(g, locEp, locEp.exprs[loc.expr]) {
			return false
		}
		rtgAsmLoadRaxMemRdxDisp(&g.asm, 8)
	} else {
		rtgAsmLoadRaxStack(&g.asm, loc.offset-8)
	}
	rtgAsmMovRcxRax(&g.asm)
	rtgAsmIncRcx(&g.asm)
	rtgAsmMovRaxRcx(&g.asm)
	if loc.mem {
		if !rtgEmitSelectorAddressRdx(g, locEp, locEp.exprs[loc.expr]) {
			return false
		}
		rtgAsmStoreRaxMemRdxDisp(&g.asm, 8)
	} else {
		rtgAsmStoreRaxStack(&g.asm, loc.offset-8)
	}
	return true
}

func rtgEmitAppendStructLocal(g *rtgLinearGen, ep rtgExprParse, locEp rtgExprParse, loc rtgSliceLocation, elemType int, value rtgExpr) bool {
	localIndex := rtgFindLocalIndex(g, value.nameStart, value.nameEnd)
	if localIndex < 0 {
		return false
	}
	elemSize := rtgTypeSize(&g.meta, elemType)
	if rtgTypeSize(&g.meta, g.locals[localIndex].typ) != elemSize {
		return false
	}
	if loc.mem {
		if loc.expr < 0 || loc.expr >= len(locEp.exprs) {
			return false
		}
		if !rtgEmitSelectorAddressRdx(g, locEp, locEp.exprs[loc.expr]) {
			return false
		}
		rtgEmitEnsureMemSlice(g, elemSize)
		rtgAsmLoadRaxMemRdxDisp(&g.asm, 0)
		rtgAsmPushRax(&g.asm)
		rtgAsmLoadRaxMemRdxDisp(&g.asm, 8)
		rtgAsmMovRcxRax(&g.asm)
		rtgAsmImulRcxImm(&g.asm, elemSize)
		rtgAsmPopRdx(&g.asm)
	} else {
		rtgAsmLoadRaxStack(&g.asm, loc.offset)
		rtgAsmPushRax(&g.asm)
		rtgAsmLoadRaxStack(&g.asm, loc.offset-8)
		rtgAsmMovRcxRax(&g.asm)
		rtgAsmImulRcxImm(&g.asm, elemSize)
		rtgAsmPopRdx(&g.asm)
	}
	rtgAsmAddRdxRcx(&g.asm)
	rtgAsmPushRdx(&g.asm)
	for at := 0; at < elemSize; at += 8 {
		rtgAsmLoadRaxStack(&g.asm, g.locals[localIndex].offset-at)
		rtgAsmPopRdx(&g.asm)
		rtgAsmStoreRaxMemRdxDisp(&g.asm, at)
		rtgAsmPushRdx(&g.asm)
	}
	rtgAsmPopRdx(&g.asm)
	if loc.mem {
		if !rtgEmitSelectorAddressRdx(g, locEp, locEp.exprs[loc.expr]) {
			return false
		}
		rtgAsmLoadRaxMemRdxDisp(&g.asm, 8)
	} else {
		rtgAsmLoadRaxStack(&g.asm, loc.offset-8)
	}
	rtgAsmMovRcxRax(&g.asm)
	rtgAsmIncRcx(&g.asm)
	rtgAsmMovRaxRcx(&g.asm)
	if loc.mem {
		if !rtgEmitSelectorAddressRdx(g, locEp, locEp.exprs[loc.expr]) {
			return false
		}
		rtgAsmStoreRaxMemRdxDisp(&g.asm, 8)
	} else {
		rtgAsmStoreRaxStack(&g.asm, loc.offset-8)
	}
	return true
}

func rtgEmitAppendStructComposite(g *rtgLinearGen, ep rtgExprParse, locEp rtgExprParse, loc rtgSliceLocation, elemType int, value rtgExpr) bool {
	elemSize := rtgTypeSize(&g.meta, elemType)
	headerOffset := 0
	if loc.mem {
		if loc.expr < 0 || loc.expr >= len(locEp.exprs) {
			return false
		}
		if !rtgEmitSelectorAddressRdx(g, locEp, locEp.exprs[loc.expr]) {
			return false
		}
		headerOffset = rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
		rtgAsmStoreRdxStack(&g.asm, headerOffset)
		rtgEmitEnsureMemSlice(g, elemSize)
		rtgAsmLoadRaxMemRdxDisp(&g.asm, 0)
		rtgAsmPushRax(&g.asm)
		rtgAsmLoadRaxMemRdxDisp(&g.asm, 8)
		rtgAsmMovRcxRax(&g.asm)
		rtgAsmImulRcxImm(&g.asm, elemSize)
		rtgAsmPopRdx(&g.asm)
	} else {
		rtgAsmLoadRaxStack(&g.asm, loc.offset)
		rtgAsmPushRax(&g.asm)
		rtgAsmLoadRaxStack(&g.asm, loc.offset-8)
		rtgAsmMovRcxRax(&g.asm)
		rtgAsmImulRcxImm(&g.asm, elemSize)
		rtgAsmPopRdx(&g.asm)
	}
	rtgAsmAddRdxRcx(&g.asm)
	destOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	rtgAsmStoreRdxStack(&g.asm, destOffset)
	for i := 0; i < value.argCount; i++ {
		field := ep.fields[value.firstArg+i]
		fieldOffset := rtgStructFieldOffset(g, elemType, field.nameStart, field.nameEnd)
		if fieldOffset < 0 {
			return false
		}
		fieldType := rtgStructFieldType(g, elemType, field.nameStart, field.nameEnd)
		if fieldType == 0 {
			return false
		}
		if !rtgEmitCompositeFieldToMem(g, ep, field.expr, fieldType, destOffset, fieldOffset) {
			return false
		}
	}
	if loc.mem {
		rtgAsmLoadRdxStack(&g.asm, headerOffset)
		rtgAsmLoadRaxMemRdxDisp(&g.asm, 8)
	} else {
		rtgAsmLoadRaxStack(&g.asm, loc.offset-8)
	}
	rtgAsmMovRcxRax(&g.asm)
	rtgAsmIncRcx(&g.asm)
	rtgAsmMovRaxRcx(&g.asm)
	if loc.mem {
		rtgAsmLoadRdxStack(&g.asm, headerOffset)
		rtgAsmStoreRaxMemRdxDisp(&g.asm, 8)
	} else {
		rtgAsmStoreRaxStack(&g.asm, loc.offset-8)
	}
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
	if elem.kind != rtgTypeInt && elem.kind != rtgTypeByte && elem.kind != rtgTypeBool {
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
	if elem.kind == rtgTypeByte || elem.kind == rtgTypeBool {
		rtgAsmStoreAlMemRdxRcx1(&g.asm)
	} else {
		rtgAsmStoreRaxMemRdxRcx8(&g.asm)
	}
	rtgAsmIncRcx(&g.asm)
	rtgAsmMovRaxRcx(&g.asm)
	rtgAsmStoreRaxStack(&g.asm, offset-8)
	return true
}

func rtgEmitStringCompare(g *rtgLinearGen, ep rtgExprParse, left int, right int, notEqual bool) bool {
	leftPtr := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	leftLen := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	rightPtr := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	rightLen := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	index := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	if !rtgEmitStringValueRegs(g, ep, left) {
		return false
	}
	rtgAsmStoreRaxStack(&g.asm, leftPtr)
	rtgAsmStoreRdxStack(&g.asm, leftLen)
	if !rtgEmitStringValueRegs(g, ep, right) {
		return false
	}
	rtgAsmStoreRaxStack(&g.asm, rightPtr)
	rtgAsmStoreRdxStack(&g.asm, rightLen)

	notEqualLabel := rtgAsmNewLabel(&g.asm)
	equalLabel := rtgAsmNewLabel(&g.asm)
	loopLabel := rtgAsmNewLabel(&g.asm)
	endLabel := rtgAsmNewLabel(&g.asm)

	rtgAsmLoadRaxStack(&g.asm, leftLen)
	rtgAsmPushRax(&g.asm)
	rtgAsmLoadRaxStack(&g.asm, rightLen)
	rtgAsmPopRcx(&g.asm)
	rtgAsmCmpRcxRaxSet(&g.asm, 0x94)
	rtgAsmCmpRaxImm8(&g.asm, 0)
	rtgAsmJzLabel(&g.asm, notEqualLabel)

	rtgAsmMovRaxImm(&g.asm, 0)
	rtgAsmStoreRaxStack(&g.asm, index)
	rtgAsmMarkLabel(&g.asm, loopLabel)
	rtgAsmLoadRaxStack(&g.asm, index)
	rtgAsmPushRax(&g.asm)
	rtgAsmLoadRaxStack(&g.asm, leftLen)
	rtgAsmPopRcx(&g.asm)
	rtgAsmCmpRcxRaxSet(&g.asm, 0x9d)
	rtgAsmCmpRaxImm8(&g.asm, 0)
	rtgAsmJnzLabel(&g.asm, equalLabel)

	rtgAsmLoadRaxStack(&g.asm, index)
	rtgAsmPushRax(&g.asm)
	rtgAsmLoadRaxStack(&g.asm, leftPtr)
	rtgAsmPopRcx(&g.asm)
	rtgAsmLoadByteRaxIndexRcx(&g.asm)
	rtgAsmPushRax(&g.asm)
	rtgAsmLoadRaxStack(&g.asm, index)
	rtgAsmPushRax(&g.asm)
	rtgAsmLoadRaxStack(&g.asm, rightPtr)
	rtgAsmPopRcx(&g.asm)
	rtgAsmLoadByteRaxIndexRcx(&g.asm)
	rtgAsmPopRcx(&g.asm)
	rtgAsmCmpRcxRaxSet(&g.asm, 0x94)
	rtgAsmCmpRaxImm8(&g.asm, 0)
	rtgAsmJzLabel(&g.asm, notEqualLabel)

	rtgAsmLoadRaxStack(&g.asm, index)
	rtgAsmIncRax(&g.asm)
	rtgAsmStoreRaxStack(&g.asm, index)
	rtgAsmJmpLabel(&g.asm, loopLabel)

	rtgAsmMarkLabel(&g.asm, equalLabel)
	if notEqual {
		rtgAsmMovRaxImm(&g.asm, 0)
	} else {
		rtgAsmMovRaxImm(&g.asm, 1)
	}
	rtgAsmJmpLabel(&g.asm, endLabel)
	rtgAsmMarkLabel(&g.asm, notEqualLabel)
	if notEqual {
		rtgAsmMovRaxImm(&g.asm, 1)
	} else {
		rtgAsmMovRaxImm(&g.asm, 0)
	}
	rtgAsmMarkLabel(&g.asm, endLabel)
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

func rtgEmitSliceBasePtrLenTokens(g *rtgLinearGen, p rtgProgram, start int, end int, ep rtgExprParse, idx int) bool {
	if start+1 == end && rtgTokIsKind(p, start, rtgTokIdent) {
		nameStart := p.toks[start].start
		nameEnd := p.toks[start].end
		localIndex := rtgFindLocalIndex(g, nameStart, nameEnd)
		if localIndex >= 0 {
			if !rtgTypeIsSlice(&g.meta, g.locals[localIndex].typ) {
				return false
			}
			rtgAsmLoadRaxStack(&g.asm, g.locals[localIndex].offset)
			rtgAsmLoadRcxStack(&g.asm, g.locals[localIndex].offset-8)
			return true
		}
		globalOffset := rtgFindGlobalOffset(g, nameStart, nameEnd)
		globalType := rtgFindGlobalType(g, nameStart, nameEnd)
		if globalOffset >= 0 && rtgTypeIsSlice(&g.meta, globalType) {
			rtgAsmLoadRaxBss(&g.asm, globalOffset+8)
			rtgAsmMovRcxRax(&g.asm)
			rtgAsmLoadRaxBss(&g.asm, globalOffset)
			return true
		}
		return false
	}
	if start+3 == end && rtgTokIsKind(p, start, rtgTokIdent) && rtgTokTextIs(p, start+1, ".") && rtgTokIsKind(p, start+2, rtgTokIdent) {
		localIndex := rtgFindLocalIndex(g, p.toks[start].start, p.toks[start].end)
		if localIndex < 0 {
			return false
		}
		fieldType := rtgStructFieldType(g, g.locals[localIndex].typ, p.toks[start+2].start, p.toks[start+2].end)
		if !rtgTypeIsSlice(&g.meta, fieldType) {
			return false
		}
		fieldOffset := rtgStructFieldOffset(g, g.locals[localIndex].typ, p.toks[start+2].start, p.toks[start+2].end)
		if fieldOffset < 0 {
			return false
		}
		t := rtgResolveType(&g.meta, g.locals[localIndex].typ)
		if t.kind == rtgTypePointer {
			rtgAsmLoadRdxStack(&g.asm, g.locals[localIndex].offset)
			if fieldOffset != 0 {
				rtgAsmAddRdxImm(&g.asm, fieldOffset)
			}
		} else {
			rtgAsmLeaRdxStack(&g.asm, g.locals[localIndex].offset-fieldOffset)
		}
		rtgAsmLoadRaxMemRdxDisp(&g.asm, 0)
		rtgAsmLoadRcxMemRdxDisp(&g.asm, 8)
		return true
	}
	return rtgEmitSlicePtrLen(g, ep, idx)
}

func rtgEmitSlicePtrLen(g *rtgLinearGen, ep rtgExprParse, idx int) bool {
	if idx < 0 || idx >= len(ep.exprs) {
		return false
	}
	e := ep.exprs[idx]
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 {
			globalOffset := rtgFindGlobalOffset(g, e.nameStart, e.nameEnd)
			globalType := rtgFindGlobalType(g, e.nameStart, e.nameEnd)
			if globalOffset < 0 || !rtgTypeIsSlice(&g.meta, globalType) {
				return false
			}
			rtgAsmLoadRaxBss(&g.asm, globalOffset+8)
			rtgAsmMovRcxRax(&g.asm)
			rtgAsmLoadRaxBss(&g.asm, globalOffset)
			return true
		}
		if !rtgTypeIsSlice(&g.meta, g.locals[localIndex].typ) {
			return false
		}
		rtgAsmLoadRaxStack(&g.asm, g.locals[localIndex].offset)
		rtgAsmLoadRcxStack(&g.asm, g.locals[localIndex].offset-8)
		return true
	}
	if e.kind == rtgExprComposite {
		sliceType := rtgCompositeExprType(g, ep, e)
		if !rtgTypeIsSlice(&g.meta, sliceType) {
			return false
		}
		if !rtgEmitSliceLiteralRegs(g, ep, e, sliceType) {
			return false
		}
		rtgAsmMovRcxRdx(&g.asm)
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
	elemSize := rtgTypeSize(&g.meta, sliceType.elem)
	rtgAsmImulRcxImm(&g.asm, elemSize)
	rtgAsmLoadQwordRaxIndexRcxDisp(&g.asm, fieldOffset)
	return true
}

func rtgEmitIndexAddressRax(g *rtgLinearGen, ep rtgExprParse, indexExpr rtgExpr) bool {
	leftType := rtgInferParsedExprType(g, ep, indexExpr.left)
	sliceType := rtgResolveType(&g.meta, leftType)
	if sliceType.kind != rtgTypeSlice {
		return false
	}
	elemSize := rtgTypeSize(&g.meta, sliceType.elem)
	if elemSize < 1 {
		elemSize = 8
	}
	if !rtgEmitIntExpr(g, ep, indexExpr.right) {
		return false
	}
	rtgAsmPushRax(&g.asm)
	if !rtgEmitSlicePtrLen(g, ep, indexExpr.left) {
		return false
	}
	rtgAsmPopRcx(&g.asm)
	if elemSize != 1 {
		rtgAsmImulRcxImm(&g.asm, elemSize)
	}
	rtgAsmAddRaxRcx(&g.asm)
	return true
}

func rtgEmitIndexExpr(g *rtgLinearGen, ep rtgExprParse, e rtgExpr) bool {
	left := ep.exprs[e.left]
	if left.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, left.nameStart, left.nameEnd)
		if localIndex < 0 {
			globalOffset := rtgFindGlobalOffset(g, left.nameStart, left.nameEnd)
			globalType := rtgFindGlobalType(g, left.nameStart, left.nameEnd)
			if globalOffset >= 0 && rtgTypeIsString(&g.meta, globalType) {
				if !rtgEmitIntExpr(g, ep, e.right) {
					return false
				}
				rtgAsmPushRax(&g.asm)
				rtgAsmLoadRaxBss(&g.asm, globalOffset)
				rtgAsmPopRcx(&g.asm)
				rtgAsmLoadByteRaxIndexRcx(&g.asm)
				return true
			}
			if globalOffset >= 0 && rtgTypeIsSlice(&g.meta, globalType) {
				t := rtgResolveType(&g.meta, globalType)
				elem := rtgResolveType(&g.meta, t.elem)
				if elem.kind != rtgTypeInt && elem.kind != rtgTypeByte && elem.kind != rtgTypeBool {
					return false
				}
				if !rtgEmitIntExpr(g, ep, e.right) {
					return false
				}
				rtgAsmPushRax(&g.asm)
				rtgAsmLoadRaxBss(&g.asm, globalOffset)
				rtgAsmPopRcx(&g.asm)
				if elem.kind == rtgTypeByte || elem.kind == rtgTypeBool {
					rtgAsmLoadByteRaxIndexRcx(&g.asm)
				} else {
					rtgAsmLoadQwordRaxIndexRcx8(&g.asm)
				}
				return true
			}
			constTok := rtgFindConstStringToken(g, left.nameStart, left.nameEnd)
			if constTok >= 0 {
				if !rtgEmitIntExpr(g, ep, e.right) {
					return false
				}
				msg := rtgDecodeStringToken(g.prog.src, g.prog.toks[constTok])
				msgOff := len(g.asm.data)
				for i := 0; i < len(msg); i++ {
					g.asm.data = append(g.asm.data, msg[i])
				}
				g.asm.data = append(g.asm.data, 0)
				rtgAsmPushRax(&g.asm)
				rtgAsmMovRaxDataAddr(&g.asm, msgOff)
				rtgAsmPopRcx(&g.asm)
				rtgAsmLoadByteRaxIndexRcx(&g.asm)
				return true
			}
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
			if elem.kind != rtgTypeInt && elem.kind != rtgTypeByte && elem.kind != rtgTypeBool {
				return false
			}
			if !rtgEmitIntExpr(g, ep, e.right) {
				return false
			}
			rtgAsmPushRax(&g.asm)
			rtgAsmLoadRaxStack(&g.asm, g.locals[localIndex].offset)
			rtgAsmPopRcx(&g.asm)
			if elem.kind == rtgTypeByte || elem.kind == rtgTypeBool {
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
		if t.kind == rtgTypeString {
			if !rtgEmitIntExpr(g, ep, e.right) {
				return false
			}
			rtgAsmPushRax(&g.asm)
			if !rtgEmitSelectorAddressRdx(g, ep, left) {
				return false
			}
			rtgAsmLoadRaxMemRdxDisp(&g.asm, 0)
			rtgAsmPopRcx(&g.asm)
			rtgAsmLoadByteRaxIndexRcx(&g.asm)
			return true
		}
		if t.kind == rtgTypeSlice {
			elem := rtgResolveType(&g.meta, t.elem)
			if elem.kind != rtgTypeInt && elem.kind != rtgTypeByte && elem.kind != rtgTypeBool {
				return false
			}
			if !rtgEmitIntExpr(g, ep, e.right) {
				return false
			}
			rtgAsmPushRax(&g.asm)
			if !rtgEmitSelectorAddressRdx(g, ep, left) {
				return false
			}
			rtgAsmLoadRaxMemRdxDisp(&g.asm, 0)
			rtgAsmPopRcx(&g.asm)
			if elem.kind == rtgTypeByte || elem.kind == rtgTypeBool {
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
	if e.kind == rtgExprCall && e.argCount == 1 && rtgExprIsIdentText(g.prog, ep, e.left, "string") {
		argIndex := ep.args[e.firstArg]
		arg := ep.exprs[argIndex]
		if arg.kind != rtgExprIdent {
			return false
		}
		localIndex := rtgFindLocalIndex(g, arg.nameStart, arg.nameEnd)
		if localIndex < 0 {
			return false
		}
		t := rtgResolveType(&g.meta, g.locals[localIndex].typ)
		if t.kind != rtgTypeSlice {
			return false
		}
		elem := rtgResolveType(&g.meta, t.elem)
		if elem.kind != rtgTypeByte {
			return false
		}
		rtgAsmLoadRaxStack(&g.asm, g.locals[localIndex].offset)
		return true
	}
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			if !rtgTypeIsString(&g.meta, g.locals[localIndex].typ) {
				return false
			}
			rtgAsmLoadRaxStack(&g.asm, g.locals[localIndex].offset)
			return true
		}
		globalOffset := rtgFindGlobalOffset(g, e.nameStart, e.nameEnd)
		globalType := rtgFindGlobalType(g, e.nameStart, e.nameEnd)
		if globalOffset >= 0 && rtgTypeIsString(&g.meta, globalType) {
			rtgAsmLoadRaxBss(&g.asm, globalOffset)
			return true
		}
		constTok := rtgFindConstStringToken(g, e.nameStart, e.nameEnd)
		if constTok >= 0 {
			msg := rtgDecodeStringToken(g.prog.src, g.prog.toks[constTok])
			msgOff := len(g.asm.data)
			for i := 0; i < len(msg); i++ {
				g.asm.data = append(g.asm.data, msg[i])
			}
			g.asm.data = append(g.asm.data, 0)
			rtgAsmMovRaxDataAddr(&g.asm, msgOff)
			return true
		}
		return false
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
	for i := len(g.locals) - 1; i >= 0; i-- {
		if rtgBytesEqualRange(g.prog.src, g.locals[i].nameStart, g.locals[i].nameEnd, nameStart, nameEnd) {
			return g.locals[i].offset
		}
	}
	return -1
}

func rtgFindLocalIndex(g *rtgLinearGen, nameStart int, nameEnd int) int {
	for i := len(g.locals) - 1; i >= 0; i-- {
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
	if base.kind == rtgExprIndex {
		leftType := rtgInferParsedExprType(g, ep, base.left)
		sliceType := rtgResolveType(&g.meta, leftType)
		elemType := rtgResolveType(&g.meta, sliceType.elem)
		if sliceType.kind != rtgTypeSlice || elemType.kind != rtgTypeStruct {
			return false
		}
		if !rtgEmitIntExpr(g, ep, base.right) {
			return false
		}
		rtgAsmPushRax(&g.asm)
		if !rtgEmitSlicePtrLen(g, ep, base.left) {
			return false
		}
		rtgAsmPopRcx(&g.asm)
		elemSize := rtgTypeSize(&g.meta, sliceType.elem)
		rtgAsmImulRcxImm(&g.asm, elemSize)
		rtgAsmMovRdxRax(&g.asm)
		rtgAsmAddRdxRcx(&g.asm)
		if fieldOffset != 0 {
			rtgAsmAddRdxImm(&g.asm, fieldOffset)
		}
		return true
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
	t := rtgResolveType(&g.meta, typ)
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
	t := rtgResolveType(&g.meta, typ)
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

func rtgFindGlobalType(g *rtgLinearGen, nameStart int, nameEnd int) int {
	for i := 0; i < len(g.meta.globals); i++ {
		s := g.meta.globals[i]
		if s.kind == rtgTokVar && rtgBytesEqualRange(g.prog.src, s.nameStart, s.nameEnd, nameStart, nameEnd) {
			return s.typ
		}
	}
	return 0
}

func rtgFindConstStringToken(g *rtgLinearGen, nameStart int, nameEnd int) int {
	for i := 0; i < len(g.meta.globals); i++ {
		s := g.meta.globals[i]
		if s.kind == rtgTokConst && rtgBytesEqualRange(g.prog.src, s.nameStart, s.nameEnd, nameStart, nameEnd) {
			if s.initStart+1 == s.initEnd && rtgTokIsKind(g.prog, s.initStart, rtgTokString) {
				return s.initStart
			}
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
		backingSize := 8388608
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
	builtin := rtgEvalBuiltinConst(g, nameStart, nameEnd)
	if builtin.ok {
		return builtin
	}
	for i := 0; i < len(g.meta.globals); i++ {
		s := g.meta.globals[i]
		if s.kind == rtgTokConst && rtgBytesEqualRange(g.prog.src, s.nameStart, s.nameEnd, nameStart, nameEnd) {
			ep := rtgParseExpression(g.prog, s.initStart, s.initEnd)
			if !ep.ok || len(ep.exprs) == 0 {
				var r rtgConstResult
				return r
			}
			rootIndex := len(ep.exprs) - 1
			result := rtgEvalConstExpr(g, ep, rootIndex)
			return result
		}
	}
	var r rtgConstResult
	return r
}

func rtgConstResultOk(value int) rtgConstResult {
	var r rtgConstResult
	r.value = value
	r.ok = true
	return r
}

func rtgEvalBuiltinConst(g *rtgLinearGen, nameStart int, nameEnd int) rtgConstResult {
	if rtgBytesEqualText(g.prog.src, nameStart, nameEnd, "O_RDONLY") {
		return rtgConstResultOk(0)
	}
	if rtgBytesEqualText(g.prog.src, nameStart, nameEnd, "O_WRONLY") {
		return rtgConstResultOk(1)
	}
	if rtgBytesEqualText(g.prog.src, nameStart, nameEnd, "O_RDWR") {
		return rtgConstResultOk(2)
	}
	if rtgBytesEqualText(g.prog.src, nameStart, nameEnd, "O_CREATE") {
		return rtgConstResultOk(64)
	}
	if rtgBytesEqualText(g.prog.src, nameStart, nameEnd, "O_TRUNC") {
		return rtgConstResultOk(512)
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
		value := rtgParseIntToken(p.src, p.toks[e.tok])
		return rtgConstResultOk(value)
	}
	if e.kind == rtgExprChar {
		value := rtgParseCharToken(p.src, p.toks[e.tok])
		return rtgConstResultOk(value)
	}
	if e.kind == rtgExprBool {
		if rtgBytesEqualText(p.src, p.toks[e.tok].start, p.toks[e.tok].end, "true") {
			return rtgConstResultOk(1)
		}
		return rtgConstResultOk(0)
	}
	if e.kind == rtgExprIdent {
		result := rtgEvalConstByName(g, e.nameStart, e.nameEnd)
		return result
	}
	if e.kind == rtgExprCall {
		if e.argCount == 1 && rtgExprIsIdentText(p, ep, e.left, "int") || e.argCount == 1 && rtgExprIsIdentText(p, ep, e.left, "byte") || e.argCount == 1 && rtgExprIsIdentText(p, ep, e.left, "int64") {
			result := rtgEvalConstExpr(g, ep, ep.args[e.firstArg])
			return result
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
			return rtgConstResultOk(-inner.value)
		}
		if rtgTokTextIs(p, e.tok, "+") {
			return rtgConstResultOk(inner.value)
		}
		if rtgTokTextIs(p, e.tok, "!") {
			if inner.value == 0 {
				return rtgConstResultOk(1)
			}
			return rtgConstResultOk(0)
		}
		var r rtgConstResult
		return r
	}
	if e.kind == rtgExprBinary {
		rightExpr := ep.exprs[e.right]
		rightKind := rightExpr.kind
		rightTok := rightExpr.tok
		left := rtgEvalConstExpr(g, ep, e.left)
		if !left.ok {
			var r rtgConstResult
			return r
		}
		if rtgTokTextIs(p, e.tok, "&&") {
			if left.value == 0 {
				return rtgConstResultOk(0)
			}
			right := rtgEvalConstExpr(g, ep, e.right)
			if !right.ok {
				var r rtgConstResult
				return r
			}
			if right.value != 0 {
				return rtgConstResultOk(1)
			}
			return rtgConstResultOk(0)
		}
		if rtgTokTextIs(p, e.tok, "||") {
			if left.value != 0 {
				return rtgConstResultOk(1)
			}
			right := rtgEvalConstExpr(g, ep, e.right)
			if !right.ok {
				var r rtgConstResult
				return r
			}
			if right.value != 0 {
				return rtgConstResultOk(1)
			}
			return rtgConstResultOk(0)
		}
		var right rtgConstResult
		if rightKind == rtgExprInt {
			value := rtgParseIntToken(p.src, p.toks[rightTok])
			right = rtgConstResultOk(value)
		} else if rightKind == rtgExprChar {
			value := rtgParseCharToken(p.src, p.toks[rightTok])
			right = rtgConstResultOk(value)
		} else if rightKind == rtgExprBool {
			if rtgBytesEqualText(p.src, p.toks[rightTok].start, p.toks[rightTok].end, "true") {
				right = rtgConstResultOk(1)
			} else {
				right = rtgConstResultOk(0)
			}
		} else {
			right = rtgEvalConstExpr(g, ep, e.right)
		}
		if !right.ok {
			var r rtgConstResult
			return r
		}
		result := rtgEvalConstBinary(g, e.tok, left.value, right.value)
		return result
	}
	var r rtgConstResult
	return r
}

func rtgEvalConstBinary(g *rtgLinearGen, tok int, left int, right int) rtgConstResult {
	p := g.prog
	if rtgTokTextIs(p, tok, "+") {
		var r rtgConstResult
		r.value = left + right
		r.ok = true
		return r
	}
	if rtgTokTextIs(p, tok, "-") {
		return rtgConstResultOk(left - right)
	}
	if rtgTokTextIs(p, tok, "*") {
		var r rtgConstResult
		r.value = left * right
		r.ok = true
		return r
	}
	if rtgTokTextIs(p, tok, "/") {
		if right == 0 {
			var r rtgConstResult
			return r
		}
		return rtgConstResultOk(left / right)
	}
	if rtgTokTextIs(p, tok, "%") {
		if right == 0 {
			var r rtgConstResult
			return r
		}
		return rtgConstResultOk(left % right)
	}
	if rtgTokTextIs(p, tok, "&") {
		return rtgConstResultOk(left & right)
	}
	if rtgTokTextIs(p, tok, "|") {
		return rtgConstResultOk(left | right)
	}
	if rtgTokTextIs(p, tok, "^") {
		return rtgConstResultOk(left ^ right)
	}
	if rtgTokTextIs(p, tok, "&^") {
		return rtgConstResultOk(left &^ right)
	}
	if rtgTokTextIs(p, tok, "<<") {
		return rtgConstResultOk(left << right)
	}
	if rtgTokTextIs(p, tok, ">>") {
		return rtgConstResultOk(left >> right)
	}
	if rtgTokTextIs(p, tok, "==") {
		if left == right {
			return rtgConstResultOk(1)
		}
		return rtgConstResultOk(0)
	}
	if rtgTokTextIs(p, tok, "!=") {
		if left != right {
			return rtgConstResultOk(1)
		}
		return rtgConstResultOk(0)
	}
	if rtgTokTextIs(p, tok, "<") {
		if left < right {
			return rtgConstResultOk(1)
		}
		return rtgConstResultOk(0)
	}
	if rtgTokTextIs(p, tok, "<=") {
		if left <= right {
			return rtgConstResultOk(1)
		}
		return rtgConstResultOk(0)
	}
	if rtgTokTextIs(p, tok, ">") {
		if left > right {
			return rtgConstResultOk(1)
		}
		return rtgConstResultOk(0)
	}
	if rtgTokTextIs(p, tok, ">=") {
		if left >= right {
			return rtgConstResultOk(1)
		}
		return rtgConstResultOk(0)
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
	var code []byte
	var labelPos []int
	var labelSet []bool
	var relocs []rtgLabelRef
	var dataRelocs []rtgDataRef
	var absRelocs []rtgAbsRef
	var data []byte
	a.code = code
	a.labelPos = labelPos
	a.labelSet = labelSet
	a.relocs = relocs
	a.dataRelocs = dataRelocs
	a.absRelocs = absRelocs
	a.data = data
	a.bssSize = 0
	a.imageBase = 0x400000
	a.codeOffset = 64 + 56
	a.dataOffset = 0
}

func rtgAsmNewLabel(a *rtgAsm) int {
	a.labelPos = append(a.labelPos, 0)
	a.labelSet = append(a.labelSet, false)
	label := len(a.labelPos) - 1
	return label
}

func rtgAsmMarkLabel(a *rtgAsm, label int) {
	if label >= 0 && label < len(a.labelPos) {
		codeLen := len(a.code)
		a.labelPos[label] = codeLen
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

func rtgAsmBuildArgvEnvSlices(a *rtgAsm, bssOff int, envOff int, envLenOff int) {
	loopLabel := rtgAsmNewLabel(a)
	strlenLabel := rtgAsmNewLabel(a)
	afterLenLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	envScanLabel := rtgAsmNewLabel(a)
	envStartLabel := rtgAsmNewLabel(a)
	envLoopLabel := rtgAsmNewLabel(a)
	envStrlenLabel := rtgAsmNewLabel(a)
	envAfterLenLabel := rtgAsmNewLabel(a)
	envDoneLabel := rtgAsmNewLabel(a)
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x8b)
	rtgAsmEmit8(a, 0x04)
	rtgAsmEmit8(a, 0x24)
	rtgAsmEmit8(a, 0x49)
	rtgAsmEmit8(a, 0x89)
	rtgAsmEmit8(a, 0xc0)
	rtgAsmEmit8(a, 0x49)
	rtgAsmEmit8(a, 0xff)
	rtgAsmEmit8(a, 0xc8)
	rtgAsmEmit8(a, 0x4c)
	rtgAsmEmit8(a, 0x8d)
	rtgAsmEmit8(a, 0x4c)
	rtgAsmEmit8(a, 0x24)
	rtgAsmEmit8(a, 0x10)
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
	rtgAsmEmit8(a, 0x8d)
	rtgAsmEmit8(a, 0x4c)
	rtgAsmEmit8(a, 0x24)
	rtgAsmEmit8(a, 0x08)
	rtgAsmMarkLabel(a, envScanLabel)
	rtgAsmEmit8(a, 0x49)
	rtgAsmEmit8(a, 0x83)
	rtgAsmEmit8(a, 0x39)
	rtgAsmEmit8(a, 0x00)
	rtgAsmJzLabel(a, envStartLabel)
	rtgAsmEmit8(a, 0x49)
	rtgAsmEmit8(a, 0x83)
	rtgAsmEmit8(a, 0xc1)
	rtgAsmEmit8(a, 0x08)
	rtgAsmJmpLabel(a, envScanLabel)
	rtgAsmMarkLabel(a, envStartLabel)
	rtgAsmEmit8(a, 0x49)
	rtgAsmEmit8(a, 0x83)
	rtgAsmEmit8(a, 0xc1)
	rtgAsmEmit8(a, 0x08)
	rtgAsmMovR10BssAddr(a, envOff)
	rtgAsmEmit8(a, 0x4d)
	rtgAsmEmit8(a, 0x31)
	rtgAsmEmit8(a, 0xdb)
	rtgAsmMarkLabel(a, envLoopLabel)
	rtgAsmEmit8(a, 0x4b)
	rtgAsmEmit8(a, 0x8b)
	rtgAsmEmit8(a, 0x3c)
	rtgAsmEmit8(a, 0xd9)
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x83)
	rtgAsmEmit8(a, 0xff)
	rtgAsmEmit8(a, 0x00)
	rtgAsmJzLabel(a, envDoneLabel)
	rtgAsmEmit8(a, 0x49)
	rtgAsmEmit8(a, 0x89)
	rtgAsmEmit8(a, 0x3a)
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x31)
	rtgAsmEmit8(a, 0xc0)
	rtgAsmMarkLabel(a, envStrlenLabel)
	rtgAsmEmit8(a, 0x80)
	rtgAsmEmit8(a, 0x3c)
	rtgAsmEmit8(a, 0x07)
	rtgAsmEmit8(a, 0x00)
	rtgAsmJzLabel(a, envAfterLenLabel)
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0xff)
	rtgAsmEmit8(a, 0xc0)
	rtgAsmJmpLabel(a, envStrlenLabel)
	rtgAsmMarkLabel(a, envAfterLenLabel)
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
	rtgAsmJmpLabel(a, envLoopLabel)
	rtgAsmMarkLabel(a, envDoneLabel)
	rtgAsmMovRaxR11(a)
	rtgAsmStoreRaxBss(a, envLenOff)

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

func rtgAsmMovRcxRdx(a *rtgAsm) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x89)
	rtgAsmEmit8(a, 0xd1)
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

func rtgAsmMovRaxR11(a *rtgAsm) {
	rtgAsmEmit8(a, 0x4c)
	rtgAsmEmit8(a, 0x89)
	rtgAsmEmit8(a, 0xd8)
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

func rtgAsmStoreRaxMemRcxDisp(a *rtgAsm, disp int) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0x89)
	rtgAsmEmit8(a, 0x81)
	rtgAsmEmit32(a, disp)
}

func rtgAsmStoreAlMemRdxRcx1(a *rtgAsm) {
	rtgAsmEmit8(a, 0x88)
	rtgAsmEmit8(a, 0x04)
	rtgAsmEmit8(a, 0x0a)
}

func rtgAsmStoreAlMemRdxDisp(a *rtgAsm, disp int) {
	rtgAsmEmit8(a, 0x88)
	rtgAsmEmit8(a, 0x82)
	rtgAsmEmit32(a, disp)
}

func rtgAsmIncRcx(a *rtgAsm) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0xff)
	rtgAsmEmit8(a, 0xc1)
}

func rtgAsmIncRax(a *rtgAsm) {
	rtgAsmEmit8(a, 0x48)
	rtgAsmEmit8(a, 0xff)
	rtgAsmEmit8(a, 0xc0)
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

func rtgPut32At(out []byte, at int, v int) {
	b0 := byte(v)
	b1 := byte(v >> 8)
	b2 := byte(v >> 16)
	b3 := byte(v >> 24)
	out[at] = b0
	out[at+1] = b1
	out[at+2] = b2
	out[at+3] = b3
}

func rtgPut64At(out []byte, at int, v int) {
	b0 := byte(v)
	b1 := byte(v >> 8)
	b2 := byte(v >> 16)
	b3 := byte(v >> 24)
	b4 := byte(v >> 32)
	b5 := byte(v >> 40)
	b6 := byte(v >> 48)
	b7 := byte(v >> 56)
	out[at] = b0
	out[at+1] = b1
	out[at+2] = b2
	out[at+3] = b3
	out[at+4] = b4
	out[at+5] = b5
	out[at+6] = b6
	out[at+7] = b7
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
			var compositeFields []rtgCompositeField
			ep.pos++
			for ep.ok && ep.pos < ep.end && !rtgTokTextIs(ep.prog, ep.pos, "}") {
				var field rtgCompositeField
				if rtgTokIsKind(ep.prog, ep.pos, rtgTokIdent) && rtgTokTextIs(ep.prog, ep.pos+1, ":") {
					nameTok := ep.prog.toks[ep.pos]
					ep.pos += 2
					fieldEnd := rtgFindExprBoundary(ep.prog, ep.pos, ep.end)
					oldEnd := ep.end
					ep.end = fieldEnd
					fieldRoot := rtgParseBinaryExpr(ep, 1)
					ep.end = oldEnd
					field.nameStart = nameTok.start
					field.nameEnd = nameTok.end
					field.expr = fieldRoot
					ep.pos = fieldEnd
				} else if rtgTokTextIs(ep.prog, ep.pos, "{") {
					fieldEnd := rtgFindExprBoundary(ep.prog, ep.pos, ep.end)
					oldEnd := ep.end
					ep.end = fieldEnd
					field.expr = rtgParseImplicitCompositeExpr(ep)
					ep.end = oldEnd
					ep.pos = fieldEnd
				} else {
					fieldEnd := rtgFindExprBoundary(ep.prog, ep.pos, ep.end)
					oldEnd := ep.end
					ep.end = fieldEnd
					field.expr = rtgParseBinaryExpr(ep, 1)
					ep.end = oldEnd
					ep.pos = fieldEnd
				}
				compositeFields = append(compositeFields, field)
				if rtgTokTextIs(ep.prog, ep.pos, ",") {
					ep.pos++
				}
			}
			if !rtgTokTextIs(ep.prog, ep.pos, "}") {
				ep.ok = false
				return left
			}
			ep.pos++
			first := len(ep.fields)
			for i := 0; i < len(compositeFields); i++ {
				field := compositeFields[i]
				ep.fields = append(ep.fields, field)
			}
			count := len(compositeFields)
			left = rtgAddExpr(ep, rtgExpr{kind: rtgExprComposite, tok: base.tok, firstArg: first, argCount: count, nameStart: base.nameStart, nameEnd: base.nameEnd})
			continue
		}
		if rtgTokTextIs(ep.prog, ep.pos, "(") {
			callTok := ep.pos
			var callArgs []int
			callExpanded := false
			ep.pos++
			for ep.ok && ep.pos < ep.end && !rtgTokTextIs(ep.prog, ep.pos, ")") {
				argEnd := rtgFindExprBoundary(ep.prog, ep.pos, ep.end)
				if rtgTokTextIs(ep.prog, argEnd, "{") {
					closeTok := rtgSkipBalanced(ep.prog, argEnd, "{", "}")
					if closeTok > argEnd {
						argEnd = closeTok
					}
				}
				parseEnd := argEnd
				if rtgHasTrailingEllipsis(ep.prog, ep.pos, argEnd) {
					callExpanded = true
					parseEnd = argEnd - 3
				}
				oldEnd := ep.end
				ep.end = parseEnd
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
			expanded := 0
			if callExpanded {
				expanded = 1
			}
			left = rtgAddExpr(ep, rtgExpr{kind: rtgExprCall, tok: callTok, left: left, firstArg: first, argCount: count, nameStart: expanded})
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

func rtgHasTrailingEllipsis(p rtgProgram, start int, end int) bool {
	if end-start < 4 {
		return false
	}
	if !rtgTokTextIs(p, end-3, ".") {
		return false
	}
	if !rtgTokTextIs(p, end-2, ".") {
		return false
	}
	if !rtgTokTextIs(p, end-1, ".") {
		return false
	}
	return true
}

func rtgParseImplicitCompositeExpr(ep *rtgExprParse) int {
	openTok := ep.pos
	if !rtgTokTextIs(ep.prog, ep.pos, "{") {
		ep.ok = false
		return 0
	}
	var compositeFields []rtgCompositeField
	ep.pos++
	for ep.ok && ep.pos < ep.end && !rtgTokTextIs(ep.prog, ep.pos, "}") {
		if !rtgTokIsKind(ep.prog, ep.pos, rtgTokIdent) || !rtgTokTextIs(ep.prog, ep.pos+1, ":") {
			ep.ok = false
			return 0
		}
		nameTok := ep.prog.toks[ep.pos]
		ep.pos += 2
		fieldEnd := rtgFindExprBoundary(ep.prog, ep.pos, ep.end)
		oldEnd := ep.end
		ep.end = fieldEnd
		fieldRoot := rtgParseBinaryExpr(ep, 1)
		ep.end = oldEnd
		compositeFields = append(compositeFields, rtgCompositeField{nameStart: nameTok.start, nameEnd: nameTok.end, expr: fieldRoot})
		ep.pos = fieldEnd
		if rtgTokTextIs(ep.prog, ep.pos, ",") {
			ep.pos++
		}
	}
	if !rtgTokTextIs(ep.prog, ep.pos, "}") {
		ep.ok = false
		return 0
	}
	ep.pos++
	first := len(ep.fields)
	for i := 0; i < len(compositeFields); i++ {
		field := compositeFields[i]
		ep.fields = append(ep.fields, field)
	}
	count := len(compositeFields)
	return rtgAddExpr(ep, rtgExpr{kind: rtgExprComposite, tok: openTok, firstArg: first, argCount: count})
}

func rtgParsePrimaryExpr(ep *rtgExprParse) int {
	if ep.pos >= ep.end {
		ep.ok = false
		return 0
	}
	tok := ep.prog.toks[ep.pos]
	if rtgTokTextIs(ep.prog, ep.pos, "[") && rtgTokTextIs(ep.prog, ep.pos+1, "]") && rtgTokIsKind(ep.prog, ep.pos+2, rtgTokIdent) {
		startTok := ep.pos
		nameTok := ep.prog.toks[ep.pos+2]
		ep.pos += 3
		return rtgAddExpr(ep, rtgExpr{kind: rtgExprIdent, tok: startTok, nameStart: ep.prog.toks[startTok].start, nameEnd: nameTok.end})
	}
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
	index := len(ep.exprs) - 1
	return index
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
	var stmts []rtgStmt
	bp.prog = p
	bp.stmts = stmts
	bp.ok = true
	rtgParseStatements(&bp, fn.bodyStart+1, fn.bodyEnd)
	return bp
}

func rtgParseStatements(bp *rtgBodyParse, start int, end int) int {
	i := start
	for bp.ok && i < end {
		if i < 0 || i >= len(bp.prog.toks) {
			return i
		}
		if rtgTokTextIs(bp.prog, i, ";") {
			i++
			continue
		}
		if bp.prog.toks[i].start < 0 || bp.prog.toks[i].start >= len(bp.prog.src) {
			return i
		}
		if rtgTokIsKind(bp.prog, i, rtgTokEOF) {
			return i
		}
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
	if rtgTokIsKeyword(p, start, rtgTokSwitch) {
		bodyStart := rtgFindNextTokenText(p, start+1, end, "{")
		if bodyStart <= start {
			return start
		}
		bodyEnd := rtgFindMatchingBrace(p, bodyStart, end)
		if bodyEnd <= bodyStart {
			return start
		}
		bp.stmts = append(bp.stmts, rtgStmt{kind: rtgStmtSwitch, startTok: start, endTok: bodyEnd + 1, exprStart: start + 1, exprEnd: bodyStart, bodyStart: bodyStart + 1, bodyEnd: bodyEnd})
		return bodyEnd + 1
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
	if rtgTokIsKeyword(p, start, rtgTokVar) || rtgTokIsKeyword(p, start, rtgTokConst) {
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
	if !(rtgTokIsKeyword(p, start, rtgTokIf)) {
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
		if i > start && paren == 0 && brack == 0 && brace == 0 {
			if rtgTokIsKind(p, i, rtgTokEOF) {
				return i
			}
			if rtgTokTextIs(p, i, ";") {
				return i
			}
			if rtgTokIsKeyword(p, i, rtgTokReturn) || rtgTokIsKeyword(p, i, rtgTokIf) || rtgTokIsKeyword(p, i, rtgTokFor) || rtgTokIsKeyword(p, i, rtgTokSwitch) || rtgTokIsKeyword(p, i, rtgTokCase) || rtgTokIsKeyword(p, i, rtgTokDefault) || rtgTokIsKeyword(p, i, rtgTokVar) || rtgTokIsKeyword(p, i, rtgTokConst) || rtgTokIsKeyword(p, i, rtgTokBreak) || rtgTokIsKeyword(p, i, rtgTokContinue) || rtgTokIsKeyword(p, i, rtgTokGoto) {
				return i
			}
		}
		closed := false
		if rtgTokTextIs(p, i, "(") {
			paren++
		} else if rtgTokTextIs(p, i, ")") {
			paren--
			closed = true
		} else if rtgTokTextIs(p, i, "[") {
			brack++
		} else if rtgTokTextIs(p, i, "]") {
			brack--
			closed = true
		} else if rtgTokTextIs(p, i, "{") {
			brace++
		} else if rtgTokTextIs(p, i, "}") {
			if brace == 0 {
				return i
			}
			brace--
			closed = true
		}
		if i > start && p.toks[i].line != line && paren == 0 && brack == 0 && brace == 0 {
			if closed {
				return i + 1
			}
			return i
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

func rtgBuildMeta(pp *rtgProgram) rtgMeta {
	var p rtgProgram
	p.src = pp.src
	p.toks = pp.toks
	p.decls = pp.decls
	p.funcs = pp.funcs
	p.ok = pp.ok
	var m rtgMeta
	m.src = p.src
	m.prog.src = p.src
	m.prog.toks = p.toks
	m.prog.decls = p.decls
	m.prog.funcs = p.funcs
	m.prog.ok = p.ok
	m.ok = true
	rtgInitBuiltinTypes(&m)

	for i := 0; i < len(p.decls); i++ {
		decl := p.decls[i]
		if decl.kind != rtgTokType && decl.kind != rtgTokVar && decl.kind != rtgTokConst {
			continue
		}
		entryStart := decl.startTok + 1
		if rtgTokTextIs(p, entryStart, "(") {
			groupEnd := decl.endTok
			j := entryStart + 1
			for j < groupEnd-1 {
				if rtgTokIsKind(p, j, rtgTokIdent) {
					entryEnd := rtgStatementLineEnd(p, j, groupEnd-1)
					rtgParseTopDeclEntry(&m, p, decl.kind, j, entryEnd)
					if entryEnd <= j {
						j++
					} else {
						j = entryEnd
					}
				} else {
					j++
				}
			}
			continue
		}
		rtgParseTopDeclEntry(&m, p, decl.kind, entryStart, decl.endTok)
	}
	for i := 0; i < len(p.funcs); i++ {
		rtgParseFuncInfo(&m, i)
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

func rtgParseTopDeclEntry(m *rtgMeta, p rtgProgram, kind int, start int, end int) {
	if start >= end || !rtgTokIsKind(p, start, rtgTokIdent) {
		m.ok = false
		return
	}
	name := p.toks[start]
	if kind == rtgTokType {
		typeResult := rtgParseType(m, p, start+1, end)
		if typeResult.typ == 0 || typeResult.next > end {
			m.ok = false
			return
		}
		if m.types[typeResult.typ].kind == rtgTypeStruct || m.types[typeResult.typ].kind == rtgTypePointer || m.types[typeResult.typ].kind == rtgTypeSlice {
			m.types[typeResult.typ].nameStart = name.start
			m.types[typeResult.typ].nameEnd = name.end
		} else {
			size := rtgTypeSize(m, typeResult.typ)
			typ := rtgTypeInfo{kind: rtgTypeNamed, elem: typeResult.typ, size: size, nameStart: name.start, nameEnd: name.end}
			rtgAddType(m, typ)
		}
		return
	}
	eq := start
	j := start + 1
	for j < end {
		if j >= 0 && j < len(p.toks) {
			tok := p.toks[j]
			if tok.kind == rtgTokOp && tok.end-tok.start == 1 && p.src[tok.start] == '=' {
				eq = j
				j = end
				continue
			}
		}
		j++
	}
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
		typeResult := rtgParseType(m, p, start+1, typeEnd)
		typ = typeResult.typ
	}
	if typ == 0 && initStart < initEnd {
		typ = rtgInferTopLiteralType(m, p, initStart, initEnd)
	}
	m.globals = append(m.globals, rtgSymbolInfo{nameStart: name.start, nameEnd: name.end, kind: kind, typ: typ, initStart: initStart, initEnd: initEnd})
}

func rtgInferTopLiteralType(m *rtgMeta, p rtgProgram, start int, end int) int {
	if start+1 == end && rtgTokIsKind(p, start, rtgTokString) {
		return rtgTypeString
	}
	open := start
	depth := 0
	for open < end {
		if depth == 0 && rtgTokTextIs(p, open, "{") {
			typeResult := rtgParseType(m, p, start, open)
			if typeResult.typ != 0 && rtgTypeIsSlice(m, typeResult.typ) {
				return typeResult.typ
			}
			return 0
		}
		if rtgTokTextIs(p, open, "(") || rtgTokTextIs(p, open, "[") {
			depth++
		} else if rtgTokTextIs(p, open, ")") || rtgTokTextIs(p, open, "]") {
			if depth > 0 {
				depth--
			}
		}
		open++
	}
	return 0
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
	rtgParseParamList(m, p, lparen+1, rparen, &paramCount)
	resultType := 0
	if rparen+1 < fn.bodyStart {
		typeResult := rtgParseType(m, p, rparen+1, fn.bodyStart)
		resultType = typeResult.typ
	}
	m.funcs = append(m.funcs, rtgFuncInfo{declIndex: fnIndex, nameStart: nameStart, nameEnd: nameEnd, firstParam: firstParam, paramCount: paramCount, resultType: resultType, bodyStart: fn.bodyStart + 1, bodyEnd: fn.bodyEnd})
}

func rtgParseParamList(m *rtgMeta, p rtgProgram, start int, end int, count *int) {
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
		typeResult := rtgParseType(m, p, typeStart, entryEnd)
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

func rtgParseType(m *rtgMeta, p rtgProgram, start int, end int) rtgTypeResult {
	if start >= end {
		return rtgTypeResult{next: start}
	}
	if rtgTokTextIs(p, start, "*") {
		elem := rtgParseType(m, p, start+1, end)
		if elem.typ == 0 {
			return rtgTypeResult{next: start}
		}
		info := rtgTypeInfo{kind: rtgTypePointer, elem: elem.typ, size: 8}
		typ := rtgAddType(m, info)
		return rtgTypeResult{typ: typ, next: elem.next}
	}
	if rtgTokTextIs(p, start, "[") && rtgTokTextIs(p, start+1, "]") {
		elem := rtgParseType(m, p, start+2, end)
		if elem.typ == 0 {
			return rtgTypeResult{next: start}
		}
		info := rtgTypeInfo{kind: rtgTypeSlice, elem: elem.typ, size: 24}
		typ := rtgAddType(m, info)
		return rtgTypeResult{typ: typ, next: elem.next}
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
				fieldType := rtgParseType(m, p, i+1, lineEnd)
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
		size := rtgAlignTo8(offset)
		info := rtgTypeInfo{kind: rtgTypeStruct, first: firstField, count: count, size: size}
		typ := rtgAddType(m, info)
		return rtgTypeResult{typ: typ, next: closeTok + 1}
	}
	if rtgTokIsKind(p, start, rtgTokIdent) {
		tok := p.toks[start]
		builtin := rtgBuiltinTypeFromToken(p, tok)
		if builtin != 0 {
			return rtgTypeResult{typ: builtin, next: start + 1}
		}
		return rtgTypeResult{typ: rtgNamedTypeFromToken(m, p, tok), next: start + 1}
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

func rtgNamedTypeFromToken(m *rtgMeta, p rtgProgram, tok rtgToken) int {
	for i := 0; i < len(m.types); i++ {
		if m.types[i].nameEnd > m.types[i].nameStart && rtgBytesEqualRange(p.src, m.types[i].nameStart, m.types[i].nameEnd, tok.start, tok.end) {
			return i
		}
	}
	info := rtgTypeInfo{kind: rtgTypeNamed, size: 8, nameStart: tok.start, nameEnd: tok.end}
	return rtgAddType(m, info)
}

func rtgAddType(m *rtgMeta, typ rtgTypeInfo) int {
	m.types = append(m.types, typ)
	index := len(m.types) - 1
	return index
}

func rtgTypeSize(m *rtgMeta, typ int) int {
	t := rtgResolveType(m, typ)
	if t.size > 0 {
		return t.size
	}
	return 8
}

func rtgResolveType(m *rtgMeta, typ int) rtgTypeInfo {
	if typ >= 0 && typ < len(m.types) {
		t := m.types[typ]
		if t.kind == rtgTypeNamed && t.elem > 0 && t.elem < len(m.types) {
			return m.types[t.elem]
		}
		if t.kind == rtgTypeNamed && t.elem == 0 && t.nameEnd > t.nameStart {
			for i := 0; i < len(m.types); i++ {
				other := m.types[i]
				if i != typ && other.nameEnd > other.nameStart && rtgBytesEqualRange(m.src, other.nameStart, other.nameEnd, t.nameStart, t.nameEnd) {
					if other.kind != rtgTypeNamed || other.elem > 0 {
						return rtgResolveType(m, i)
					}
				}
			}
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

func rtgFindFuncInfoByName(meta *rtgMeta, name string) int {
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

func rtgFindAssignTokenInRange(p rtgProgram, start int, end int) int {
	i := start
	for i < end {
		if i >= 0 && i < len(p.toks) {
			tok := p.toks[i]
			if tok.start >= 0 && tok.start < len(p.src) && tok.end-tok.start == 1 && p.src[tok.start] == '=' {
				return i
			}
		}
		i++
	}
	return start
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

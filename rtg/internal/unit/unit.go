//go:build rtg

package unit

type Import struct {
	NameTok int
	PathTok int
}

type Symbol struct {
	Name    string
	Package int
	Token   int
}

type Call struct {
	Kind      int
	CalleeTok int
	BaseTok   int
	DotTok    int
}

type NameRef struct {
	Kind    int
	Token   int
	Index   int
	Package int
}

type Selector struct {
	BaseTok     int
	DotTok      int
	NameTok     int
	BaseKind    int
	BaseIndex   int
	BasePackage int
	Package     int
	Symbol      int
}

type TypeRef struct {
	Kind    int
	Token   int
	BaseTok int
	DotTok  int
	Package int
	Symbol  int
}

type Program struct {
	Package    string
	ImportPath string
	Text       []byte
	Tokens     []Token
	Imports    []Import
	Symbols    []Symbol
	Decls      []Decl
	Funcs      []Func
	TypeRefs   []TypeRef
	Calls      []Call
	Refs       []NameRef
	Selectors  []Selector
}

func Marshal(program Program) ([]byte, bool) {
	capacity := 50 + len(program.Package) + len(program.ImportPath) + len(program.Text) + len(program.Tokens)*5 + len(program.Decls)*8 + len(program.Funcs)*12
	out := make([]byte, 0, capacity)
	for i := 0; i < len(Magic); i++ {
		out = append(out, Magic[i])
	}
	out = appendUint16(out, Version)
	out = appendUint16(out, 0)
	out = appendUint16(out, TagUnit)
	rootLength := len(out)
	out = appendUint32(out, 0)
	out = appendStringNode(out, TagPackage, program.Package)
	out = appendStringNode(out, TagImportPath, program.ImportPath)
	out = appendNode(out, TagText, program.Text)
	tokenHeader := len(out)
	out = appendNodeHeader(out, TagTokens, 0)
	tokenStart := len(out)
	out = appendEncodedTokens(out, program.Tokens)
	patchUint32(out, tokenHeader+2, len(out)-tokenStart)
	declHeader := len(out)
	out = appendNodeHeader(out, TagDecls, 0)
	declStart := len(out)
	out = appendEncodedDecls(out, program.Decls)
	patchUint32(out, declHeader+2, len(out)-declStart)
	funcHeader := len(out)
	out = appendNodeHeader(out, TagFuncs, 0)
	funcStart := len(out)
	out = appendEncodedFuncs(out, program.Funcs)
	patchUint32(out, funcHeader+2, len(out)-funcStart)
	patchUint32(out, rootLength, len(out)-14)
	return out, true
}

func appendEncodedTokens(out []byte, tokens []Token) []byte {
	out = appendVarint(out, len(tokens))
	prevStart := 0
	prevLine := 0
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		out = appendVarint(out, tok.Kind)
		out = appendVarint(out, tok.Start-prevStart)
		out = appendVarint(out, tok.Size)
		out = appendVarint(out, tok.Line-prevLine)
		prevStart = tok.Start
		prevLine = tok.Line
	}
	return out
}

func appendEncodedDecls(out []byte, decls []Decl) []byte {
	out = appendVarint(out, len(decls))
	for i := 0; i < len(decls); i++ {
		decl := decls[i]
		out = appendVarint(out, decl.Kind)
		out = appendVarint(out, decl.NameStart)
		out = appendVarint(out, decl.NameEnd-decl.NameStart)
		out = appendVarint(out, decl.StartTok)
		out = appendVarint(out, decl.EndTok-decl.StartTok)
	}
	return out
}

func appendEncodedFuncs(out []byte, funcs []Func) []byte {
	out = appendVarint(out, len(funcs))
	for i := 0; i < len(funcs); i++ {
		fn := funcs[i]
		out = appendVarint(out, fn.NameStart)
		out = appendVarint(out, fn.NameEnd-fn.NameStart)
		out = appendVarint(out, fn.StartTok)
		out = appendVarint(out, fn.NameTok-fn.StartTok)
		out = appendVarint(out, fn.ReceiverStart)
		out = appendVarint(out, fn.ReceiverEnd-fn.ReceiverStart)
		out = appendVarint(out, fn.BodyStart)
		out = appendVarint(out, fn.BodyEnd-fn.BodyStart)
		out = appendVarint(out, fn.EndTok-fn.BodyEnd)
	}
	return out
}

func appendStringNode(out []byte, tag int, payload string) []byte {
	out = appendNodeHeader(out, tag, len(payload))
	for i := 0; i < len(payload); i++ {
		out = append(out, payload[i])
	}
	return out
}

func patchUint32(out []byte, at int, value int) {
	out[at] = byte(value)
	out[at+1] = byte(value >> 8)
	out[at+2] = byte(value >> 16)
	out[at+3] = byte(value >> 24)
}

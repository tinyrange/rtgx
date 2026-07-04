package unit

const (
	Magic   = "RTGU"
	Version = 1
)

const (
	TagUnit    = 1
	TagPackage = 2
	TagText    = 7
	TagTokens  = 8
	TagDecls   = 9
	TagFuncs   = 10
)

const (
	TokenEOF = iota
	TokenIdent
	TokenNumber
	TokenFloat
	TokenString
	TokenChar
	TokenPackage
	TokenConst
	TokenVar
	TokenType
	TokenFunc
	TokenStruct
	TokenReturn
	TokenIf
	TokenElse
	TokenFor
	TokenBreak
	TokenContinue
	TokenGoto
	TokenSwitch
	TokenCase
	TokenDefault
	TokenOp
)

type Token struct {
	Kind  int
	Start int
	Size  int
	Line  int
}

type Decl struct {
	Kind      int
	NameStart int
	NameEnd   int
	StartTok  int
	EndTok    int
}

type Func struct {
	NameStart     int
	NameEnd       int
	StartTok      int
	NameTok       int
	ReceiverStart int
	ReceiverEnd   int
	BodyStart     int
	BodyEnd       int
	EndTok        int
}

type Program struct {
	Package string
	Text    []byte
	Tokens  []Token
	Decls   []Decl
	Funcs   []Func
}

func Marshal(program Program) ([]byte, bool) {
	if len(program.Package) == 0 || len(program.Text) == 0 || len(program.Tokens) == 0 {
		return nil, false
	}
	tokenData, ok := encodeTokens(program.Text, program.Tokens)
	if !ok {
		return nil, false
	}
	declData, ok := encodeDecls(program.Decls)
	if !ok {
		return nil, false
	}
	funcData, ok := encodeFuncs(program.Funcs)
	if !ok {
		return nil, false
	}
	var root []byte
	root = appendNode(root, TagPackage, []byte(program.Package))
	root = appendNode(root, TagText, program.Text)
	root = appendNode(root, TagTokens, tokenData)
	root = appendNode(root, TagDecls, declData)
	root = appendNode(root, TagFuncs, funcData)

	out := make([]byte, 0, 14+len(root))
	out = append(out, 'R', 'T', 'G', 'U')
	out = appendUint16(out, Version)
	out = appendUint16(out, 0)
	out = appendNode(out, TagUnit, root)
	return out, true
}

func Unmarshal(data []byte) (Program, bool) {
	var program Program
	if len(data) < 14 {
		return program, false
	}
	if data[0] != 'R' || data[1] != 'T' || data[2] != 'G' || data[3] != 'U' {
		return program, false
	}
	if readUint16(data, 4) != Version {
		return program, false
	}
	rootTag := readUint16(data, 8)
	rootLength := readUint32(data, 10)
	if rootTag != TagUnit {
		return program, false
	}
	rootStart := 14
	rootEnd := rootStart + rootLength
	if rootEnd < rootStart || rootEnd != len(data) {
		return program, false
	}
	tokenData := []byte{}
	seenPackage := false
	seenText := false
	seenTokens := false
	seenDecls := false
	seenFuncs := false
	pos := rootStart
	for pos < rootEnd {
		if pos+6 > rootEnd {
			return program, false
		}
		tag := readUint16(data, pos)
		length := readUint32(data, pos+2)
		pos += 6
		next := pos + length
		if next < pos || next > rootEnd {
			return program, false
		}
		payload := data[pos:next]
		if tag == TagPackage {
			if seenPackage {
				return program, false
			}
			seenPackage = true
			program.Package = string(payload)
		} else if tag == TagText {
			if seenText {
				return program, false
			}
			seenText = true
			program.Text = payload
		} else if tag == TagTokens {
			if seenTokens {
				return program, false
			}
			seenTokens = true
			tokenData = payload
		} else if tag == TagDecls {
			if seenDecls {
				return program, false
			}
			seenDecls = true
			decls, ok := decodeDecls(payload)
			if !ok {
				return program, false
			}
			program.Decls = decls
		} else if tag == TagFuncs {
			if seenFuncs {
				return program, false
			}
			seenFuncs = true
			funcs, ok := decodeFuncs(payload)
			if !ok {
				return program, false
			}
			program.Funcs = funcs
		} else {
			return program, false
		}
		pos = next
	}
	if !seenPackage || !seenText || !seenTokens || !seenDecls || !seenFuncs {
		return program, false
	}
	if len(program.Package) == 0 || len(program.Text) == 0 {
		return program, false
	}
	tokens, ok := decodeTokens(program.Text, tokenData)
	if !ok || len(tokens) == 0 {
		return program, false
	}
	program.Tokens = tokens
	return program, true
}

func encodeTokens(text []byte, tokens []Token) ([]byte, bool) {
	out := make([]byte, 0, len(tokens)*4)
	out = appendVarint(out, len(tokens))
	prevStart := 0
	prevLine := 0
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		if tok.Kind < 0 || tok.Kind > 255 || tok.Start < prevStart || tok.Line < prevLine || tok.Size < 0 {
			return nil, false
		}
		if tok.Start > len(text) || tok.Start+tok.Size > len(text) {
			return nil, false
		}
		if tok.Start > 0xffffff || tok.Line > 0xffff {
			return nil, false
		}
		if tok.Kind == TokenOp {
			if tok.Size > 255 {
				return nil, false
			}
		} else if tok.Size > 0xffff {
			return nil, false
		}
		out = appendVarint(out, tok.Kind)
		out = appendVarint(out, tok.Start-prevStart)
		out = appendVarint(out, tok.Size)
		out = appendVarint(out, tok.Line-prevLine)
		prevStart = tok.Start
		prevLine = tok.Line
	}
	return out, true
}

func decodeTokens(text []byte, data []byte) ([]Token, bool) {
	pos := 0
	count, ok := readVarint(data, &pos)
	if !ok || count < 0 {
		return nil, false
	}
	tokens := make([]Token, 0, count)
	prevStart := 0
	prevLine := 0
	for i := 0; i < count; i++ {
		kind, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		startDelta, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		size, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		lineDelta, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		tok := Token{
			Kind:  kind,
			Start: prevStart + startDelta,
			Size:  size,
			Line:  prevLine + lineDelta,
		}
		if tok.Kind < 0 || tok.Kind > 255 || tok.Size < 0 || tok.Start < prevStart || tok.Line < prevLine {
			return nil, false
		}
		if tok.Start > len(text) || tok.Start+tok.Size > len(text) {
			return nil, false
		}
		if tok.Start > 0xffffff || tok.Line > 0xffff {
			return nil, false
		}
		if tok.Kind == TokenOp {
			if tok.Size > 255 {
				return nil, false
			}
		} else if tok.Size > 0xffff {
			return nil, false
		}
		tokens = append(tokens, tok)
		prevStart = tok.Start
		prevLine = tok.Line
	}
	if pos != len(data) {
		return nil, false
	}
	return tokens, true
}

func encodeDecls(decls []Decl) ([]byte, bool) {
	out := make([]byte, 0, len(decls)*5+1)
	out = appendVarint(out, len(decls))
	for i := 0; i < len(decls); i++ {
		decl := decls[i]
		if decl.Kind < 0 || decl.NameStart < 0 || decl.NameEnd < decl.NameStart || decl.StartTok < 0 || decl.EndTok < decl.StartTok {
			return nil, false
		}
		out = appendVarint(out, decl.Kind)
		out = appendVarint(out, decl.NameStart)
		out = appendVarint(out, decl.NameEnd-decl.NameStart)
		out = appendVarint(out, decl.StartTok)
		out = appendVarint(out, decl.EndTok-decl.StartTok)
	}
	return out, true
}

func decodeDecls(data []byte) ([]Decl, bool) {
	pos := 0
	count, ok := readVarint(data, &pos)
	if !ok || count < 0 {
		return nil, false
	}
	decls := make([]Decl, 0, count)
	for i := 0; i < count; i++ {
		kind, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		nameStart, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		nameSize, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		startTok, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		tokenSize, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		decl := Decl{
			Kind:      kind,
			NameStart: nameStart,
			NameEnd:   nameStart + nameSize,
			StartTok:  startTok,
			EndTok:    startTok + tokenSize,
		}
		if decl.Kind < 0 || decl.NameStart < 0 || decl.NameEnd < decl.NameStart || decl.StartTok < 0 || decl.EndTok < decl.StartTok {
			return nil, false
		}
		decls = append(decls, decl)
	}
	if pos != len(data) {
		return nil, false
	}
	return decls, true
}

func encodeFuncs(funcs []Func) ([]byte, bool) {
	out := make([]byte, 0, len(funcs)*9+1)
	out = appendVarint(out, len(funcs))
	for i := 0; i < len(funcs); i++ {
		fn := funcs[i]
		if fn.NameStart < 0 || fn.NameEnd < fn.NameStart || fn.StartTok < 0 || fn.NameTok < fn.StartTok {
			return nil, false
		}
		if fn.ReceiverStart < 0 || fn.ReceiverEnd < fn.ReceiverStart || fn.BodyStart < 0 || fn.BodyEnd < fn.BodyStart || fn.EndTok < fn.BodyEnd {
			return nil, false
		}
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
	return out, true
}

func decodeFuncs(data []byte) ([]Func, bool) {
	pos := 0
	count, ok := readVarint(data, &pos)
	if !ok || count < 0 {
		return nil, false
	}
	funcs := make([]Func, 0, count)
	for i := 0; i < count; i++ {
		nameStart, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		nameSize, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		startTok, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		nameDelta, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		receiverStart, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		receiverSize, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		bodyStart, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		bodySize, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		endSize, ok := readVarint(data, &pos)
		if !ok {
			return nil, false
		}
		fn := Func{
			NameStart:     nameStart,
			NameEnd:       nameStart + nameSize,
			StartTok:      startTok,
			NameTok:       startTok + nameDelta,
			ReceiverStart: receiverStart,
			ReceiverEnd:   receiverStart + receiverSize,
			BodyStart:     bodyStart,
			BodyEnd:       bodyStart + bodySize,
			EndTok:        bodyStart + bodySize + endSize,
		}
		if fn.NameStart < 0 || fn.NameEnd < fn.NameStart || fn.StartTok < 0 || fn.NameTok < fn.StartTok {
			return nil, false
		}
		if fn.ReceiverStart < 0 || fn.ReceiverEnd < fn.ReceiverStart || fn.BodyStart < 0 || fn.BodyEnd < fn.BodyStart || fn.EndTok < fn.BodyEnd {
			return nil, false
		}
		funcs = append(funcs, fn)
	}
	if pos != len(data) {
		return nil, false
	}
	return funcs, true
}

func appendNode(out []byte, tag int, payload []byte) []byte {
	out = appendUint16(out, tag)
	out = appendUint32(out, len(payload))
	out = append(out, payload...)
	return out
}

func appendUint16(out []byte, v int) []byte {
	return append(out, byte(v), byte(v>>8))
}

func appendUint32(out []byte, v int) []byte {
	return append(out, byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
}

func readUint16(data []byte, pos int) int {
	return int(data[pos]) | int(data[pos+1])<<8
}

func readUint32(data []byte, pos int) int {
	return int(data[pos]) | int(data[pos+1])<<8 | int(data[pos+2])<<16 | int(data[pos+3])<<24
}

func appendVarint(out []byte, v int) []byte {
	for v >= 0x80 {
		out = append(out, byte(v)|0x80)
		v = v >> 7
	}
	return append(out, byte(v))
}

func readVarint(data []byte, pos *int) (int, bool) {
	value := 0
	shift := 0
	for *pos < len(data) {
		b := int(data[*pos])
		*pos = *pos + 1
		if shift >= 28 && b >= 0x10 {
			return 0, false
		}
		value = value | (b&0x7f)<<shift
		if b < 0x80 {
			return value, true
		}
		shift += 7
		if shift > 28 {
			return 0, false
		}
	}
	return 0, false
}

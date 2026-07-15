package unit

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

const (
	CallUnknown = iota
	CallScope
	CallPackage
	CallImportSelector
	CallBuiltin
)

const (
	RefUnknown = iota
	RefScope
	RefPackage
	RefImport
	RefBuiltin
	RefLabel
)

const (
	TypeRefUnknown = iota
	TypeRefScope
	TypeRefPackage
	TypeRefImportSelector
	TypeRefBuiltin
)

func appendNode(out []byte, tag int, payload []byte) []byte {
	out = appendNodeHeader(out, tag, len(payload))
	for i := 0; i < len(payload); i++ {
		out = append(out, payload[i])
	}
	return out
}

func appendNodeHeader(out []byte, tag int, size int) []byte {
	out = appendUint16(out, tag)
	out = appendUint32(out, size)
	return out
}

func appendUint16(out []byte, v int) []byte {
	out = append(out, byte(v))
	out = append(out, byte(v>>8))
	return out
}

func appendUint32(out []byte, v int) []byte {
	out = append(out, byte(v))
	out = append(out, byte(v>>8))
	out = append(out, byte(v>>16))
	out = append(out, byte(v>>24))
	return out
}

func appendVarint(out []byte, v int) []byte {
	for v >= 0x80 {
		out = append(out, byte(v)|0x80)
		v = v >> 7
	}
	return append(out, byte(v))
}

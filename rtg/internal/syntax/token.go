package syntax

const (
	TokenEOF = iota
	TokenIdent
	TokenNumber
	TokenString
	TokenChar
	TokenOperator
	TokenPackage
	TokenImport
	TokenConst
	TokenVar
	TokenType
	TokenFunc
	TokenStruct
	TokenInterface
	TokenMap
	TokenReturn
	TokenIf
	TokenElse
	TokenFor
	TokenRange
	TokenSwitch
	TokenCase
	TokenDefault
	TokenBreak
	TokenContinue
	TokenGoto
	TokenDefer
	TokenGo
	TokenSelect
	TokenChan
	TokenFallthrough
)

type Token struct {
	Kind  int
	Start int
	End   int
	Line  int
}

func TokenText(src []byte, tok Token) []byte {
	if tok.Start < 0 || tok.End < tok.Start || tok.End > len(src) {
		return nil
	}
	return src[tok.Start:tok.End]
}

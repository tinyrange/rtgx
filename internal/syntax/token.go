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

const (
	TokenOperatorCharShift = 8
	TokenOperatorCharMask  = 127
	TokenOperatorLineShift = 15
	TokenLineLimit         = 65535
)

type Token struct {
	// KindLine packs the kind into the low byte. Operator tokens additionally
	// cache a one-byte ASCII operator in bits 8..14 and store the line above it;
	// all other token kinds store the line directly above the kind byte.
	KindLine int
	Start    int
	End      int
}

func MakeToken(kind int, start int, end int, line int) Token {
	if kind == TokenOperator {
		return Token{KindLine: kind | line<<TokenOperatorLineShift, Start: start, End: end}
	}
	return Token{KindLine: kind | line<<8, Start: start, End: end}
}

func TokenLine(tok Token) int {
	if tok.KindLine&255 == TokenOperator {
		return tok.KindLine >> TokenOperatorLineShift & TokenLineLimit
	}
	return tok.KindLine >> 8
}

func TokenText(src []byte, tok Token) []byte {
	if tok.Start < 0 || tok.End < tok.Start || tok.End > len(src) {
		return nil
	}
	return src[tok.Start:tok.End]
}

func NumberTokenIsFloat(src []byte, tok Token) bool {
	start := tok.Start
	end := tok.End
	if tok.KindLine&255 != TokenNumber || start < 0 || end < start || end > len(src) {
		return false
	}
	if end-start > 2 && src[start] == '0' {
		prefix := src[start+1]
		if prefix == 'x' || prefix == 'X' {
			for i := start + 2; i < end; i++ {
				c := src[i]
				if c == '.' || c == 'p' || c == 'P' {
					return true
				}
			}
			return false
		}
		if prefix == 'b' || prefix == 'B' || prefix == 'o' || prefix == 'O' {
			return false
		}
	}
	for i := start; i < end; i++ {
		c := src[i]
		if c == '.' || c == 'e' || c == 'E' || c == 'p' || c == 'P' {
			return true
		}
	}
	return false
}

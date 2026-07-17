//go:build rtg

package strconv

type NumError struct {
	text string
}

var syntaxErrorValue = NumError{text: "invalid syntax"}
var rangeErrorValue = NumError{text: "value out of range"}

func (e *NumError) Error() string {
	if e == nil {
		return ""
	}
	return e.text
}

func syntaxError() *NumError { return &syntaxErrorValue }
func rangeError() *NumError  { return &rangeErrorValue }

func Itoa(i int) string {
	return FormatInt(i, 10)
}

func Atoi(s string) (int, *NumError) {
	value, err := ParseInt(s, 10, 0)
	return int(value), err
}

func FormatInt(i int, base int) string {
	if base == 16 {
		return formatBase(i, 16)
	}
	return formatBase(i, 10)
}

func formatBase(i int, base int) string {
	if i == 0 {
		return "0"
	}
	var out []byte
	if i < 0 {
		out = append(out, '-')
		i = -i
	}
	var digits []byte
	for i > 0 {
		d := i % base
		if d < 10 {
			digits = append(digits, byte('0'+d))
		} else {
			digits = append(digits, byte('a'+d-10))
		}
		i = i / base
	}
	for i := len(digits) - 1; i >= 0; i-- {
		out = append(out, digits[i])
	}
	return string(out)
}

func ParseInt(s string, base int, bitSize int) (int64, *NumError) {
	if len(s) == 0 {
		return 0, syntaxError()
	}
	negative := false
	if s[0] == '+' || s[0] == '-' {
		negative = s[0] == '-'
		s = s[1:]
	}
	value, err := ParseUint(s, base, bitSize)
	if err != nil {
		return 0, err
	}
	if negative {
		return -int64(value), nil
	}
	return int64(value), nil
}

func ParseUint(s string, base int, bitSize int) (uint64, *NumError) {
	if len(s) == 0 {
		return 0, syntaxError()
	}
	if base == 0 {
		base = 10
		if len(s) > 1 && s[0] == '0' {
			base = 8
			if s[1] == 'x' || s[1] == 'X' {
				base = 16
				s = s[2:]
			} else if s[1] == 'b' || s[1] == 'B' {
				base = 2
				s = s[2:]
			} else if s[1] == 'o' || s[1] == 'O' {
				s = s[2:]
			}
		}
	}
	if base < 2 || base > 36 || len(s) == 0 {
		return 0, syntaxError()
	}
	var value uint64
	for i := 0; i < len(s); i++ {
		digit, ok := parseDigit(s[i])
		if !ok || digit >= base {
			return 0, syntaxError()
		}
		next := value*uint64(base) + uint64(digit)
		if next < value {
			return 0, rangeError()
		}
		value = next
	}
	return value, nil
}

func parseDigit(c byte) (int, bool) {
	if c >= '0' && c <= '9' {
		return int(c - '0'), true
	}
	if c >= 'a' && c <= 'z' {
		return int(c-'a') + 10, true
	}
	if c >= 'A' && c <= 'Z' {
		return int(c-'A') + 10, true
	}
	return 0, false
}

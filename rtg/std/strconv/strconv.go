package strconv

func Itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := false
	n := i
	if n < 0 {
		neg = true
		n = -n
	}
	var rev []byte
	for n > 0 {
		d := n % 10
		rev = append(rev, byte(48+d))
		n = n / 10
	}
	var out []byte
	if neg {
		out = append(out, '-')
	}
	for i := len(rev) - 1; i >= 0; i = i - 1 {
		out = append(out, rev[i])
	}
	return string(out)
}

func FormatInt(i int64, base int) string {
	if base < 2 || base > 36 {
		base = 10
	}
	if i == 0 {
		return "0"
	}
	neg := false
	n := i
	if n < 0 {
		neg = true
		n = -n
	}
	var rev []byte
	for n > 0 {
		d := int(n % int64(base))
		if d < 10 {
			rev = append(rev, byte(48+d))
		} else {
			rev = append(rev, byte(87+d))
		}
		n = n / int64(base)
	}
	var out []byte
	if neg {
		out = append(out, '-')
	}
	for i := len(rev) - 1; i >= 0; i = i - 1 {
		out = append(out, rev[i])
	}
	return string(out)
}

func Atoi(s string) (int, error) {
	n := 0
	neg := false
	i := 0
	if len(s) > 0 && s[0] == '-' {
		neg = true
		i = 1
	}
	for i < len(s) {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, parseError("invalid syntax")
		}
		n = n*10 + int(c-'0')
		i = i + 1
	}
	if neg {
		n = -n
	}
	return n, nil
}

func ParseInt(s string, base int, bitSize int) (int64, error) {
	if len(s) == 0 {
		return 0, parseError("invalid syntax")
	}
	neg := false
	i := 0
	if s[0] == '-' || s[0] == '+' {
		neg = s[0] == '-'
		i = 1
		if i >= len(s) {
			return 0, parseError("invalid syntax")
		}
	}
	if base == 0 {
		base = 10
		if i+1 < len(s) && s[i] == '0' {
			next := s[i+1]
			if next == 'x' || next == 'X' {
				base = 16
				i = i + 2
			} else if next == 'b' || next == 'B' {
				base = 2
				i = i + 2
			} else if next == 'o' || next == 'O' {
				base = 8
				i = i + 2
			} else {
				base = 8
				i = i + 1
			}
		}
	}
	if base < 2 || base > 36 || i >= len(s) {
		return 0, parseError("invalid syntax")
	}
	var n int64
	for i < len(s) {
		d := digitValue(s[i])
		if d < 0 || d >= base {
			return 0, parseError("invalid syntax")
		}
		n = n*int64(base) + int64(d)
		i = i + 1
	}
	if neg {
		n = -n
	}
	return n, nil
}

func digitValue(c byte) int {
	if c >= '0' && c <= '9' {
		return int(c - '0')
	}
	if c >= 'a' && c <= 'z' {
		return int(c-'a') + 10
	}
	if c >= 'A' && c <= 'Z' {
		return int(c-'A') + 10
	}
	return -1
}

func Quote(s string) string {
	var out []byte
	out = append(out, '"')
	i := 0
	for i < len(s) {
		c := s[i]
		if c == '\\' {
			out = append(out, '\\')
			out = append(out, '\\')
		} else if c == '"' {
			out = append(out, '\\')
			out = append(out, '"')
		} else if c == '\n' {
			out = append(out, '\\')
			out = append(out, 'n')
		} else if c == '\r' {
			out = append(out, '\\')
			out = append(out, 'r')
		} else if c == '\t' {
			out = append(out, '\\')
			out = append(out, 't')
		} else {
			out = append(out, c)
		}
		i = i + 1
	}
	out = append(out, '"')
	return string(out)
}

func Unquote(s string) (string, error) {
	if len(s) < 2 {
		return "", parseError("invalid syntax")
	}
	quote := s[0]
	if quote != '"' && quote != '`' {
		return "", parseError("invalid syntax")
	}
	if s[len(s)-1] != quote {
		return "", parseError("invalid syntax")
	}
	if quote == '`' {
		return s[1 : len(s)-1], nil
	}
	var out []byte
	i := 1
	for i < len(s)-1 {
		c := s[i]
		if c != '\\' {
			out = append(out, c)
			i = i + 1
			continue
		}
		i = i + 1
		if i >= len(s)-1 {
			return "", parseError("invalid syntax")
		}
		esc := s[i]
		if esc == 'n' {
			out = append(out, '\n')
		} else if esc == 'r' {
			out = append(out, '\r')
		} else if esc == 't' {
			out = append(out, '\t')
		} else {
			out = append(out, esc)
		}
		i = i + 1
	}
	return string(out), nil
}

type parseError string

func (err parseError) Error() string {
	return string(err)
}

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

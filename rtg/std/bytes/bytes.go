package bytes

type Buffer struct {
	buf []byte
}

func NewBuffer(buf []byte) *Buffer {
	b := new(Buffer)
	b.buf = buf
	return b
}

func NewBufferString(s string) *Buffer {
	return NewBuffer([]byte(s))
}

func Equal(a []byte, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	i := 0
	for i < len(a) {
		if a[i] != b[i] {
			return false
		}
		i = i + 1
	}
	return true
}

func HasPrefix(s []byte, prefix []byte) bool {
	n := len(prefix)
	if n > len(s) {
		return false
	}
	return Equal(s[0:n], prefix)
}

func HasSuffix(s []byte, suffix []byte) bool {
	n := len(suffix)
	if n > len(s) {
		return false
	}
	start := len(s) - n
	return Equal(s[start:len(s)], suffix)
}

func Contains(s []byte, subslice []byte) bool {
	return Index(s, subslice) >= 0
}

func Index(s []byte, subslice []byte) int {
	n := len(subslice)
	if n == 0 {
		return 0
	}
	if n > len(s) {
		return -1
	}
	limit := len(s) - n
	i := 0
	for i <= limit {
		if Equal(s[i:i+n], subslice) {
			return i
		}
		i = i + 1
	}
	return -1
}

func IndexByte(s []byte, c byte) int {
	i := 0
	for i < len(s) {
		if s[i] == c {
			return i
		}
		i = i + 1
	}
	return -1
}

func (b *Buffer) WriteString(s string) int {
	i := 0
	for i < len(s) {
		b.buf = append(b.buf, s[i])
		i = i + 1
	}
	return len(s)
}

func (b *Buffer) Write(p []byte) int {
	i := 0
	for i < len(p) {
		b.buf = append(b.buf, p[i])
		i = i + 1
	}
	return len(p)
}

func (b *Buffer) WriteByte(c byte) int {
	b.buf = append(b.buf, c)
	return 0
}

func (b *Buffer) Len() int {
	return len(b.buf)
}

func (b *Buffer) String() string {
	return string(b.buf)
}

func (b *Buffer) Bytes() []byte {
	return b.buf
}

func (b *Buffer) Reset() {
	var empty []byte
	b.buf = empty
}

package scan

import "testing"

func TestTokensTrackPositionsAndSkipComments(t *testing.T) {
	toks, err := Tokens([]byte("package main\n// ignored\nfunc appMain() int { return 0 }\n"))
	if err != nil {
		t.Fatalf("Tokens failed: %v", err)
	}
	if toks[0].Text != "package" || toks[0].Line != 1 || toks[0].Column != 1 {
		t.Fatalf("first token = %#v", toks[0])
	}
	if toks[2].Text != "func" || toks[2].Line != 3 || toks[2].Column != 1 {
		t.Fatalf("func token = %#v", toks[2])
	}
}

func TestTokensRecognizesCompoundOperators(t *testing.T) {
	toks, err := Tokens([]byte("a := b && c <- d ..."))
	if err != nil {
		t.Fatalf("Tokens failed: %v", err)
	}
	want := []string{"a", ":=", "b", "&&", "c", "<-", "d", "..."}
	for i, text := range want {
		if toks[i].Text != text {
			t.Fatalf("token %d = %q, want %q", i, toks[i].Text, text)
		}
	}
}

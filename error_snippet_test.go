package jsonvx

import (
	"errors"
	"strings"
	"testing"
)

func TestErrorSnippet(t *testing.T) {
	tests := []struct {
		msg   string
		input string
		want  string
	}{
		{msg: "short token is unchanged", input: `@`, want: `@`},
		{msg: "stops at the end of the line", input: "// cache settings\n  \"ttl\": 300\n}", want: "// cache settings…"},
		{msg: "stops at a Windows line ending", input: "// note\r\n}", want: "// note…"},
		{msg: "a line break at the end of the token only", input: "abc\n", want: "abc…"},
		{msg: "long line is cut", input: strings.Repeat("x", 50), want: strings.Repeat("x", 30) + "…"},
		{msg: "exactly the limit is unchanged", input: strings.Repeat("x", 30), want: strings.Repeat("x", 30)},
		{msg: "cuts by characters, not bytes", input: strings.Repeat("é", 40), want: strings.Repeat("é", 30) + "…"},
		{msg: "starts with a line break", input: "\nrest", want: "\n…"},
		{msg: "starts with a Windows line break", input: "\r\nrest", want: "\r\n…"},
		{msg: "only a line break", input: "\n", want: "\n"},
		{msg: "empty", input: "", want: ""},
	}

	for _, test := range tests {
		t.Run(test.msg, func(t *testing.T) {
			if got := errorSnippet([]byte(test.input)); got != test.want {
				t.Errorf("errorSnippet(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

// Strict mode stops at the first invalid character. The message quotes only that
// line, not the rest of the input, and keeps the exact position.
func TestUnexpectedCharErrorIsShort(t *testing.T) {
	config := []byte("{\n  // cache settings\n  \"ttl\": 300,\n  \"hosts\": [\"a.example.com\", \"b.example.com\"],\n}")

	_, err := ParseObject(config, nil)

	want := `unexpected character in JSON input: "// cache settings…" at line 2, column 3`
	if err == nil || err.Error() != want {
		t.Fatalf("got %v, want %s", err, want)
	}
	if !errors.Is(err, ErrJSONUnexpectedChar) {
		t.Errorf("errors.Is(err, ErrJSONUnexpectedChar) = false, want true")
	}
}

// A long invalid section on one line is cut, so the message stays readable.
func TestLongInvalidInputIsCut(t *testing.T) {
	input := []byte(`{"a": 'single quotes are not JSON` + strings.Repeat(" and this goes on", 20) + `'}`)

	_, err := ParseObject(input, nil)

	if err == nil {
		t.Fatal("got nil error, want a syntax error")
	}
	if len(err.Error()) > 120 {
		t.Errorf("message is %d bytes, want it short: %s", len(err.Error()), err)
	}
	if !strings.Contains(err.Error(), "…") || !strings.Contains(err.Error(), "line 1, column 7") {
		t.Errorf("got %q, want a shortened snippet and the position", err)
	}
}

func TestNumberOutOfRangeErrorIsShort(t *testing.T) {
	root, err := ParseObject([]byte(`{"id": `+strings.Repeat("9", 500)+`}`), nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = root.GetInt("id")

	if !errors.Is(err, ErrNumberOutOfRange) {
		t.Fatalf("got %v, want ErrNumberOutOfRange", err)
	}
	if len(err.Error()) > 120 {
		t.Errorf("message is %d bytes, want it short: %s", len(err.Error()), err)
	}
}

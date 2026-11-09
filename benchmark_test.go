package jsonvx

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"testing"
)

// Benchmarks compare jsonvx with encoding/json decoding into `any` on the same
// realistic document (about 0.5 MB compact, 0.8 MB indented). Run them with:
//
//	go test -run '^$' -bench . -benchmem

var (
	benchOnce     sync.Once
	benchCompact  []byte
	benchIndented []byte
)

func benchmarkData() ([]byte, []byte) {
	benchOnce.Do(func() {
		type user struct {
			ID      int64             `json:"id"`
			Name    string            `json:"name"`
			Email   string            `json:"email"`
			Active  bool              `json:"active"`
			Score   float64           `json:"score"`
			Tags    []string          `json:"tags"`
			Address map[string]string `json:"address"`
			Manager *user             `json:"manager"`
		}

		r := rand.New(rand.NewSource(1))
		users := make([]user, 2500)
		for i := range users {
			users[i] = user{
				ID:      int64(i),
				Name:    fmt.Sprintf("User Number %d", i),
				Email:   fmt.Sprintf("user%d@example.com", i),
				Active:  r.Intn(2) == 0,
				Score:   r.Float64() * 1000,
				Tags:    []string{"alpha", "beta", "gamma"},
				Address: map[string]string{"street": "123 Main St", "city": "Springfield", "zip": "12345"},
			}
		}

		benchCompact, _ = json.Marshal(users)
		benchIndented, _ = json.MarshalIndent(users, "", "  ")
	})

	return benchCompact, benchIndented
}

func benchmarkJsonvx(b *testing.B, data []byte) {
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		parser := NewParser(data, nil)
		if _, err := parser.Parse(); err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkEncodingJSON(b *testing.B, data []byte) {
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var v any
		if err := json.Unmarshal(data, &v); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseCompact(b *testing.B) {
	compact, _ := benchmarkData()
	benchmarkJsonvx(b, compact)
}

func BenchmarkParseIndented(b *testing.B) {
	_, indented := benchmarkData()
	benchmarkJsonvx(b, indented)
}

func BenchmarkEncodingJSONCompact(b *testing.B) {
	compact, _ := benchmarkData()
	benchmarkEncodingJSON(b, compact)
}

func BenchmarkEncodingJSONIndented(b *testing.B) {
	_, indented := benchmarkData()
	benchmarkEncodingJSON(b, indented)
}

func TestTokensStillIncludesWhitespace(t *testing.T) {
	// The parser now uses TokensWithout(WHITESPACE), but the public Tokens method
	// keeps returning every token, whitespace included.
	input := []byte("[1, 2]")

	all := NewLexer(input, NewParserConfig()).Tokens()
	withoutWS := NewLexer(input, NewParserConfig()).TokensWithout(WHITESPACE)

	countWS := func(tokens Tokens) int {
		n := 0
		for _, token := range tokens {
			if token.Kind == WHITESPACE {
				n++
			}
		}
		return n
	}

	if countWS(all) != 1 {
		t.Errorf("Tokens() has %d whitespace tokens, expected 1", countWS(all))
	}

	if countWS(withoutWS) != 0 {
		t.Errorf("TokensWithout(WHITESPACE) has %d whitespace tokens, expected 0", countWS(withoutWS))
	}

	if len(all)-len(withoutWS) != 1 {
		t.Errorf("token counts %d and %d should differ by exactly the whitespace token", len(all), len(withoutWS))
	}
}

func TestParseWhitespaceHeavyInput(t *testing.T) {
	// Whitespace everywhere it is allowed, including around every structural character.
	input := " \n\t{ \n \"a\" \t : \n [ 1 , \n 2 ] , \"b\" : { \"c\" : null } \n } \t\n "

	obj := mustParseObject(t, input, nil)

	if got := obj.String(); got != "{a: [1, 2], b: {c: \033[1mnull\033[0m}}" {
		t.Errorf("String() = %q", got)
	}
}

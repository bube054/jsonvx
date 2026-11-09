package jsonvx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestJSONTestSuite checks the parser, in its default strict mode, against the
// test_parsing cases from JSONTestSuite (https://github.com/nst/JSONTestSuite),
// stored in testdata/JSONTestSuite. The file name says what a strict parser must do:
//
//   - y_*.json must be accepted
//   - n_*.json must be rejected
//   - i_*.json may go either way, but must not crash
func TestJSONTestSuite(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("testdata", "JSONTestSuite", "*.json"))
	if err != nil {
		t.Fatal(err)
	}

	if len(files) == 0 {
		t.Fatal("no JSONTestSuite files found in testdata/JSONTestSuite")
	}

	for _, file := range files {
		name := filepath.Base(file)

		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}

			parser := NewParser(data, nil)
			_, parseErr := parser.Parse()

			switch {
			case strings.HasPrefix(name, "y_"):
				if parseErr != nil {
					t.Errorf("must be accepted, got error: %v\ninput: %q", parseErr, truncate(data))
				}
			case strings.HasPrefix(name, "n_"):
				if parseErr == nil {
					t.Errorf("must be rejected, but was accepted\ninput: %q", truncate(data))
				}
			case strings.HasPrefix(name, "i_"):
				t.Logf("implementation-defined: accepted=%v", parseErr == nil)
			}
		})
	}
}

func truncate(data []byte) []byte {
	if len(data) > 80 {
		return append(data[:80:80], "..."...)
	}

	return data
}

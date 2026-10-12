package cleanenv

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestReadConfigPreservesParserError(t *testing.T) {
	for _, tc := range []struct{ name, extension, content string }{
		{"empty JSON", ".json", ""},
		{"empty YAML", ".yaml", ""},
		{"invalid JSON", ".json", "{invalid}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, err := ioutil.TempFile("", "cleanenv-*"+tc.extension)
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(f.Name())
			if _, err = f.WriteString(tc.content); err != nil {
				f.Close()
				t.Fatal(err)
			}
			if err = f.Close(); err != nil {
				t.Fatal(err)
			}
			var cfg struct{ Value string }
			err = ReadConfig(f.Name(), &cfg)
			if err == nil || !strings.HasPrefix(err.Error(), "config file parsing error: ") {
				t.Fatalf("expected contextual parser error, got %v", err)
			}
			if tc.content == "" {
				if !errors.Is(err, io.EOF) {
					t.Fatalf("expected wrapped EOF, got %v", err)
				}
			} else {
				var syntaxError *json.SyntaxError
				if !errors.As(err, &syntaxError) {
					t.Fatalf("expected wrapped JSON syntax error, got %v", err)
				}
			}
		})
	}
}

package cleanenv

import (
	"fmt"
	"os"
	"testing"
)

type textWrappedValue struct{ value string }

func (v *textWrappedValue) UnmarshalText(text []byte) error {
	if string(text) == "invalid" {
		return fmt.Errorf("invalid wrapped value")
	}
	v.value = string(text)
	return nil
}

func TestStructTextUnmarshaler(t *testing.T) {
	const key = "CLEANENV_TEXT_WRAPPER_TEST"
	original, exists := os.LookupEnv(key)
	defer func() {
		if exists {
			os.Setenv(key, original)
		} else {
			os.Unsetenv(key)
		}
	}()
	for _, tc := range []struct {
		name, value    string
		set, wantError bool
	}{
		{"environment", "parsed", true, false},
		{"default", "fallback", false, false},
		{"invalid", "invalid", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.set {
				os.Setenv(key, tc.value)
			} else {
				os.Unsetenv(key)
			}
			var cfg struct {
				Value textWrappedValue `env:"CLEANENV_TEXT_WRAPPER_TEST" env-default:"fallback"`
			}
			err := ReadEnv(&cfg)
			if (err != nil) != tc.wantError {
				t.Fatalf("ReadEnv error=%v, wantError=%v", err, tc.wantError)
			}
			if !tc.wantError && cfg.Value.value != tc.value {
				t.Errorf("got %q, want %q", cfg.Value.value, tc.value)
			}
		})
	}
}

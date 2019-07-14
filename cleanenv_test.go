package cleanenv

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

func TestReadEnvVars(t *testing.T) {
	type Combined struct {
		Empty   int
		Default int `env:"TEST0" default:"1"`
		Global  int `env:"TEST1" default:"1"`
		local   int `env:"TEST2" default:"1"`
	}

	type AllTypes struct {
		Integer         int64             `env:"TEST_INTEGER"`
		UnsInteger      uint64            `env:"TEST_UNSINTEGER"`
		Float           float64           `env:"TEST_FLOAT"`
		Boolean         bool              `env:"TEST_BOOLEAN"`
		String          string            `env:"TEST_STRING"`
		ArrayInt        []int             `env:"TEST_ARRAYINT"`
		ArrayString     []string          `env:"TEST_ARRAYSTRING"`
		MapStringInt    map[string]int    `env:"TEST_MAPSTRINGINT"`
		MapStringString map[string]string `env:"TEST_MAPSTRINGSTRING"`
	}

	tests := []struct {
		name    string
		env     map[string]string
		cfg     interface{}
		want    interface{}
		wantErr bool
	}{
		{
			name: "combined",
			env: map[string]string{
				"TEST1": "2",
				"TEST2": "3",
			},
			cfg: &Combined{},
			want: &Combined{
				Empty:   0,
				Default: 1,
				Global:  2,
				local:   0,
			},
			wantErr: false,
		},

		{
			name: "all types",
			env: map[string]string{
				"TEST_INTEGER":         "-5",
				"TEST_UNSINTEGER":      "5",
				"TEST_FLOAT":           "5.5",
				"TEST_BOOLEAN":         "true",
				"TEST_STRING":          "test",
				"TEST_ARRAYINT":        "1,2,3",
				"TEST_ARRAYSTRING":     "a,b,c",
				"TEST_MAPSTRINGINT":    "a:1,b:2,c:3",
				"TEST_MAPSTRINGSTRING": "a:x,b:y,c:z",
			},
			cfg: &AllTypes{},
			want: &AllTypes{
				Integer:     -5,
				UnsInteger:  5,
				Float:       5.5,
				Boolean:     true,
				String:      "test",
				ArrayInt:    []int{1, 2, 3},
				ArrayString: []string{"a", "b", "c"},
				MapStringInt: map[string]int{
					"a": 1,
					"b": 2,
					"c": 3,
				},
				MapStringString: map[string]string{
					"a": "x",
					"b": "y",
					"c": "z",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for env, val := range tt.env {
				os.Setenv(env, val)
			}
			defer os.Clearenv()

			if err := readEnvVars(tt.cfg); (err != nil) != tt.wantErr {
				t.Errorf("wrong error behavior %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.cfg, tt.want) {
				t.Errorf("wrong data %v, want %v", tt.cfg, tt.want)
			}
		})
	}
}

func TestParseFile(t *testing.T) {
	type configObject struct {
		One int `yaml:"one" json:"one" toml:"one"`
		Two int `yaml:"two" json:"two" toml:"two"`
	}
	type config struct {
		Number  int64        `yaml:"number" json:"number" toml:"number"`
		Float   float64      `yaml:"float" json:"float" toml:"float"`
		String  string       `yaml:"string" json:"string" toml:"string"`
		Boolean bool         `yaml:"boolean" json:"boolean" toml:"boolean"`
		Object  configObject `yaml:"object" json:"object" toml:"object"`
		Array   []int        `yaml:"array" json:"array" toml:"array"`
	}

	wantConfig := config{
		Number:  1,
		Float:   2.3,
		String:  "test",
		Boolean: true,
		Object:  configObject{1, 2},
		Array:   []int{1, 2, 3},
	}

	tests := []struct {
		name    string
		file    string
		ext     string
		want    *config
		wantErr bool
	}{
		{
			name: "yaml",
			file: `
number: 1
float: 2.3
string: test
boolean: yes
object:
  one: 1
  two: 2
array:
  - 1
  - 2
  - 3`,
			ext:     "yaml",
			want:    &wantConfig,
			wantErr: false,
		},

		{
			name: "json",
			file: `{
	"number": 1,
	"float": 2.3,
	"string": "test",
	"boolean": true,
	"object": {
		"one": 1,
		"two": 2
	},
	"array": [1, 2, 3]
}`,
			ext:     "json",
			want:    &wantConfig,
			wantErr: false,
		},

		{
			name: "toml",
			file: `
number = 1
float = 2.3
string = "test"
boolean = true

array = [1, 2, 3]

[object]
one = 1
two = 2`,
			ext:     "toml",
			want:    &wantConfig,
			wantErr: false,
		},

		{
			name:    "unknown",
			file:    "-",
			ext:     "",
			want:    nil,
			wantErr: true,
		},

		{
			name:    "parsing error",
			file:    "-",
			ext:     "json",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := ioutil.TempFile(os.TempDir(), fmt.Sprintf("*.%s", tt.ext))
			if err != nil {
				t.Fatal("cannot create temporary file:", err)
			}
			defer os.Remove(tmpFile.Name())

			text := []byte(tt.file)
			if _, err = tmpFile.Write(text); err != nil {
				t.Fatal("failed to write to temporary file:", err)
			}

			var cfg config
			if err = parseFile(tmpFile.Name(), &cfg); (err != nil) != tt.wantErr {
				t.Errorf("wrong error behavior %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && !reflect.DeepEqual(&cfg, tt.want) {
				t.Errorf("wrong data %v, want %v", &cfg, tt.want)
			}
		})
	}
}

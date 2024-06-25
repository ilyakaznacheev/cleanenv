package cleanenv

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"reflect"
	"testing"
	"time"
)

type testUpdater struct {
	Data string `env:"DATA"`
	err  error
}

func (t *testUpdater) Update() error {
	return t.err
}

func TestReadEnvVars(t *testing.T) {
	durationFunc := func(s string) time.Duration {
		d, err := time.ParseDuration(s)
		if err != nil {
			t.Fatal(err)
		}
		return d
	}

	timeFunc := func(s, l string) time.Time {
		tm, err := time.Parse(l, s)
		if err != nil {
			t.Fatal(err)
		}
		return tm
	}

	ta := &testUpdater{
		err: errors.New("test"),
	}

	type Combined struct {
		Empty   int
		Default int `env:"TEST0" env-default:"1"`
		Global  int `env:"TEST1" env-default:"1"`
		local   int `env:"TEST2" env-default:"1"`
	}

	type AllTypes struct {
		Integer    int64         `env:"TEST_INTEGER"`
		UnsInteger uint64        `env:"TEST_UNSINTEGER"`
		Float      float64       `env:"TEST_FLOAT"`
		Boolean    bool          `env:"TEST_BOOLEAN"`
		String     string        `env:"TEST_STRING"`
		Duration   time.Duration `env:"TEST_DURATION"`
		Time       time.Time     `env:"TEST_TIME"`
		// Location depends on the system, so we test it with time.UTC
		Location        *time.Location    `env:"TEST_LOCATION"`
		ArrayInt        []int             `env:"TEST_ARRAYINT"`
		ArrayString     []string          `env:"TEST_ARRAYSTRING"`
		MapStringInt    map[string]int    `env:"TEST_MAPSTRINGINT"`
		MapStringString map[string]string `env:"TEST_MAPSTRINGSTRING"`
	}

	type TimeTypes struct {
		Time1 time.Time            `env:"TEST_TIME1"`
		Time2 time.Time            `env:"TEST_TIME2" env-layout:"Mon Jan _2 15:04:05 2006"`
		Time3 time.Time            `env:"TEST_TIME3" env-layout:"Jan _2 15:04:05"`
		Time4 time.Time            `env:"TEST_TIME4" env-default:"2012-04-23T18:25:43.511Z"`
		Time5 time.Time            `env:"TEST_TIME5" env-default:"Mon Mar 10 11:11:11 2011" env-layout:"Mon Jan _2 15:04:05 2006"`
		Time6 []time.Time          `env:"TEST_TIME6" env-separator:"|"`
		Time7 map[string]time.Time `env:"TEST_TIME7" env-separator:"|"`
	}

	type Required struct {
		NotRequired int `env:"NOT_REQUIRED"`
		Required    int `env:"REQUIRED" env-required:"true"`
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
				"TEST_INTEGER":    "-5",
				"TEST_UNSINTEGER": "5",
				"TEST_FLOAT":      "5.5",
				"TEST_BOOLEAN":    "true",
				"TEST_STRING":     "test",
				"TEST_DURATION":   "1h5m10s",
				"TEST_TIME":       "2012-04-23T18:25:43.511Z",
				// Location depends on the system, so we test it with time.UTC
				"TEST_LOCATION":        "UTC",
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
				Duration:    durationFunc("1h5m10s"),
				Time:        timeFunc("2012-04-23T18:25:43.511Z", time.RFC3339),
				Location:    time.UTC,
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

		{
			name: "times",
			env: map[string]string{
				"TEST_TIME1": "2012-04-23T18:25:43.511Z",
				"TEST_TIME2": "Mon Mar 10 11:11:11 2011",
				"TEST_TIME3": "Dec 1 11:11:11",
				"TEST_TIME6": "2012-04-23T18:25:43.511Z|2012-05-23T18:25:43.511Z",
				"TEST_TIME7": "a:2012-04-23T18:25:43.511Z|b:2012-05-23T18:25:43.511Z",
			},
			cfg: &TimeTypes{},
			want: &TimeTypes{
				Time1: timeFunc("2012-04-23T18:25:43.511Z", time.RFC3339),
				Time2: timeFunc("Mon Mar 10 11:11:11 2011", time.ANSIC),
				Time3: timeFunc("Dec 1 11:11:11", time.Stamp),
				Time4: timeFunc("2012-04-23T18:25:43.511Z", time.RFC3339),
				Time5: timeFunc("Mon Mar 10 11:11:11 2011", time.ANSIC),
				Time6: []time.Time{
					timeFunc("2012-04-23T18:25:43.511Z", time.RFC3339),
					timeFunc("2012-05-23T18:25:43.511Z", time.RFC3339),
				},
				Time7: map[string]time.Time{
					"a": timeFunc("2012-04-23T18:25:43.511Z", time.RFC3339),
					"b": timeFunc("2012-05-23T18:25:43.511Z", time.RFC3339),
				},
			},
			wantErr: false,
		},

		{
			name: "wrong types",
			env: map[string]string{
				"TEST_INTEGER":         "a",
				"TEST_UNSINTEGER":      "b",
				"TEST_FLOAT":           "c",
				"TEST_BOOLEAN":         "xxx",
				"TEST_STRING":          "",
				"TEST_DURATION":        "-",
				"TEST_ARRAYINT":        "a,b,c",
				"TEST_ARRAYSTRING":     "1,2,3",
				"TEST_MAPSTRINGINT":    "a:x,b:y,c:z",
				"TEST_MAPSTRINGSTRING": "a:1,b:2,c:3",
			},
			cfg:     &AllTypes{},
			want:    &AllTypes{},
			wantErr: true,
		},

		{
			name: "wrong int",
			env: map[string]string{
				"TEST_INTEGER": "a",
			},
			cfg:     &AllTypes{},
			want:    &AllTypes{},
			wantErr: true,
		},

		{
			name: "wrong uint",
			env: map[string]string{
				"TEST_UNSINTEGER": "b",
			},
			cfg:     &AllTypes{},
			want:    &AllTypes{},
			wantErr: true,
		},

		{
			name: "wrong float",
			env: map[string]string{
				"TEST_FLOAT": "c",
			},
			cfg:     &AllTypes{},
			want:    &AllTypes{},
			wantErr: true,
		},

		{
			name: "wrong boolean",
			env: map[string]string{
				"TEST_BOOLEAN": "xxx",
			},
			cfg:     &AllTypes{},
			want:    &AllTypes{},
			wantErr: true,
		},

		{
			name: "wrong duration",
			env: map[string]string{
				"TEST_DURATION": "-",
			},
			cfg:     &AllTypes{},
			want:    &AllTypes{},
			wantErr: true,
		},

		{
			name: "wrong array int",
			env: map[string]string{
				"TEST_ARRAYINT": "a,b,c",
			},
			cfg:     &AllTypes{},
			want:    &AllTypes{},
			wantErr: true,
		},

		{
			name: "wrong map int",
			env: map[string]string{
				"TEST_MAPSTRINGINT": "a:x,b:y,c:z",
			},
			cfg:     &AllTypes{},
			want:    &AllTypes{},
			wantErr: true,
		},

		{
			name: "wrong map type int",
			env: map[string]string{
				"TEST_MAPSTRINGINT": "-",
			},
			cfg:     &AllTypes{},
			want:    &AllTypes{},
			wantErr: true,
		},

		{
			name: "wrong map type string",
			env: map[string]string{
				"TEST_MAPSTRINGSTRING": "-",
			},
			cfg:     &AllTypes{},
			want:    &AllTypes{},
			wantErr: true,
		},

		{
			name:    "wrong config type",
			cfg:     42,
			want:    42,
			wantErr: true,
		},

		{
			name:    "updater error",
			cfg:     ta,
			want:    ta,
			wantErr: true,
		},

		{
			name:    "required error",
			cfg:     &Required{},
			want:    &Required{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for env, val := range tt.env {
				os.Setenv(env, val)
			}
			defer os.Clearenv()

			if err := readEnvVars(tt.cfg, false); (err != nil) != tt.wantErr {
				t.Errorf("wrong error behavior %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.cfg, tt.want) {
				t.Errorf("wrong data %v, want %v", tt.cfg, tt.want)
			}
		})
	}
}

func TestReadEnvVarsURL(t *testing.T) {
	urlFunc := func(u string) url.URL {
		parsed, err := url.Parse(u)
		if err != nil {
			t.Fatal(err)
		}
		return *parsed
	}

	type WithURL struct {
		DatabaseURL url.URL `env:"DB_URL"`
	}

	tests := []struct {
		name    string
		env     map[string]string
		cfg     interface{}
		want    interface{}
		wantErr bool
	}{
		{
			name: "time",
			env: map[string]string{
				"DB_URL": "://:password@invalid-url:port/db-name",
			},
			cfg: &WithURL{},
			want: &WithURL{
				DatabaseURL: url.URL{},
			},
			wantErr: true,
		},
		{
			name: "time",
			env: map[string]string{
				"DB_URL": "postgres://user:password@host:1234/db-name",
			},
			cfg: &WithURL{},
			want: &WithURL{
				DatabaseURL: urlFunc("postgres://user:password@host:1234/db-name"),
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

			if err := readEnvVars(tt.cfg, false); (err != nil) != tt.wantErr {
				t.Errorf("wrong error behavior %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.cfg, tt.want) {
				fmt.Println(tt.cfg.(*WithURL).DatabaseURL)
				t.Errorf("wrong data: got %v, want %v", tt.cfg, tt.want)
			}
		})
	}
}

func TestReadEnvVarsTime(t *testing.T) {
	timeFunc := func(s, l string) time.Time {
		tm, err := time.Parse(l, s)
		if err != nil {
			t.Fatal(err)
		}
		return tm
	}

	type Timed struct {
		Time time.Time `env:"TEST_TIME" env-layout:"Mon Jan _2 15:04:05 2006"`
	}

	tests := []struct {
		name    string
		env     map[string]string
		cfg     interface{}
		want    interface{}
		wantErr bool
	}{
		{
			name: "time",
			env: map[string]string{
				"TEST_TIME": "Mon Mar 10 11:11:11 2011",
			},
			cfg: &Timed{},
			want: &Timed{
				Time: timeFunc("Mon Mar 10 11:11:11 2011", time.ANSIC),
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

			if err := readEnvVars(tt.cfg, false); (err != nil) != tt.wantErr {
				t.Errorf("wrong error behavior %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.cfg, tt.want) {
				t.Errorf("wrong data %v, want %v", tt.cfg, tt.want)
			}
		})
	}
}
func TestReadEnvVarsWithPrefix(t *testing.T) {
	type Logging struct {
		Debug bool `env:"DEBUG"`
	}

	type DBConfig struct {
		Host    string  `env:"DB_HOST"`
		Port    int     `env:"DB_PORT"`
		Logging Logging `env-prefix:"DB_"`
	}

	type Config struct {
		Default  DBConfig
		ReadOnly DBConfig `env-prefix:"READONLY_"`
		Extra    DBConfig `env-prefix:"EXTRA_"`
	}

	var env = map[string]string{
		"DB_HOST":           "db1.host",
		"DB_PORT":           "10000",
		"DB_DEBUG":          "true",
		"READONLY_DB_HOST":  "db2.host",
		"READONLY_DB_PORT":  "20000",
		"READONLY_DB_DEBUG": "true",
		"EXTRA_DB_HOST":     "db3.host",
		"EXTRA_DB_PORT":     "30000",
		"EXTRA_DB_DEBUG":    "true",
	}
	for k, v := range env {
		os.Setenv(k, v)
	}

	var cfg Config
	if err := readEnvVars(&cfg, false); err != nil {
		t.Fatal("failed to read env vars", err)
	}

	var expected = Config{
		Default: DBConfig{
			Host:    "db1.host",
			Port:    10000,
			Logging: Logging{Debug: true},
		},
		ReadOnly: DBConfig{
			Host:    "db2.host",
			Port:    20000,
			Logging: Logging{Debug: true},
		},
		Extra: DBConfig{
			Host:    "db3.host",
			Port:    30000,
			Logging: Logging{Debug: true},
		},
	}

	if !reflect.DeepEqual(cfg, expected) {
		t.Errorf("wrong data %v, want %v", cfg, expected)
	}
}

type testConfigUpdateFunction struct {
	One   string
	Two   string
	Three string
}

func (f *testConfigUpdateFunction) Update() error {
	f.One = "upd1:" + f.One
	f.Two = "upd2:" + f.Two
	f.Three = "upd3:" + f.Three
	return nil
}

type testConfigUpdateNoFunction struct {
	One   string
	Two   string
	Three string
}

func TestReadUpdateFunctions(t *testing.T) {

	tests := []struct {
		name    string
		cfg     interface{}
		want    interface{}
		wantErr bool
	}{
		{
			name: "update structure with function",
			cfg: &testConfigUpdateFunction{
				One:   "test1",
				Two:   "test2",
				Three: "test3",
			},
			want: &testConfigUpdateFunction{
				One:   "upd1:test1",
				Two:   "upd2:test2",
				Three: "upd3:test3",
			},
			wantErr: false,
		},

		{
			name: "no update",
			cfg: &testConfigUpdateNoFunction{
				One:   "test1",
				Two:   "test2",
				Three: "test3",
			},
			want: &testConfigUpdateNoFunction{
				One:   "test1",
				Two:   "test2",
				Three: "test3",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := readEnvVars(tt.cfg, false); (err != nil) != tt.wantErr {
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
array: [1, 2, 3]`,
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

	t.Run("invalid path", func(t *testing.T) {
		err := parseFile("invalid file path", nil)
		if err == nil {
			t.Error("expected error for invalid file path")
		}
	})
}

func TestParseFileEnv(t *testing.T) {
	type dummy struct{}

	tests := []struct {
		name    string
		rawFile string
		has     map[string]string
		want    map[string]string
		wantErr bool
	}{
		{
			name: "simple file",
			has: map[string]string{
				"TEST1": "aaa",
				"TEST2": "bbb",
				"TEST3": "ccc",
			},
			want: map[string]string{
				"TEST1": "aaa",
				"TEST2": "bbb",
				"TEST3": "ccc",
			},
			wantErr: false,
		},

		{
			name:    "empty file",
			has:     map[string]string{},
			want:    map[string]string{},
			wantErr: false,
		},

		{
			name:    "error",
			rawFile: "-",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := ioutil.TempFile(os.TempDir(), "*.env")
			if err != nil {
				t.Fatal("cannot create temporary file:", err)
			}
			defer os.Remove(tmpFile.Name())

			var file string
			if tt.rawFile == "" {
				for key, val := range tt.has {
					file += fmt.Sprintf("%s=%s\n", key, val)
				}
			} else {
				file = tt.rawFile
			}

			text := []byte(file)
			if _, err = tmpFile.Write(text); err != nil {
				t.Fatal("failed to write to temporary file:", err)
			}

			var cfg dummy
			if err = parseFile(tmpFile.Name(), &cfg); (err != nil) != tt.wantErr {
				t.Errorf("wrong error behavior %v, wantErr %v", err, tt.wantErr)
			}
			for key, val := range tt.has {
				if envVal := os.Getenv(key); err == nil && val != envVal {
					t.Errorf("wrong value %s of var %s, want %s", envVal, key, val)
				}
			}

			os.Clearenv()
		})
	}
}

func TestGetDescription(t *testing.T) {
	type testSingleEnv struct {
		One   int `env:"ONE" env-description:"one"`
		Two   int `env:"TWO" env-description:"two"`
		Three int `env:"THREE" env-description:"three"`
	}

	type testSeveralEnv struct {
		One int `env:"ONE,ENO" env-description:"one"`
		Two int `env:"TWO,OWT" env-description:"two"`
	}

	type testDefaultEnv struct {
		One   int `env:"ONE" env-description:"one" env-default:"1"`
		Two   int `env:"TWO" env-description:"two" env-default:"2"`
		Three int `env:"THREE" env-description:"three" env-default:"3"`
	}

	type testSubOne struct {
		One int `env:"ONE" env-description:"one"`
	}

	type testSubTwo struct {
		Two int `env:"TWO" env-description:"two"`
	}

	type testDeep struct {
		OneStruct testSubOne
		TwoStruct testSubTwo
	}

	type testNoEnv struct {
		One   int
		Two   int
		Three int
	}

	header := "test header:"

	tests := []struct {
		name    string
		cfg     interface{}
		header  *string
		want    string
		wantErr bool
	}{
		{
			name:   "single env",
			cfg:    &testSingleEnv{},
			header: nil,
			want: "Environment variables:" +
				"\n  ONE int\n    \tone" +
				"\n  THREE int\n    \tthree" +
				"\n  TWO int\n    \ttwo",
			wantErr: false,
		},

		{
			name:   "several env",
			cfg:    &testSeveralEnv{},
			header: nil,
			want: "Environment variables:" +
				"\n  ENO int (alternative to ONE)\n    \tone" +
				"\n  ONE int\n    \tone" +
				"\n  OWT int (alternative to TWO)\n    \ttwo" +
				"\n  TWO int\n    \ttwo",
			wantErr: false,
		},

		{
			name:   "default env",
			cfg:    &testDefaultEnv{},
			header: nil,
			want: "Environment variables:" +
				"\n  ONE int\n    \tone (default \"1\")" +
				"\n  THREE int\n    \tthree (default \"3\")" +
				"\n  TWO int\n    \ttwo (default \"2\")",
			wantErr: false,
		},

		{
			name:   "deep structure",
			cfg:    &testDeep{},
			header: nil,
			want: "Environment variables:" +
				"\n  ONE int\n    \tone" +
				"\n  TWO int\n    \ttwo",
			wantErr: false,
		},

		{
			name:    "no env",
			cfg:     &testNoEnv{},
			header:  nil,
			want:    "",
			wantErr: false,
		},

		{
			name:   "custom header",
			cfg:    &testSingleEnv{},
			header: &header,
			want: "test header:" +
				"\n  ONE int\n    \tone" +
				"\n  THREE int\n    \tthree" +
				"\n  TWO int\n    \ttwo",
			wantErr: false,
		},

		{
			name:    "error",
			cfg:     123,
			header:  nil,
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetDescription(tt.cfg, tt.header)
			if (err != nil) != tt.wantErr {
				t.Errorf("%s: wrong error behavior %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("%s: wrong description text %s, want %s", tt.name, got, tt.want)
			}
		})
	}
}

func TestFUsage(t *testing.T) {
	type testSingleEnv struct {
		One   int `env:"ONE" env-description:"one"`
		Two   int `env:"TWO" env-description:"two"`
		Three int `env:"THREE" env-description:"three"`
	}

	customHeader := "test header:"

	tests := []struct {
		name       string
		headerText *string
		usageTexts []string
		want       string
	}{
		{
			name:       "no custom usage",
			headerText: nil,
			usageTexts: nil,
			want: "Environment variables:" +
				"\n  ONE int\n    \tone" +
				"\n  THREE int\n    \tthree" +
				"\n  TWO int\n    \ttwo" +
				"\n",
		},

		{
			name:       "custom header",
			headerText: &customHeader,
			usageTexts: nil,
			want: "test header:" +
				"\n  ONE int\n    \tone" +
				"\n  THREE int\n    \tthree" +
				"\n  TWO int\n    \ttwo" +
				"\n",
		},

		{
			name:       "custom usages",
			headerText: nil,
			usageTexts: []string{
				"test1",
				"test2",
				"test3",
			},
			want: "test1\ntest2\ntest3\n" +
				"\nEnvironment variables:" +
				"\n  ONE int\n    \tone" +
				"\n  THREE int\n    \tthree" +
				"\n  TWO int\n    \ttwo" +
				"\n",
		},

		{
			name:       "custom usages and header",
			headerText: &customHeader,
			usageTexts: []string{
				"test1",
				"test2",
				"test3",
			},
			want: "test1\ntest2\ntest3\n" +
				"\ntest header:" +
				"\n  ONE int\n    \tone" +
				"\n  THREE int\n    \tthree" +
				"\n  TWO int\n    \ttwo" +
				"\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			uFuncs := make([]func(), 0, len(tt.usageTexts))
			for _, text := range tt.usageTexts {
				uFuncs = append(uFuncs, func(a string) func() {
					return func() {
						fmt.Fprintln(w, a)
					}
				}(text))
			}
			var cfg testSingleEnv
			FUsage(w, &cfg, tt.headerText, uFuncs...)()
			gotRaw, _ := ioutil.ReadAll(w)
			got := string(gotRaw)

			if got != tt.want {
				t.Errorf("wrong output %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFUsageNested(t *testing.T) {
	type testNestedEnv struct {
		App struct {
			Port  int `env:"PORT" env-description:"app port"`
			Cache struct {
				Type  string `env:"TYPE" env-description:"cache type"`
				Redis struct {
					Host string `env:"HOST" env-description:"redis host"`
				} `env-prefix:"REDIS_"`
			} `env-prefix:"CACHE_"`
		} `env-prefix:"APP_"`
		Database struct {
			Host string `env:"HOST" env-description:"database host"`
		} `env-prefix:"DATABASE_"`
	}

	customHeader := "test header:"

	tests := []struct {
		name       string
		headerText *string
		usageTexts []string
		want       string
	}{
		{
			name:       "no custom usage",
			headerText: nil,
			usageTexts: nil,
			want: "Environment variables:" +
				"\n  APP_CACHE_REDIS_HOST string\n    \tredis host" +
				"\n  APP_CACHE_TYPE string\n    \tcache type" +
				"\n  APP_PORT int\n    \tapp port" +
				"\n  DATABASE_HOST string\n    \tdatabase host" +
				"\n",
		},

		{
			name:       "custom header",
			headerText: &customHeader,
			usageTexts: nil,
			want: "test header:" +
				"\n  APP_CACHE_REDIS_HOST string\n    \tredis host" +
				"\n  APP_CACHE_TYPE string\n    \tcache type" +
				"\n  APP_PORT int\n    \tapp port" +
				"\n  DATABASE_HOST string\n    \tdatabase host" +
				"\n",
		},

		{
			name:       "custom usages",
			headerText: nil,
			usageTexts: []string{
				"test1",
				"test2",
				"test3",
			},
			want: "test1\ntest2\ntest3\n" +
				"\nEnvironment variables:" +
				"\n  APP_CACHE_REDIS_HOST string\n    \tredis host" +
				"\n  APP_CACHE_TYPE string\n    \tcache type" +
				"\n  APP_PORT int\n    \tapp port" +
				"\n  DATABASE_HOST string\n    \tdatabase host" +
				"\n",
		},

		{
			name:       "custom usages and header",
			headerText: &customHeader,
			usageTexts: []string{
				"test1",
				"test2",
				"test3",
			},
			want: "test1\ntest2\ntest3\n" +
				"\ntest header:" +
				"\n  APP_CACHE_REDIS_HOST string\n    \tredis host" +
				"\n  APP_CACHE_TYPE string\n    \tcache type" +
				"\n  APP_PORT int\n    \tapp port" +
				"\n  DATABASE_HOST string\n    \tdatabase host" +
				"\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			uFuncs := make([]func(), 0, len(tt.usageTexts))
			for _, text := range tt.usageTexts {
				uFuncs = append(uFuncs, func(a string) func() {
					return func() {
						fmt.Fprintln(w, a)
					}
				}(text))
			}
			var cfg testNestedEnv
			FUsage(w, &cfg, tt.headerText, uFuncs...)()
			gotRaw, _ := ioutil.ReadAll(w)
			got := string(gotRaw)

			if got != tt.want {
				t.Errorf("wrong output %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadConfig(t *testing.T) {
	type config struct {
		Number    int64  `edn:"number" yaml:"number" env:"TEST_NUMBER" env-default:"1"`
		String    string `edn:"string" yaml:"string" env:"TEST_STRING" env-default:"default"`
		NoDefault string `edn:"no-default" yaml:"no-default" env:"TEST_NO_DEFAULT"`
		NoEnv     string `edn:"no-env" yaml:"no-env" env-default:"default"`
	}

	tests := []struct {
		name    string
		file    string
		ext     string
		env     map[string]string
		want    *config
		wantErr bool
	}{
		{
			name: "edn_only",
			file: `
			{
				:number 2
				:string "test"
				:no-default "NoDefault"
				:no-env "this"
			}
`,
			ext: "edn",
			env: nil,
			want: &config{
				Number:    2,
				String:    "test",
				NoDefault: "NoDefault",
				NoEnv:     "this",
			},
			wantErr: false,
		},

		{
			name: "edn_and_env",
			file: `
			{
				:number 2
				:string "test"
				:no-default "NoDefault"
				:no-env "this"
			}
`,
			ext: "edn",
			env: map[string]string{
				"TEST_NUMBER": "3",
				"TEST_STRING": "fromEnv",
			},
			want: &config{
				Number:    3,
				String:    "fromEnv",
				NoDefault: "NoDefault",
				NoEnv:     "this",
			},
			wantErr: false,
		},

		{
			name: "yaml_only",
			file: `
number: 2
string: test
no-default: NoDefault
no-env: this
`,
			ext: "yaml",
			env: nil,
			want: &config{
				Number:    2,
				String:    "test",
				NoDefault: "NoDefault",
				NoEnv:     "this",
			},
			wantErr: false,
		},

		{
			name: "env_only",
			file: "none: none",
			ext:  "yaml",
			env: map[string]string{
				"TEST_NUMBER": "2",
				"TEST_STRING": "test",
			},
			want: &config{
				Number:    2,
				String:    "test",
				NoDefault: "",
				NoEnv:     "default",
			},
			wantErr: false,
		},

		{
			name: "yaml_and_env",
			file: `
number: 2
string: test
no-default: NoDefault
no-env: this
`,
			ext: "yaml",
			env: map[string]string{
				"TEST_NUMBER": "3",
				"TEST_STRING": "fromEnv",
			},
			want: &config{
				Number:    3,
				String:    "fromEnv",
				NoDefault: "NoDefault",
				NoEnv:     "this",
			},
			wantErr: false,
		},

		{
			name: "empty",
			file: "none: none",
			ext:  "yaml",
			env:  nil,
			want: &config{
				Number:    1,
				String:    "default",
				NoDefault: "",
				NoEnv:     "default",
			},
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

			for env, val := range tt.env {
				os.Setenv(env, val)
			}
			defer os.Clearenv()

			var cfg config
			if err = ReadConfig(tmpFile.Name(), &cfg); (err != nil) != tt.wantErr {
				t.Errorf("wrong error behavior %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && !reflect.DeepEqual(&cfg, tt.want) {
				t.Errorf("wrong data %v, want %v", &cfg, tt.want)
			}
		})
	}
}

// TestTimeLocation tests *time.Location parse. It is  a pointer type,
// so we need to compare it with pointer manually,
// because reflect.DeepEqual() compares only pointer values, not their structs
func TestTimeLocation(t *testing.T) {
	want := time.UTC

	var S struct {
		Location *time.Location `env:"TEST_LOCATION"`
	}

	os.Setenv("TEST_LOCATION", "UTC")
	defer os.Clearenv()

	if err := ReadEnv(&S); err != nil {
		t.Fatal("cannot read env:", err)
	}

	if want != S.Location {
		t.Errorf("wrong location pointers: got %p, want %p", S.Location, want)
	}
}

func TestSkipUnexportedField(t *testing.T) {
	conf := struct {
		Database struct {
			Host string `yaml:"host" env:"DB_HOST" env-description:"Database host"`
			Port string `yaml:"port" env:"DB_PORT" env-description:"Database port"`
		} `yaml:"database"`
		server struct {
			Host string `yaml:"host" env:"SRV_HOST,HOST" env-description:"Server host" env-default:"localhost"`
			Port string `yaml:"port" env:"SRV_PORT,PORT" env-description:"Server port" env-default:"8080"`
		} `yaml:"server"`
	}{}

	if err := ReadConfig("example/simple_config/config.yml", &conf); err != nil {
		t.Fatal(err)
	}
	if conf.server.Host != "" || conf.server.Port != "" {
		t.Fatal("unexpect value on unexported fields")
	}
	if conf.Database.Host == "" || conf.Database.Port == "" {
		t.Fatal("expect value on exported fields")
	}
}

func TestParseYAML(t *testing.T) {
	type configObject struct {
		One int `yaml:"one"`
		Two int `yaml:"two"`
	}
	type config struct {
		Number  int64        `yaml:"number"`
		Float   float64      `yaml:"float"`
		String  string       `yaml:"string"`
		Boolean bool         `yaml:"boolean"`
		Object  configObject `yaml:"object"`
		Array   []int        `yaml:"array"`
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
		r       io.Reader
		want    *config
		wantErr bool
	}{
		{
			name: "yaml",
			r: bytes.NewBufferString(`
number: 1
float: 2.3
string: test
boolean: yes
object:
  one: 1
  two: 2
array: [1, 2, 3]`),
			want:    &wantConfig,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg config
			var err error
			if err = ParseYAML(tt.r, &cfg); (err != nil) != tt.wantErr {
				t.Errorf("ParseYAML() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && !reflect.DeepEqual(&cfg, tt.want) {
				t.Errorf("wrong data %v, want %v", &cfg, tt.want)
			}
		})
	}
}

func TestParseJSON(t *testing.T) {
	type configObject struct {
		One int `json:"one"`
		Two int `json:"two"`
	}
	type config struct {
		Number  int64        `json:"number"`
		Float   float64      `json:"float"`
		String  string       `json:"string"`
		Boolean bool         `json:"boolean"`
		Object  configObject `json:"object"`
		Array   []int        `json:"array"`
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
		r       io.Reader
		want    *config
		wantErr bool
	}{
		{
			name: "json",
			r: bytes.NewBufferString(`
{
	"number": 1,
	"float": 2.3,
	"string": "test",
	"boolean": true,
	"object": {
		"one": 1,
		"two": 2
	},
	"array": [1, 2, 3]
}`),
			want:    &wantConfig,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg config
			var err error
			if err := ParseJSON(tt.r, &cfg); (err != nil) != tt.wantErr {
				t.Errorf("ParseJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && !reflect.DeepEqual(&cfg, tt.want) {
				t.Errorf("wrong data %v, want %v", &cfg, tt.want)
			}
		})
	}
}

func TestParseTOML(t *testing.T) {
	type configObject struct {
		One int `toml:"one"`
		Two int `toml:"two"`
	}
	type config struct {
		Number  int64        `toml:"number"`
		Float   float64      `toml:"float"`
		String  string       `toml:"string"`
		Boolean bool         `toml:"boolean"`
		Object  configObject `toml:"object"`
		Array   []int        `toml:"array"`
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
		r       io.Reader
		want    *config
		wantErr bool
	}{
		{
			name: "toml",
			r: bytes.NewBufferString(`
number = 1
float = 2.3
string = "test"
boolean = true
array = [1, 2, 3]
[object]
one = 1
two = 2`),
			want:    &wantConfig,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg config
			var err error
			if err := ParseTOML(tt.r, &cfg); (err != nil) != tt.wantErr {
				t.Errorf("ParseTOML() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && !reflect.DeepEqual(&cfg, tt.want) {
				t.Errorf("wrong data %v, want %v", &cfg, tt.want)
			}
		})
	}
}

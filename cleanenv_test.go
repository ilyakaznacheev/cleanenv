package cleanenv

import (
	"os"
	"reflect"
	"testing"
)

func TestReadEnvVars(t *testing.T) {
	type testStruct struct {
		number int `env:"TEST" default:"1"`
	}

	type testStructExp struct {
		Number int `env:"TEST" default:"1"`
	}

	type testStructComb struct {
		Number int `env:"TEST1" default:"1"`
		number int `env:"TEST2" default:"1"`
	}

	tests := []struct {
		name    string
		env     map[string]string
		cfg     interface{}
		want    interface{}
		wantErr bool
	}{
		{
			name: "unexported",
			env: map[string]string{
				"TEST": "2",
			},
			cfg: &testStruct{},
			want: &testStruct{
				number: 0,
			},
			wantErr: false,
		},

		{
			name: "exported",
			env: map[string]string{
				"TEST": "2",
			},
			cfg: &testStructExp{},
			want: &testStructExp{
				Number: 2,
			},
			wantErr: false,
		},

		{
			name: "combined",
			env: map[string]string{
				"TEST1": "2",
				"TEST2": "2",
			},
			cfg: &testStructComb{},
			want: &testStructComb{
				number: 0,
				Number: 2,
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

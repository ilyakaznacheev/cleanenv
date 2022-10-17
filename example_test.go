package cleanenv_test

import (
	"flag"
	"fmt"
	"net/url"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

// ExampleGetDescription builds a description text from structure tags
func ExampleGetDescription() {
	type config struct {
		One   int64   `env:"ONE" env-description:"first parameter"`
		Two   float64 `env:"TWO" env-description:"second parameter"`
		Three string  `env:"THREE" env-description:"third parameter"`
	}

	var cfg config

	text, err := cleanenv.GetDescription(&cfg, nil)
	if err != nil {
		panic(err)
	}

	fmt.Println(text)
	// Output: Environment variables:
	//   ONE int64
	//     	first parameter
	//   TWO float64
	//     	second parameter
	//   THREE string
	//     	third parameter
}

// ExampleGetDescription_defaults builds a description text from structure tags with description of default values
func ExampleGetDescription_defaults() {
	type config struct {
		One   int64   `env:"ONE" env-description:"first parameter" env-default:"1"`
		Two   float64 `env:"TWO" env-description:"second parameter" env-default:"2.2"`
		Three string  `env:"THREE" env-description:"third parameter" env-default:"test"`
	}

	var cfg config

	text, err := cleanenv.GetDescription(&cfg, nil)
	if err != nil {
		panic(err)
	}

	fmt.Println(text)
	// Output: Environment variables:
	//   ONE int64
	//     	first parameter (default "1")
	//   TWO float64
	//     	second parameter (default "2.2")
	//   THREE string
	//     	third parameter (default "test")
}

// ExampleGetDescription_variableList builds a description text from structure tags with description of alternative variables
func ExampleGetDescription_variableList() {
	type config struct {
		FirstVar int64 `env:"ONE,TWO,THREE" env-description:"first found parameter"`
	}

	var cfg config

	text, err := cleanenv.GetDescription(&cfg, nil)
	if err != nil {
		panic(err)
	}

	fmt.Println(text)
	// Output: Environment variables:
	//   ONE int64
	//     	first found parameter
	//   TWO int64 (alternative to ONE)
	//     	first found parameter
	//   THREE int64 (alternative to ONE)
	//     	first found parameter
}

// ExampleGetDescription_customHeaderText builds a description text from structure tags with custom header string
func ExampleGetDescription_customHeaderText() {
	type config struct {
		One   int64   `env:"ONE" env-description:"first parameter"`
		Two   float64 `env:"TWO" env-description:"second parameter"`
		Three string  `env:"THREE" env-description:"third parameter"`
	}

	var cfg config

	header := "Custom header text:"

	text, err := cleanenv.GetDescription(&cfg, &header)
	if err != nil {
		panic(err)
	}

	fmt.Println(text)
	// Output: Custom header text:
	//   ONE int64
	//     	first parameter
	//   TWO float64
	//     	second parameter
	//   THREE string
	//     	third parameter
}

// ExampleUpdateEnv updates variables in the configuration structure.
// Only variables with `env-upd:""` tag will be updated
func ExampleUpdateEnv() {
	type config struct {
		One int64 `env:"ONE"`
		Two int64 `env:"TWO" env-upd:""`
	}

	var cfg config

	// set environment variables
	os.Setenv("ONE", "1")
	os.Setenv("TWO", "2")

	// read environment variables into the structure
	cleanenv.ReadEnv(&cfg)
	fmt.Printf("%+v\n", cfg)

	// update environment variables
	os.Setenv("ONE", "11")
	os.Setenv("TWO", "22")

	// update only updatable environment variables in the structure
	cleanenv.UpdateEnv(&cfg)
	fmt.Printf("%+v\n", cfg)

	// Output: {One:1 Two:2}
	// {One:1 Two:22}

}

// ExampleReadEnv reads environment variables or default values into the structure
func ExampleReadEnv() {
	type config struct {
		Port     string `env:"PORT" env-default:"5432"`
		Host     string `env:"HOST" env-default:"localhost"`
		Name     string `env:"NAME" env-default:"postgres"`
		User     string `env:"USER" env-default:"user"`
		Password string `env:"PASSWORD"`
	}

	var cfg config

	os.Setenv("PORT", "5050")
	os.Setenv("NAME", "redis")
	os.Setenv("USER", "tester")
	os.Setenv("PASSWORD", "*****")

	cleanenv.ReadEnv(&cfg)
	fmt.Printf("%+v\n", cfg)

	// Output: {Port:5050 Host:localhost Name:redis User:tester Password:*****}
}

// ExampleReadEnv reads environment variables or default values into the structure
func ExampleReadEnvWithURL() {
	type config struct {
		ImageCDN url.URL `env:"IMAGE_CDN"`
	}

	var cfg config

	os.Setenv("IMAGE_CDN", "https://images.cdn/")

	cleanenv.ReadEnv(&cfg)
	fmt.Printf("%+v\n", cfg.ImageCDN.String())

	// Output: https://images.cdn/
}

// ExampleReadEnvWithNamespace reads environment variables prefixed with the given string
func ExampleReadEnvWithNamespace() {
	type commonConfig struct {
		Endpoint string `env:"ENDPOINT"`
	}

	var cfg1 commonConfig
	var cfg2 commonConfig

	os.Setenv("SERVICE1_ENDPOINT", "https://example.com/api/v1/doit")
	os.Setenv("SERVICE2_ENDPOINT", "https://somewhere.else.entire.ly:8011/getit/here")

	cleanenv.ReadEnv(&cfg1, "SERVICE1_")
	cleanenv.ReadEnv(&cfg2, "SERVICE2_")
	fmt.Printf("%+v\n", cfg1)
	fmt.Printf("%+v\n", cfg2)

	// Output:
	//
	// {Endpoint:https://example.com/api/v1/doit}
	// {Endpoint:https://somewhere.else.entire.ly:8011/getit/here}
}

// MyField is an example type with a custom setter
type MyField string

func (f *MyField) SetValue(s string) error {
	if s == "" {
		return fmt.Errorf("field value can't be empty")
	}
	*f = MyField("my field is: " + s)
	return nil
}

func (f MyField) String() string {
	return string(f)
}

// Example_setter uses type with a custom setter to parse environment variable data
func Example_setter() {
	type config struct {
		Default string  `env:"ONE"`
		Custom  MyField `env:"TWO"`
	}

	var cfg config

	os.Setenv("ONE", "test1")
	os.Setenv("TWO", "test2")

	cleanenv.ReadEnv(&cfg)
	fmt.Printf("%+v\n", cfg)
	// Output: {Default:test1 Custom:my field is: test2}
}

// ConfigUpdate is a type with a custom updater
type ConfigUpdate struct {
	Default string `env:"DEFAULT"`
	Custom  string
}

func (c *ConfigUpdate) Update() error {
	c.Custom = "custom"
	return nil
}

// Example_updater uses a type with a custom updater
func Example_updater() {
	var cfg ConfigUpdate

	os.Setenv("DEFAULT", "default")

	cleanenv.ReadEnv(&cfg)
	fmt.Printf("%+v\n", cfg)
	// Output: {Default:default Custom:custom}
}

func ExampleUsage() {
	os.Stderr = os.Stdout // replace STDERR with STDOUT for test

	type config struct {
		One   int64   `env:"ONE" env-description:"first parameter"`
		Two   float64 `env:"TWO" env-description:"second parameter"`
		Three string  `env:"THREE" env-description:"third parameter"`
	}

	// setup flags
	fset := flag.NewFlagSet("Example", flag.ContinueOnError)

	fset.String("p", "8080", "service port")
	fset.String("h", "localhost", "service host")

	var cfg config
	customHeader := "My sweet variables:"

	// get config usage with wrapped flag usage and custom header string
	u := cleanenv.Usage(&cfg, &customHeader, fset.Usage)

	// print usage to STDERR
	u()

	// Output: Usage of Example:
	//   -h string
	//     	service host (default "localhost")
	//   -p string
	//     	service port (default "8080")
	//
	// My sweet variables:
	//   ONE int64
	//     	first parameter
	//   TWO float64
	//     	second parameter
	//   THREE string
	//     	third parameter
}

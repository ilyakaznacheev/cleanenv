/*
Package cleanenv gives you a single tool to read application configuration from several sources with ease.

Features

- read from several file formats (YAML, JSON, TOML, ENV, EDN) and parse into the internal structure;

- read environment variables into the internal structure;

- output environment variable list with descriptions into help output;

- custom variable readers (e.g. if you want to read from remote config server, etc).

Usage

You can just prepare the config structure and fill it from the config file and environment variables.

	type Config struct {
		Port string `yaml:"port" env:"PORT" env-default:"8080"`
		Host string `yaml:"host" env:"HOST" env-default:"localhost"`
	}

	var cfg Config

	ReadConfig("config.yml", &cfg)

Help output

You can list all of your environment variables by means of help output:

	type ConfigServer struct {
		Port     string `env:"PORT" env-description:"server port"`
		Host     string `env:"HOST" env-description:"server host"`
	}

	var cfg ConfigRemote

	help, err := cleanenv.GetDescription(&cfg, nil)
	if err != nil {
		...
	}

	// setup help output
	f := flag.NewFlagSet("Example app", 1)
	fu := f.Usage
	f.Usage = func() {
		fu()
		envHelp, _ := cleanenv.GetDescription(&cfg, nil)
		fmt.Fprintln(f.Output())
		fmt.Fprintln(f.Output(), envHelp)
	}

	f.Parse(os.Args[1:])

Then run go run main.go -h and the output will include:

	Environment variables:
		PORT  server port
		HOST  server host

For more detailed information check examples and example tests.
*/
package cleanenv

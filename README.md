# Clean Env

Minimalistic configuration reader

It just does the following:

- reads and parses configuration file
- reads and overwrites it from environment variables

## Example

```go
type Config struct {
	Port string `yml:"port" env:"PORT" default:"8080"`
	Host string `yml:"host" env:"HOST" default:"localhost"`
}

var cfg Config

err := ReadConfig("config.yml", &Config)
if err != nil {
    ...
}
```

This code will try to read and parse the configuration file `config.yml` as the structure is described in the `Config` structure. Then it will overwrite fields from available environment variables (`PORT`, `HOST`).

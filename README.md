# Clean Env

Minimalistic configuration reader

[![GoDoc](https://godoc.org/github.com/ilyakaznacheev/cleanenv?status.svg)](https://godoc.org/github.com/ilyakaznacheev/cleanenv)
[![Go Report Card](https://goreportcard.com/badge/github.com/ilyakaznacheev/cleanenv)](https://goreportcard.com/report/github.com/ilyakaznacheev/cleanenv)
[![Coverage Status](https://codecov.io/github/ilyakaznacheev/cleanenv/coverage.svg?branch=master)](https://codecov.io/gh/ilyakaznacheev/cleanenv)
[![Build Status](https://travis-ci.org/ilyakaznacheev/cleanenv.svg?branch=master)](https://travis-ci.org/ilyakaznacheev/cleanenv)
[![Release](https://img.shields.io/github/release/ilyakaznacheev/cleanenv.svg)](https://github.com/ilyakaznacheev/cleanenv/releases/)
[![License](https://img.shields.io/github/license/ilyakaznacheev/cleanenv.svg)](https://github.com/ilyakaznacheev/cleanenv/blob/master/LICENSE)

## Overview

This is a simple configuration reading tool. It just does the following:

- reads and parses configuration structure from the file
- reads and overwrites configuration structure from environment variables

## Content

- [Installation](#installation)
- [Usage](#usage)
    - [Read Configuration](#read-configuration)
    - [Read Environment Variables Only](#read-environment-variables-only)
    - [Update Environment Variables](#update-environment-variables)
    - [Description](#description)
- [Model Format](#model-format)
- [Custom Functions](#custom-functions)
    - [Custom Value Setter](#custom-value-setter)
    - [Custom Value Update](#custom-value-update)
- [Supported File Formats](#supported-file-formats)
- [Examples](#examples)
- [Contribution](#contribution)

## Installation

To install the package run

```bash
go get -u github.com/ilyakaznacheev/cleanenv
```

## Usage

The package is oriented to be simple in use and explicitness.

The main idea is to use a structured configuration variable instead of any sort of dynamic set of configuration fields like some libraries does, to avoid unnecessary type conversions and move the configuration through the program as a simple structure, not as an object with complex behavior.

There are just several actions you can do with this tool and probably only things you want to do with your config if your application is not too complicated.

- read configuration file
- read environment variables
- read some environment variables again

### Read Configuration

You can read a configuration file and environment variables in a single function call.

```go
import github.com/ilyakaznacheev/cleanenv

type ConfigDatabase struct {
	Port     string `yml:"port" env:"PORT" env-default:"5432"`
	Host     string `yml:"host" env:"HOST" env-default:"localhost"`
	Name     string `yml:"name" env:"NAME" env-default:"postgres"`
	User     string `yml:"user" env:"USER" env-default:"user"`
	Password string `yml:"password" env:"PASSWORD"`
}

var cfg ConfigDatabase

err := cleanenv.ReadConfig("config.yml", &cfg)
if err != nil {
    ...
}
```

This will do the following:

1. parse configuration file according to YAML format (`yaml` tag in this case);
1. reads environment variables and overwrites values from the file with the values which was found in the environment (`env` tag);
1. if no value was found on the first two steps, the field will be filled with the default value (`env-default` tag) if it is set.

### Read Environment Variables Only

Sometimes you don't want to use configuration files at all, or you may want to use `.env` file format instead. Thus, you can limit yourself with only reading environment variables:

```go 
import github.com/ilyakaznacheev/cleanenv

type ConfigDatabase struct {
	Port     string `env:"PORT" env-default:"5432"`
	Host     string `env:"HOST" env-default:"localhost"`
	Name     string `env:"NAME" env-default:"postgres"`
	User     string `env:"USER" env-default:"user"`
	Password string `env:"PASSWORD"`
}

var cfg ConfigDatabase

err := cleanenv.ReadEnv(&cfg)
if err != nil {
    ...
}
```

### Update Environment Variables

Some environment variables may change during the application run. To get the new values you need to mark these variables as updatable with the tag `env-upd` and then run the update function:

```go 
import github.com/ilyakaznacheev/cleanenv

type ConfigRemote struct {
	Port     string `env:"PORT" env-upd`
    Host     string `env:"HOST" env-upd`
    UserName string `env:"USERNAME"`
}

var cfg ConfigRemote

cleanenv.ReadEnv(&cfg)

// ... some actions in-between

err := cleanenv.UpdateEnv(&cfg)
if err != nil {
    ...
}
```

Here remote host and port may change in a distributed system architecture. Fields `cfg.Port` and `cfg.Host` can be updated in the runtime from corresponding environment variables. You can update them before the remote service call. Field `cfg.UserName` will not be changed after the initial read, though.

### Description

You can get descriptions of all environment variables to use them in help documentation.

```go
import github.com/ilyakaznacheev/cleanenv

type ConfigServer struct {
    Port     string `env:"PORT" env-description:"server port"`
    Host     string `env:"HOST" env-description:"server host"`
}

var cfg ConfigRemote

help, err := cleanenv.GetDescription(&cfg, nil)
if err != nil {
    ...
}
```

You will get the following:

```
Environment variables:
  PORT  server port
  HOST  server host
```

## Model Format

Library uses tags to configure model of configuration structure. There are following tags:

- `env="<name>"` - environment variable name (e.g. `env="PORT"`);
- `env-upd` - flag to mark a field as updatable. Run `UpdateEnv(&cfg)` to refresh updatable variables from environment;
- `env-default="<value>"` - default value. If the field wasn't filled from the environment variable default value will be used instead;
- `env-separator="<value>"` - custom list and map separator. If not set, the default separator `,` will be used;
- `env-description="<value>"` - environment variable description.

## Custom Functions

To enhance package abilities you can use some custom functions.

### Custom Value Setter

To make custom type allows to set the value from the environment variable, you need to implement the `Setter` interface on the field level:

```go
type MyField string

func (f MyField) SetValue(s string) error  {
	if s == "" {
		return fmt.Errorf("field value can't be empty")
	}
	f = MyField("my field is: "+ s)
	return nil
}

type Config struct {
    Field MyField `env="MY_VALUE"`
}
```

`SetValue` method should implement conversion logic from string to custom type.

### Custom Value Update

You may need to execute some custom field update logic, e.g. for remote config load.

Thus, you need to implement the `Updater` interface on the structure level:

```go
type Config struct {
	Field string
}

func (c *Config) Update() error {
    newField, err := SomeCustomUpdate()
    f.Field = newField
	return err
}
```

## Supported File Formats

There are several most popular config file formats supported:

- YAML
- JSON
- TOML

## Examples

```go
type Config struct {
	Port string `yml:"port" env:"PORT" env-default:"8080"`
	Host string `yml:"host" env:"HOST" env-default:"localhost"`
}

var cfg Config

err := ReadConfig("config.yml", &cfg)
if err != nil {
    ...
}
```

This code will try to read and parse the configuration file `config.yml` as the structure is described in the `Config` structure. Then it will overwrite fields from available environment variables (`PORT`, `HOST`).

For more details check the [example](/example) directory.

## Contribution

The tool is open-sourced under the [MIT](LICENSE) license.

If you will find some error, want to add something or ask a question - feel free to create an issue and/or make a pull request.

Any contribution is welcome.

## Thanks

Big thanks to a project [kelseyhightower/envconfig](https://github.com/kelseyhightower/envconfig) for inspiration.
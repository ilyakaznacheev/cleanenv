/*
Package cleanenv gives you a single tool to read application configuration from several sources.

You can just prepare config structure and fill it from the config file and environment variables.

	type Config struct {
		Port string `yml:"port" env:"PORT" default:"8080"`
		Host string `yml:"host" env:"HOST" default:"localhost"`
	}

	var cfg Config

	ReadConfig("config.yml", &cfg)
*/
package cleanenv

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

var (
	ErrInconsistentType = errors.New("error during parsing config structure metadata")
)

const (
	DefaultSeparator = ","
)

type CustomSetter interface {
	SetValue(string) error
}

// ReadConfig reads configuration file and parses it depending on tags in structure provided.
// Then it reads and parses
func ReadConfig(path string, cfg interface{}) error {
	// read the configuration file
	f, err := os.OpenFile(path, os.O_RDONLY|os.O_SYNC, 0)
	if err != nil {
		return err
	}
	defer f.Close()

	// parse the file depending on the file type
	switch ext := strings.ToLower(filepath.Ext(path)); ext {
	case "yaml", "yml":
		err = parseYAML(f, cfg)
	case "json":
		err = parseYAML(f, cfg)
	default:
		return fmt.Errorf("file format %s doesn't supported by the parser", ext)
	}
	if err != nil {
		return fmt.Errorf("config file parsing error: %s", err.Error())
	}

	return readEnvVars(cfg)
}

// parseYAML parses YAML from reader to data structure
func parseYAML(r io.Reader, str interface{}) error {
	return yaml.NewDecoder(r).Decode(str)
}

// parseJSON parses JSON from reader to data structure
func parseJSON(r io.Reader, str interface{}) error {
	return json.NewDecoder(r).Decode(str)
}

type structMeta struct {
	envList []string
	// fieldType   reflect.Type
	fieldValue  reflect.Value
	defValue    *string
	separator   string
	description string
}

func readStructMetadata(cfg interface{}) ([]structMeta, error) {
	s := reflect.ValueOf(cfg)

	// check that under interface we have a pointer to the data
	if s.Kind() != reflect.Ptr {
		return nil, ErrInconsistentType
	}
	s = s.Elem()

	// process only structures
	if s.Kind() != reflect.Struct {
		return nil, ErrInconsistentType
	}
	typeInfo := s.Type()

	metas := make([]structMeta, 0)

	// read tags
	for idx := 0; idx < s.NumField(); idx++ {
		fType := typeInfo.Field(idx)

		// don't process the field if it hasn't explicit environment variable name
		if envName, ok := fType.Tag.Lookup("env"); ok {
			var (
				defValue  *string
				separator string
			)

			// if unicode.IsLower([]rune(fType.Name)[0]) {
			// 	// unexported field
			// 	continue
			// }
			if !s.Field(idx).CanSet() {
				continue
			}

			if def, ok := fType.Tag.Lookup("default"); ok {
				defValue = &def
			}

			if sep, ok := fType.Tag.Lookup("separator"); ok {
				separator = sep
			} else {
				separator = DefaultSeparator
			}

			metas = append(metas, structMeta{
				envList: strings.Split(envName, DefaultSeparator),
				// fieldType:   typeInfo,
				fieldValue:  s.Field(idx),
				defValue:    defValue,
				separator:   separator,
				description: fType.Tag.Get("descr"),
			})
		}
	}

	return metas, nil
}

func readEnvVars(cfg interface{}) error {
	metaInfo, err := readStructMetadata(cfg)
	if err != nil {
		return err
	}

	for _, meta := range metaInfo {
		var rawValue string
		if meta.defValue != nil {
			rawValue = *meta.defValue
		}

		for _, env := range meta.envList {
			if value, ok := os.LookupEnv(env); ok {
				rawValue = value
				break
			}
		}

		if err := parseValue(meta.fieldValue, rawValue, meta.separator); err != nil {
			return err
		}
	}

	return nil
}

// parseValue parses value into the corresponding field.
// In case of maps and slices it uses provided separator to split raw value string
func parseValue(field reflect.Value, value, sep string) error {

	if field.CanInterface() {
		if cs, ok := field.Interface().(CustomSetter); ok {
			return cs.SetValue(value)
		}
	}

	valueType := field.Type()

	switch valueType.Kind() {
	// parse string value
	case reflect.String:
		field.SetString(value)

	// parse boolean value
	case reflect.Bool:
		if b, err := strconv.ParseBool(value); err != nil {
			return err
		} else {
			field.SetBool(b)
		}

	// parse integer (or time) value
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if field.Kind() == reflect.Int64 && valueType.PkgPath() == "time" && valueType.Name() == "Duration" {
			// try to parse time
			if d, err := time.ParseDuration(value); err != nil {
				return err
			} else {
				field.SetInt(int64(d))
			}
		} else {
			// parse regular integer
			if number, err := strconv.ParseInt(value, 0, valueType.Bits()); err != nil {
				return err
			} else {
				field.SetInt(number)
			}
		}

	// parse unsigned integer value
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if number, err := strconv.ParseUint(value, 0, valueType.Bits()); err != nil {
			return err
		} else {
			field.SetUint(number)
		}

	// parse floating point value
	case reflect.Float32, reflect.Float64:
		if number, err := strconv.ParseFloat(value, valueType.Bits()); err != nil {
			return err
		} else {
			field.SetFloat(number)
		}

	// parse sliced value
	case reflect.Slice:
		sliceValue := reflect.MakeSlice(valueType, 0, 0)

		if valueType.Elem().Kind() == reflect.Uint8 {
			sliceValue = reflect.ValueOf([]byte(value))
		} else if len(strings.TrimSpace(value)) != 0 {
			values := strings.Split(value, sep)
			sliceValue = reflect.MakeSlice(valueType, len(values), len(values))

			for i, val := range values {
				if err := parseValue(sliceValue.Index(i), val, sep); err != nil {
					return err
				}
			}
		}

		field.Set(sliceValue)

	// parse mapped value
	case reflect.Map:
		mapValue := reflect.MakeMap(valueType)
		if len(strings.TrimSpace(value)) != 0 {
			pairs := strings.Split(value, sep)
			for _, pair := range pairs {
				kvPair := strings.Split(pair, ":")
				if len(kvPair) != 2 {
					return fmt.Errorf("invalid map item: %q", pair)
				}
				k := reflect.New(valueType.Key()).Elem()
				err := parseValue(k, kvPair[0], sep)
				if err != nil {
					return err
				}
				v := reflect.New(valueType.Elem()).Elem()
				err = parseValue(v, kvPair[1], sep)
				if err != nil {
					return err
				}
				mapValue.SetMapIndex(k, v)
			}
		}

	default:
		return fmt.Errorf("unsupported type %s.%s", valueType.PkgPath(), valueType.Name())
	}

	return nil
}

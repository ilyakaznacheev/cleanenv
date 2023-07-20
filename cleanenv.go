package cleanenv

import (
	"encoding"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
	"olympos.io/encoding/edn"
)

const (
	// DefaultSeparator is a default list and map separator character
	DefaultSeparator = ","
)

// Supported tags
const (
	// TagEnv name of the environment variable or a list of names
	TagEnv = "env"

	// TagEnvLayout value parsing layout (for types like time.Time)
	TagEnvLayout = "env-layout"

	// TagEnvDefault default value
	TagEnvDefault = "env-default"

	// TagEnvSeparator custom list and map separator
	TagEnvSeparator = "env-separator"

	// TagEnvDescription environment variable description
	TagEnvDescription = "env-description"

	// TagEnvUpd flag to mark a field as updatable
	TagEnvUpd = "env-upd"

	// TagEnvRequired flag to mark a field as required
	TagEnvRequired = "env-required"

	// TagEnvPrefix аlag to specify prefix for structure fields
	TagEnvPrefix = "env-prefix"
)

// Setter is an interface for a custom value setter.
//
// To implement a custom value setter you need to add a SetValue function to your type that will receive a string raw value:
//
//	type MyField string
//
//	func (f *MyField) SetValue(s string) error {
//		if s == "" {
//			return fmt.Errorf("field value can't be empty")
//		}
//		*f = MyField("my field is: " + s)
//		return nil
//	}
type Setter interface {
	SetValue(string) error
}

// Updater gives an ability to implement custom update function for a field or a whole structure
type Updater interface {
	Update() error
}

// ReadConfig reads configuration file and parses it depending on tags in structure provided.
// Then it reads and parses
//
// Example:
//
//	type ConfigDatabase struct {
//		Port     string `yaml:"port" env:"PORT" env-default:"5432"`
//		Host     string `yaml:"host" env:"HOST" env-default:"localhost"`
//		Name     string `yaml:"name" env:"NAME" env-default:"postgres"`
//		User     string `yaml:"user" env:"USER" env-default:"user"`
//		Password string `yaml:"password" env:"PASSWORD"`
//	}
//
//	var cfg ConfigDatabase
//
//	err := cleanenv.ReadConfig("config.yml", &cfg)
//	if err != nil {
//	    ...
//	}
func ReadConfig(path string, cfg interface{}) error {
	err := parseFile(path, cfg)
	if err != nil {
		return err
	}

	return readEnvVars(cfg, false)
}

// ReadEnv reads environment variables into the structure.
func ReadEnv(cfg interface{}) error {
	return readEnvVars(cfg, false)
}

// UpdateEnv rereads (updates) environment variables in the structure.
func UpdateEnv(cfg interface{}) error {
	return readEnvVars(cfg, true)
}

// parseFile parses configuration file according to it's extension
//
// Currently following file extensions are supported:
//
// - yaml
//
// - json
//
// - toml
//
// - env
//
// - edn
func parseFile(path string, cfg interface{}) error {
	// open the configuration file
	f, err := os.OpenFile(path, os.O_RDONLY|os.O_SYNC, 0)
	if err != nil {
		return err
	}
	defer f.Close()

	// parse the file depending on the file type
	switch ext := strings.ToLower(filepath.Ext(path)); ext {
	case ".yaml", ".yml":
		err = ParseYAML(f, cfg)
	case ".json":
		err = ParseJSON(f, cfg)
	case ".toml":
		err = ParseTOML(f, cfg)
	case ".edn":
		err = parseEDN(f, cfg)
	case ".env":
		err = parseENV(f, cfg)
	default:
		return fmt.Errorf("file format '%s' doesn't supported by the parser", ext)
	}
	if err != nil {
		return fmt.Errorf("config file parsing error: %s", err.Error())
	}
	return nil
}

// ParseYAML parses YAML from reader to data structure
func ParseYAML(r io.Reader, str interface{}) error {
	return yaml.NewDecoder(r).Decode(str)
}

// ParseJSON parses JSON from reader to data structure
func ParseJSON(r io.Reader, str interface{}) error {
	return json.NewDecoder(r).Decode(str)
}

// ParseTOML parses TOML from reader to data structure
func ParseTOML(r io.Reader, str interface{}) error {
	_, err := toml.NewDecoder(r).Decode(str)
	return err
}

// parseEDN parses EDN from reader to data structure
func parseEDN(r io.Reader, str interface{}) error {
	return edn.NewDecoder(r).Decode(str)
}

// parseENV, in fact, doesn't fill the structure with environment variable values.
// It just parses ENV file and sets all variables to the environment.
// Thus, the structure should be filled at the next steps.
func parseENV(r io.Reader, _ interface{}) error {
	vars, err := godotenv.Parse(r)
	if err != nil {
		return err
	}

	for env, val := range vars {
		if err = os.Setenv(env, val); err != nil {
			return fmt.Errorf("set environment: %w", err)
		}
	}

	return nil
}

// parseSlice parses value into a slice of given type
func parseSlice(valueType reflect.Type, value string, sep string, layout *string) (*reflect.Value, error) {
	sliceValue := reflect.MakeSlice(valueType, 0, 0)
	if valueType.Elem().Kind() == reflect.Uint8 {
		sliceValue = reflect.ValueOf([]byte(value))
	} else if len(strings.TrimSpace(value)) != 0 {
		values := strings.Split(value, sep)
		sliceValue = reflect.MakeSlice(valueType, len(values), len(values))

		for i, val := range values {
			if err := parseValue(sliceValue.Index(i), val, sep, layout); err != nil {
				return nil, err
			}
		}
	}
	return &sliceValue, nil
}

// parseMap parses value into a map of given type
func parseMap(valueType reflect.Type, value string, sep string, layout *string) (*reflect.Value, error) {
	mapValue := reflect.MakeMap(valueType)
	if len(strings.TrimSpace(value)) != 0 {
		pairs := strings.Split(value, sep)
		for _, pair := range pairs {
			kvPair := strings.SplitN(pair, ":", 2)
			if len(kvPair) != 2 {
				return nil, fmt.Errorf("invalid map item: %q", pair)
			}
			k := reflect.New(valueType.Key()).Elem()
			err := parseValue(k, kvPair[0], sep, layout)
			if err != nil {
				return nil, err
			}
			v := reflect.New(valueType.Elem()).Elem()
			err = parseValue(v, kvPair[1], sep, layout)
			if err != nil {
				return nil, err
			}
			mapValue.SetMapIndex(k, v)
		}
	}
	return &mapValue, nil
}

// structMeta is a structure metadata entity
type structMeta struct {
	envList     []string
	fieldName   string
	fieldValue  reflect.Value
	defValue    *string
	layout      *string
	separator   string
	description string
	updatable   bool
	required    bool
}

// isFieldValueZero determines if fieldValue empty or not
func (sm *structMeta) isFieldValueZero() bool {
	return sm.fieldValue.IsZero()
}

// parseFunc custom value parser function
type parseFunc func(*reflect.Value, string, *string) error

// Any specific supported struct can be added here
var validStructs = map[reflect.Type]parseFunc{

	reflect.TypeOf(time.Time{}): func(field *reflect.Value, value string, layout *string) error {
		var l string
		if layout != nil {
			l = *layout
		} else {
			l = time.RFC3339
		}
		val, err := time.Parse(l, value)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(val))
		return nil
	},

	reflect.TypeOf(url.URL{}): func(field *reflect.Value, value string, _ *string) error {
		val, err := url.Parse(value)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(*val))
		return nil
	},

	reflect.TypeOf(&time.Location{}): func(field *reflect.Value, value string, _ *string) error {
		loc, err := time.LoadLocation(value)
		if err != nil {
			return err
		}

		field.Set(reflect.ValueOf(loc))
		return nil
	},
}

// readStructMetadata reads structure metadata (types, tags, etc.)
func readStructMetadata(cfgRoot interface{}) ([]structMeta, error) {
	type cfgNode struct {
		Val    interface{}
		Prefix string
	}

	cfgStack := []cfgNode{{cfgRoot, ""}}
	metas := make([]structMeta, 0)

	for i := 0; i < len(cfgStack); i++ {

		s := reflect.ValueOf(cfgStack[i].Val)
		sPrefix := cfgStack[i].Prefix

		// unwrap pointer
		if s.Kind() == reflect.Ptr {
			s = s.Elem()
		}

		// process only structures
		if s.Kind() != reflect.Struct {
			return nil, fmt.Errorf("wrong type %v", s.Kind())
		}
		typeInfo := s.Type()

		// read tags
		for idx := 0; idx < s.NumField(); idx++ {
			fType := typeInfo.Field(idx)

			var (
				defValue  *string
				layout    *string
				separator string
			)

			// process nested structure (except of supported ones)
			if fld := s.Field(idx); fld.Kind() == reflect.Struct {
				//skip unexported
				if !fld.CanInterface() {
					continue
				}
				// add structure to parsing stack
				if _, found := validStructs[fld.Type()]; !found {
					prefix, _ := fType.Tag.Lookup(TagEnvPrefix)
					cfgStack = append(cfgStack, cfgNode{fld.Addr().Interface(), sPrefix + prefix})
					continue
				}

				// process time.Time
				if l, ok := fType.Tag.Lookup(TagEnvLayout); ok {
					layout = &l
				}
			}

			// check is the field value can be changed
			if !s.Field(idx).CanSet() {
				continue
			}

			if def, ok := fType.Tag.Lookup(TagEnvDefault); ok {
				defValue = &def
			}

			if sep, ok := fType.Tag.Lookup(TagEnvSeparator); ok {
				separator = sep
			} else {
				separator = DefaultSeparator
			}

			_, upd := fType.Tag.Lookup(TagEnvUpd)

			_, required := fType.Tag.Lookup(TagEnvRequired)

			envList := make([]string, 0)

			if envs, ok := fType.Tag.Lookup(TagEnv); ok && len(envs) != 0 {
				envList = strings.Split(envs, DefaultSeparator)
				if sPrefix != "" {
					for i := range envList {
						envList[i] = sPrefix + envList[i]
					}
				}
			}

			metas = append(metas, structMeta{
				envList:     envList,
				fieldName:   s.Type().Field(idx).Name,
				fieldValue:  s.Field(idx),
				defValue:    defValue,
				layout:      layout,
				separator:   separator,
				description: fType.Tag.Get(TagEnvDescription),
				updatable:   upd,
				required:    required,
			})
		}

	}

	return metas, nil
}

// readEnvVars reads environment variables to the provided configuration structure
func readEnvVars(cfg interface{}, update bool) error {
	metaInfo, err := readStructMetadata(cfg)
	if err != nil {
		return err
	}

	if updater, ok := cfg.(Updater); ok {
		if err := updater.Update(); err != nil {
			return err
		}
	}

	for _, meta := range metaInfo {
		// update only updatable fields
		if update && !meta.updatable {
			continue
		}

		var rawValue *string

		for _, env := range meta.envList {
			if value, ok := os.LookupEnv(env); ok {
				rawValue = &value
				break
			}
		}

		if rawValue == nil && meta.required && meta.isFieldValueZero() {
			return fmt.Errorf(
				"field %q is required but the value is not provided",
				meta.fieldName,
			)
		}

		if rawValue == nil && meta.isFieldValueZero() {
			rawValue = meta.defValue
		}

		if rawValue == nil {
			continue
		}

		var envName string
		if len(meta.envList) > 0 {
			envName = meta.envList[0]
		}

		if err := parseValue(meta.fieldValue, *rawValue, meta.separator, meta.layout); err != nil {
			return fmt.Errorf("parsing field %v env %v: %v", meta.fieldName, envName, err)
		}
	}

	return nil
}

// parseValue parses value into the corresponding field.
// In case of maps and slices it uses provided separator to split raw value string
func parseValue(field reflect.Value, value, sep string, layout *string) error {
	// TODO: simplify recursion

	valueType := field.Type()

	// look for supported struct parser
	// parsing of struct must be done before checking the implementation `encoding.TextUnmarshaler`
	// standard struct types already have the implementation `encoding.TextUnmarshaler` (for example `time.Time`)
	if structParser, found := validStructs[valueType]; found {
		return structParser(&field, value, layout)
	}

	if field.CanInterface() {
		if ct, ok := field.Interface().(encoding.TextUnmarshaler); ok {
			return ct.UnmarshalText([]byte(value))
		} else if ctp, ok := field.Addr().Interface().(encoding.TextUnmarshaler); ok {
			return ctp.UnmarshalText([]byte(value))
		}

		if cs, ok := field.Interface().(Setter); ok {
			return cs.SetValue(value)
		} else if csp, ok := field.Addr().Interface().(Setter); ok {
			return csp.SetValue(value)
		}
	}

	switch valueType.Kind() {

	// parse string value
	case reflect.String:
		field.SetString(value)

	// parse boolean value
	case reflect.Bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(b)

	// parse integer
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		number, err := strconv.ParseInt(value, 0, valueType.Bits())
		if err != nil {
			return err
		}
		field.SetInt(number)

	case reflect.Int64:
		if valueType == reflect.TypeOf(time.Duration(0)) {
			// try to parse time
			d, err := time.ParseDuration(value)
			if err != nil {
				return err
			}
			field.SetInt(int64(d))
		} else {
			// parse regular integer
			number, err := strconv.ParseInt(value, 0, valueType.Bits())
			if err != nil {
				return err
			}
			field.SetInt(number)
		}

	// parse unsigned integer value
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		number, err := strconv.ParseUint(value, 0, valueType.Bits())
		if err != nil {
			return err
		}
		field.SetUint(number)

	// parse floating point value
	case reflect.Float32, reflect.Float64:
		number, err := strconv.ParseFloat(value, valueType.Bits())
		if err != nil {
			return err
		}
		field.SetFloat(number)

	// parse sliced value
	case reflect.Slice:
		sliceValue, err := parseSlice(valueType, value, sep, layout)
		if err != nil {
			return err
		}

		field.Set(*sliceValue)

	// parse mapped value
	case reflect.Map:
		mapValue, err := parseMap(valueType, value, sep, layout)
		if err != nil {
			return err
		}

		field.Set(*mapValue)

	default:
		return fmt.Errorf("unsupported type %s.%s", valueType.PkgPath(), valueType.Name())
	}

	return nil
}

// GetDescription returns a description of environment variables.
// You can provide a custom header text.
func GetDescription(cfg interface{}, headerText *string) (string, error) {
	meta, err := readStructMetadata(cfg)
	if err != nil {
		return "", err
	}

	var header string

	if headerText != nil {
		header = *headerText
	} else {
		header = "Environment variables:"
	}

	description := make([]string, 0)

	for _, m := range meta {
		if len(m.envList) == 0 {
			continue
		}

		for idx, env := range m.envList {

			elemDescription := fmt.Sprintf("\n  %s %s", env, m.fieldValue.Kind())
			if idx > 0 {
				elemDescription += fmt.Sprintf(" (alternative to %s)", m.envList[0])
			}
			elemDescription += fmt.Sprintf("\n    \t%s", m.description)
			if m.defValue != nil {
				elemDescription += fmt.Sprintf(" (default %q)", *m.defValue)
			}
			description = append(description, elemDescription)
		}
	}

	if len(description) == 0 {
		return "", nil
	}

	sort.Strings(description)

	return header + strings.Join(description, ""), nil
}

// Usage returns a configuration usage help.
// Other usage instructions can be wrapped in and executed before this usage function.
// The default output is STDERR.
func Usage(cfg interface{}, headerText *string, usageFuncs ...func()) func() {
	return FUsage(os.Stderr, cfg, headerText, usageFuncs...)
}

// FUsage prints configuration help into the custom output.
// Other usage instructions can be wrapped in and executed before this usage function
func FUsage(w io.Writer, cfg interface{}, headerText *string, usageFuncs ...func()) func() {
	return func() {
		for _, fn := range usageFuncs {
			fn()
		}

		_ = flag.Usage

		text, err := GetDescription(cfg, headerText)
		if err != nil {
			return
		}
		if len(usageFuncs) > 0 {
			fmt.Fprintln(w)
		}
		fmt.Fprintln(w, text)
	}
}

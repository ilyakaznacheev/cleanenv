package cleanenv

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
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
	// Name of the environment variable or a list of names
	TagEnv = "env"
	// Value parsing layout (for types like time.Time)
	TagEnvLayout = "env-layout"
	// Default value
	TagEnvDefault = "env-default"
	// Custom list and map separator
	TagEnvSeparator = "env-separator"
	// Environment variable description
	TagEnvDescription = "env-description"
	// Flag to mark a field as updatable
	TagEnvUpd = "env-upd"
	// Flag to mark a field as required
	TagEnvRequired = "env-required"
	// Flag to specify prefix for structure fields
	TagEnvPrefix = "env-prefix"
)

// Setter is an interface for a custom value setter.
//
// To implement a custom value setter you need to add a SetValue function to your type that will receive a string raw value:
//
// 	type MyField string
//
// 	func (f *MyField) SetValue(s string) error {
// 		if s == "" {
// 			return fmt.Errorf("field value can't be empty")
// 		}
// 		*f = MyField("my field is: " + s)
// 		return nil
// 	}
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
//	 type ConfigDatabase struct {
//	 	Port     string `yaml:"port" env:"PORT" env-default:"5432"`
//	 	Host     string `yaml:"host" env:"HOST" env-default:"localhost"`
//	 	Name     string `yaml:"name" env:"NAME" env-default:"postgres"`
//	 	User     string `yaml:"user" env:"USER" env-default:"user"`
//	 	Password string `yaml:"password" env:"PASSWORD"`
//	 }
//
//	 var cfg ConfigDatabase
//
//	 err := cleanenv.ReadConfig("config.yml", &cfg)
//	 if err != nil {
//	     ...
//	 }
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
		err = parseYAML(f, cfg)
	case ".json":
		err = parseJSON(f, cfg)
	case ".toml":
		err = parseTOML(f, cfg)
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

// parseYAML parses YAML from reader to data structure
func parseYAML(r io.Reader, str interface{}) error {
	return yaml.NewDecoder(r).Decode(str)
}

// parseJSON parses JSON from reader to data structure
func parseJSON(r io.Reader, str interface{}) error {
	return json.NewDecoder(r).Decode(str)
}

// parseTOML parses TOML from reader to data structure
func parseTOML(r io.Reader, str interface{}) error {
	_, err := toml.DecodeReader(r, str)
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
		os.Setenv(env, val)
	}
	return nil
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
	return isZero(sm.fieldValue)
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

			// process nested structure (except of time.Time)
			if fld := s.Field(idx); fld.Kind() == reflect.Struct {
				// add structure to parsing stack
				if fld.Type() != reflect.TypeOf(time.Time{}) {
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
			err := fmt.Errorf("field %q is required but the value is not provided",
				meta.fieldName)
			return err
		}

		if rawValue == nil && meta.isFieldValueZero() {
			rawValue = meta.defValue
		}

		if rawValue == nil {
			continue
		}

		if err := parseValue(meta.fieldValue, *rawValue, meta.separator, meta.layout); err != nil {
			return err
		}
	}

	return nil
}

// parseValue parses value into the corresponding field.
// In case of maps and slices it uses provided separator to split raw value string
func parseValue(field reflect.Value, value, sep string, layout *string) error {
	// TODO: simplify recursion

	if field.CanInterface() {
		if cs, ok := field.Interface().(Setter); ok {
			return cs.SetValue(value)
		} else if csp, ok := field.Addr().Interface().(Setter); ok {
			return csp.SetValue(value)
		}
	}

	valueType := field.Type()

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

	// parse integer (or time) value
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if field.Kind() == reflect.Int64 && valueType.PkgPath() == "time" && valueType.Name() == "Duration" {
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

	case reflect.Struct:
		// process time.Time only
		if valueType.PkgPath() == "time" && valueType.Name() == "Time" {

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
		}

	default:
		return fmt.Errorf("unsupported type %s.%s", valueType.PkgPath(), valueType.Name())
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

// GetDescription returns a description of environment variables.
// You can provide a custom header text.
func GetDescription(cfg interface{}, headerText *string) (string, error) {
	meta, err := readStructMetadata(cfg)
	if err != nil {
		return "", err
	}

	var header, description string

	if headerText != nil {
		header = *headerText
	} else {
		header = "Environment variables:"
	}

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
			description += elemDescription
		}
	}

	if description != "" {
		return header + description, nil
	}
	return "", nil
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

// isZero is a backport of reflect.Value.IsZero()
func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return math.Float64bits(v.Float()) == 0
	case reflect.Complex64, reflect.Complex128:
		c := v.Complex()
		return math.Float64bits(real(c)) == 0 && math.Float64bits(imag(c)) == 0
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if !isZero(v.Index(i)) {
				return false
			}
		}
		return true
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice, reflect.UnsafePointer:
		return v.IsNil()
	case reflect.String:
		return v.Len() == 0
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if !isZero(v.Field(i)) {
				return false
			}
		}
		return true
	default:
		// This should never happens, but will act as a safeguard for
		// later, as a default value doesn't makes sense here.
		panic(fmt.Sprintf("Value.IsZero: %v", v.Kind()))
	}
}

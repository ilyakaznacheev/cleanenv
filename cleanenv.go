package cleanenv

import (
	"flag"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
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
		// look for supported struct parser
		if structParser, found := validStructs[valueType]; found {
			return structParser(&field, value, layout)
		}

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

package cleanenv

import (
	"net/url"
	"reflect"
	"time"
)

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

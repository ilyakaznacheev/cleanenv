package cleanenv

import (
	"fmt"
)

type RequireError struct {
	FieldName string
	EnvName   string
}

func newRequireError(fieldName string, envName string) RequireError {
	return RequireError{
		FieldName: fieldName,
		EnvName:   envName,
	}
}

func (r RequireError) Error() string {
	return fmt.Sprintf(
		"field %q is required but the value is not provided",
		r.FieldName,
	)
}

type ParsingError struct {
	Err       error
	FieldName string
	EnvName   string
}

func newParsingError(fieldName string, envName string, err error) ParsingError {
	return ParsingError{
		FieldName: fieldName,
		EnvName:   envName,
		Err:       err,
	}
}

func (p ParsingError) Error() string {
	return fmt.Sprintf("parsing field %v env %v: %v", p.FieldName, p.EnvName, p.Err)
}

func (p ParsingError) Unwrap() error {
	return p.Err
}

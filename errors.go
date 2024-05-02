package cleanenv

import (
	"fmt"
	"strings"
)

type RequireError struct {
	FieldName string
	FieldPath []string
	EnvName   string
}

func newRequireError(fieldName string, fieldPath []string, envName string) RequireError {
	return RequireError{
		FieldName: fieldName,
		FieldPath: fieldPath,
		EnvName:   envName,
	}
}

func (r RequireError) Error() string {
	return fmt.Sprintf(
		"field %q is required but the value is not provided",
		strings.Join(append(r.FieldPath, r.FieldName), "."),
	)
}

type ParsingError struct {
	Err       error
	FieldName string
	FieldPath []string
	EnvName   string
}

func newParsingError(fieldName string, fieldPath []string, envName string, err error) ParsingError {
	return ParsingError{
		FieldName: fieldName,
		FieldPath: fieldPath,
		EnvName:   envName,
		Err:       err,
	}
}

func (p ParsingError) Error() string {
	return fmt.Sprintf(
		"parsing field %v env %v: %v",
		strings.Join(append(p.FieldPath, p.FieldName), "."),
		p.EnvName,
		p.Err,
	)
}

func (p ParsingError) Unwrap() error {
	return p.Err
}

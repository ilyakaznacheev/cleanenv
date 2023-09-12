package cleanenv

import (
	"os"
	"strconv"
	"text/tabwriter"
	"text/template"
)

const (
	DefaultTableFormat = `The following environment variables can be used for configuration:

KEY	TYPE	DEFAULT	REQUIRED	DESCRIPTION
{{range .}}{{usage_key .}}	{{usage_type .}}	{{usage_default .}}	{{usage_required .}}	{{usage_description .}}
{{end}}`
)

func PrintDescription(cfg interface{}) error {
	meta, err := readStructMetadata(cfg)
	if err != nil {
		return err
	}

	// Specify the default usage template functions
	functions := template.FuncMap{
		"usage_key":         func(v structMeta) string { return v.envList[0] },
		"usage_description": func(v structMeta) string { return v.description },
		"usage_type":        func(v structMeta) string { return v.fieldValue.Kind().String() },
		"usage_default":     func(v structMeta) string { return derefString(v.defValue) },
		"usage_required":    func(v structMeta) string { return formatBool(v.required) },
	}

	tmpl, err := template.New("cleanenv").Funcs(functions).Parse(DefaultTableFormat)
	if err != nil {
		return err
	}
	tabs := tabwriter.NewWriter(os.Stdout, 1, 0, 4, ' ', 0)
	err = tmpl.Execute(tabs, meta)
	if err != nil {
		return err
	}
	tabs.Flush()
	return nil
}

func derefString(s *string) string {
	if s != nil {
		return *s
	}

	return ""
}

func formatBool(b bool) string {
	if b {
		return strconv.FormatBool(b)
	}
	return ""
}

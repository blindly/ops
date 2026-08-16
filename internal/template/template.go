package template

import (
	"bytes"
	"fmt"
	"text/template"
)

func Render(s string, vars map[string]string) (string, error) {
	t, err := template.New("task").Option("missingkey=error").Parse(s)
	if err != nil {
		return "", fmt.Errorf("template parse: %w", err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("template execute: %w", err)
	}
	return buf.String(), nil
}

package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/adortb/adortb-dco/internal/model"
)

// renderHTML executes the Go text/template for banner ads.
func renderHTML(tmpl *model.Template, slots map[string]string) (string, error) {
	if tmpl.HTMLTemplate == "" {
		return "", nil
	}
	t, err := template.New("").Option("missingkey=zero").Parse(tmpl.HTMLTemplate)
	if err != nil {
		return "", fmt.Errorf("parse html template %d: %w", tmpl.ID, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, slots); err != nil {
		return "", fmt.Errorf("execute html template %d: %w", tmpl.ID, err)
	}
	return buf.String(), nil
}

// renderNative substitutes slot values into the native JSON template.
func renderNative(tmpl *model.Template, slots map[string]string) (map[string]string, error) {
	if len(tmpl.NativeTemplate) == 0 {
		return nil, nil
	}
	raw := string(tmpl.NativeTemplate)
	for slot, val := range slots {
		raw = strings.ReplaceAll(raw, "{{"+slot+"}}", val)
	}
	var out map[string]string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("unmarshal native template %d: %w", tmpl.ID, err)
	}
	return out, nil
}

// buildSlotValues merges asset values with dynamic data (dynamic_data wins).
func buildSlotValues(chosen map[string]*model.Asset, dynamicData map[string]string) map[string]string {
	vals := make(map[string]string, len(chosen)+len(dynamicData))
	for slot, asset := range chosen {
		vals[slot] = asset.Value
	}
	for k, v := range dynamicData {
		vals[k] = v
	}
	return vals
}

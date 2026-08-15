package view

import (
	"encoding/json"
	"fmt"
	"html/template"
	"reflect"
	"sort"
	"strings"
)

// dataGet reads a dotted path from maps and structs.
func dataGet(data any, path string) any {
	if data == nil || path == "" {
		return nil
	}
	cur := data
	for _, part := range strings.Split(path, ".") {
		if part == "" {
			return nil
		}
		switch m := cur.(type) {
		case map[string]any:
			var ok bool
			cur, ok = m[part]
			if !ok {
				return nil
			}
		case map[string]string:
			v, ok := m[part]
			if !ok {
				return nil
			}
			cur = v
		default:
			rv := reflect.ValueOf(cur)
			for rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface {
				if rv.IsNil() {
					return nil
				}
				rv = rv.Elem()
			}
			switch rv.Kind() {
			case reflect.Map:
				if rv.Type().Key().Kind() != reflect.String {
					return nil
				}
				val := rv.MapIndex(reflect.ValueOf(part))
				if !val.IsValid() {
					return nil
				}
				cur = val.Interface()
			case reflect.Struct:
				val := structFieldByName(rv, part)
				if !val.IsValid() {
					return nil
				}
				if val.CanInterface() {
					cur = val.Interface()
				} else {
					return nil
				}
			default:
				return nil
			}
		}
	}
	return cur
}

func structFieldByName(rv reflect.Value, name string) reflect.Value {
	if rv.Kind() != reflect.Struct {
		return reflect.Value{}
	}
	if f := rv.FieldByName(name); f.IsValid() {
		return f
	}
	t := rv.Type()
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if !sf.Anonymous {
			continue
		}
		fv := rv.Field(i)
		for fv.Kind() == reflect.Pointer {
			if fv.IsNil() {
				break
			}
			fv = fv.Elem()
		}
		if fv.Kind() != reflect.Struct {
			continue
		}
		if found := structFieldByName(fv, name); found.IsValid() {
			return found
		}
	}
	return reflect.Value{}
}

func safeStr(v any) template.HTML {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case template.HTML:
		return x
	case string:
		return template.HTML(x)
	default:
		return template.HTML(fmt.Sprint(x))
	}
}

func toJSON(v any) template.JS {
	b, err := json.Marshal(v)
	if err != nil {
		return template.JS("null")
	}
	return template.JS(b)
}

func classAttr(v any) template.HTMLAttr {
	classes := conditionalTokens(v, true)
	if len(classes) == 0 {
		return ""
	}
	return template.HTMLAttr(`class="` + template.HTMLEscapeString(strings.Join(classes, " ")) + `"`)
}

func styleAttr(v any) template.HTMLAttr {
	parts := conditionalTokens(v, false)
	if len(parts) == 0 {
		return ""
	}
	return template.HTMLAttr(`style="` + template.HTMLEscapeString(strings.Join(parts, "; ")) + `"`)
}

// conditionalTokens turns maps/slices into class or style tokens.
// For maps, truthy values keep the key; for slices, string items are kept.
func conditionalTokens(v any, sortKeys bool) []string {
	if v == nil {
		return nil
	}
	var out []string
	switch m := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		if sortKeys {
			sort.Strings(keys)
		}
		for _, k := range keys {
			if isTruthy(m[k]) {
				out = append(out, k)
			}
		}
	case map[string]bool:
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		if sortKeys {
			sort.Strings(keys)
		}
		for _, k := range keys {
			if m[k] {
				out = append(out, k)
			}
		}
	case []string:
		for _, item := range m {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
	case []any:
		for _, item := range m {
			s := strings.TrimSpace(fmt.Sprint(item))
			if s != "" && s != "<nil>" {
				out = append(out, s)
			}
		}
	case string:
		s := strings.TrimSpace(m)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func attrBool(cond any, attr string) template.HTMLAttr {
	if !isTruthy(cond) {
		return ""
	}
	return template.HTMLAttr(attr)
}

func isTruthy(v any) bool {
	if v == nil {
		return false
	}
	switch x := v.(type) {
	case bool:
		return x
	case string:
		return strings.TrimSpace(x) != ""
	case int:
		return x != 0
	case int64:
		return x != 0
	case float64:
		return x != 0
	}
	return !isEmptyValue(v)
}

func mergeDict(base map[string]any, extra map[string]any) map[string]any {
	out := make(map[string]any, len(base)+len(extra))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

// mergeDefaults copies base and fills only missing keys from defaults.
func mergeDefaults(base map[string]any, defaults map[string]any) map[string]any {
	out := make(map[string]any, len(base)+len(defaults))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range defaults {
		if _, ok := out[k]; !ok {
			out[k] = v
		}
	}
	return out
}

func dict(values ...any) map[string]any {
	out := map[string]any{}
	for i := 0; i+1 < len(values); i += 2 {
		key := fmt.Sprint(values[i])
		out[key] = values[i+1]
	}
	return out
}

// parseBladeMapExpr parses ['k' => $var, 'a' => 'lit'] into template dict call args source.
func parseBladeMapExpr(expr string) string {
	expr = strings.TrimSpace(expr)
	expr = strings.TrimPrefix(expr, "[")
	expr = strings.TrimSuffix(expr, "]")
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return `dict`
	}
	parts := splitBladeMapEntries(expr)
	args := make([]string, 0, len(parts)*2)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=>", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])
		key = strings.Trim(key, `'"`)
		args = append(args, fmt.Sprintf("%q", key))
		args = append(args, bladeValueExpr(val))
	}
	if len(args) == 0 {
		return `dict`
	}
	return `dict ` + strings.Join(args, " ")
}

func splitBladeMapEntries(expr string) []string {
	var parts []string
	var b strings.Builder
	depth := 0
	inQuote := byte(0)
	for i := 0; i < len(expr); i++ {
		ch := expr[i]
		if inQuote != 0 {
			b.WriteByte(ch)
			if ch == inQuote && (i == 0 || expr[i-1] != '\\') {
				inQuote = 0
			}
			continue
		}
		switch ch {
		case '\'', '"':
			inQuote = ch
			b.WriteByte(ch)
		case '[', '(':
			depth++
			b.WriteByte(ch)
		case ']', ')':
			if depth > 0 {
				depth--
			}
			b.WriteByte(ch)
		case ',':
			if depth == 0 {
				parts = append(parts, b.String())
				b.Reset()
				continue
			}
			b.WriteByte(ch)
		default:
			b.WriteByte(ch)
		}
	}
	if s := strings.TrimSpace(b.String()); s != "" {
		parts = append(parts, s)
	}
	return parts
}

func bladeValueExpr(val string) string {
	val = strings.TrimSpace(val)
	if strings.HasPrefix(val, "$") {
		path := strings.TrimPrefix(val, "$")
		return fmt.Sprintf(`(dataGet . %q)`, path)
	}
	if (strings.HasPrefix(val, "'") && strings.HasSuffix(val, "'")) ||
		(strings.HasPrefix(val, `"`) && strings.HasSuffix(val, `"`)) {
		lit := strings.Trim(val, `'"`)
		return fmt.Sprintf("%q", lit)
	}
	if val == "true" || val == "false" {
		return val
	}
	return fmt.Sprintf("%q", val)
}

func toFloat64(v any) (float64, bool) {
	if v == nil {
		return 0, false
	}
	switch x := v.(type) {
	case int:
		return float64(x), true
	case int8:
		return float64(x), true
	case int16:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case uint:
		return float64(x), true
	case uint8:
		return float64(x), true
	case uint16:
		return float64(x), true
	case uint32:
		return float64(x), true
	case uint64:
		return float64(x), true
	case float32:
		return float64(x), true
	case float64:
		return x, true
	case string:
		var f float64
		if _, err := fmt.Sscanf(strings.TrimSpace(x), "%f", &f); err == nil {
			return f, true
		}
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(rv.Uint()), true
	case reflect.Float32, reflect.Float64:
		return rv.Float(), true
	}
	return 0, false
}

func cmpGt(a, b any) bool {
	if af, aok := toFloat64(a); aok {
		if bf, bok := toFloat64(b); bok {
			return af > bf
		}
	}
	return fmt.Sprint(a) > fmt.Sprint(b)
}

func cmpGe(a, b any) bool {
	if af, aok := toFloat64(a); aok {
		if bf, bok := toFloat64(b); bok {
			return af >= bf
		}
	}
	return fmt.Sprint(a) >= fmt.Sprint(b)
}

func cmpLt(a, b any) bool {
	if af, aok := toFloat64(a); aok {
		if bf, bok := toFloat64(b); bok {
			return af < bf
		}
	}
	return fmt.Sprint(a) < fmt.Sprint(b)
}

func cmpLe(a, b any) bool {
	if af, aok := toFloat64(a); aok {
		if bf, bok := toFloat64(b); bok {
			return af <= bf
		}
	}
	return fmt.Sprint(a) <= fmt.Sprint(b)
}

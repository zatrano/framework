package templatehelpers

import (
	"html/template"
	"net/url"
	"reflect"
	"strings"
	"time"
)

// PathOnly normalizes path (dil kaldırıldı, her zaman aynı path).
func PathOnly(_, path string) string {
	return "/" + strings.TrimPrefix(strings.TrimSpace(path), "/")
}

func TemplateHelpers() template.FuncMap {
	fm := template.FuncMap{
		"CurrentYear": func() int { return time.Now().Year() },
		"Add":         func(a, b int) int { return a + b },
		"Subtract":    func(a, b int) int { return a - b },
		"Mul":         func(a, b int) int { return a * b },
		"Max": func(a, b int) int {
			if a > b {
				return a
			}
			return b
		},
		"max": func(a, b int) int {
			if a > b {
				return a
			}
			return b
		},

		"Min": func(a, b int) int {
			if a < b {
				return a
			}
			return b
		},
		"min": func(a, b int) int {
			if a < b {
				return a
			}
			return b
		},

		"Iterate": func(start, end int) []int {
			count := end - start + 1
			if count <= 0 {
				return []int{}
			}
			items := make([]int, count)
			for i := 0; i < count; i++ {
				items[i] = start + i
			}
			return items
		},
		"urlquery": func(s string) string { return url.QueryEscape(s) },
		"dict": func(values ...interface{}) map[string]interface{} {
			dict := make(map[string]interface{})
			if len(values)%2 != 0 {
				return dict
			}
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					continue
				}
				dict[key] = values[i+1]
			}
			return dict
		},

		"FormatTime": func(t time.Time, layout string) string {
			if t.IsZero() {
				return ""
			}
			return t.Format(layout)
		},

		"FormatDate": func(t time.Time) string {
			if t.IsZero() {
				return ""
			}
			return t.Format("02.01.2006")
		},

		"FormatDateTime": func(t time.Time) string {
			if t.IsZero() {
				return ""
			}
			return t.Format("02.01.2006 15:04")
		},
		"FormatDateTimeTR": func(t time.Time) string {
			if t.IsZero() {
				return ""
			}
			loc, err := time.LoadLocation("Europe/Istanbul")
			if err != nil {
				return t.Local().Format("02.01.2006 15:04")
			}
			return t.In(loc).Format("02.01.2006 15:04")
		},

		"hasPrefix": func(s, prefix string) bool {
			return len(s) >= len(prefix) && s[:len(prefix)] == prefix
		},

		"LangURL": func(lang, p string) string { return PathOnly(lang, p) },

		"EqUintPtr": func(a uint, b *uint) bool {
			if b == nil {
				return false
			}
			return a == *b
		},
		"upper": func(s string) string { return strings.ToUpper(s) },
		"SafeGet": func(obj interface{}, key string) interface{} {
			if obj == nil {
				return nil
			}

			v := reflect.ValueOf(obj)
			for v.Kind() == reflect.Ptr {
				if v.IsNil() {
					return nil
				}
				v = v.Elem()
			}

			if v.Kind() == reflect.Struct {
				f := v.FieldByName(key)
				if f.IsValid() {
					return f.Interface()
				}
			}

			if v.Kind() == reflect.Map {
				kv := reflect.ValueOf(key)
				val := v.MapIndex(kv)
				if val.IsValid() {
					return val.Interface()
				}
			}

			return nil
		},
		// ParamsToQueryMap, Params'ı (struct veya map) sayfalama linklerinde kullanılacak
		// bir map'e çevirir. Struct alanları map key'leri olarak kullanılır; Page, PerPage,
		// SortBy, OrderBy hariç tutulur (bunlar partial'da ayrı eklenir).
		"ParamsToQueryMap": func(in interface{}) map[string]interface{} {
			if in == nil {
				return map[string]interface{}{}
			}
			v := reflect.ValueOf(in)
			for v.Kind() == reflect.Ptr {
				if v.IsNil() {
					return map[string]interface{}{}
				}
				v = v.Elem()
			}
			skip := map[string]bool{"Page": true, "PerPage": true, "SortBy": true, "OrderBy": true}
			out := make(map[string]interface{})
			if v.Kind() == reflect.Map {
				iter := v.MapRange()
				for iter.Next() {
					k := iter.Key()
					if k.Kind() != reflect.String {
						continue
					}
					key := k.String()
					if skip[key] {
						continue
					}
					val := iter.Value()
					if val.IsValid() && !val.IsZero() {
						out[key] = val.Interface()
					}
				}
				return out
			}
			if v.Kind() == reflect.Struct {
				t := v.Type()
				for i := 0; i < v.NumField(); i++ {
					if t.Field(i).IsExported() {
						key := t.Field(i).Name
						if skip[key] {
							continue
						}
						val := v.Field(i)
						if val.IsValid() && !val.IsZero() {
							out[key] = val.Interface()
						}
					}
				}
				return out
			}
			return map[string]interface{}{}
		},
		"SafeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"SafeURL": func(s string) template.URL {
			return template.URL(s)
		},
	}
	return fm
}

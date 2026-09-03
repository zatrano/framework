package browser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/zatrano/framework/kernel"
	testkit "github.com/zatrano/framework/packages/testing"
)

// Browser is an HTTP feature-test helper wrapping TestCase (not a real browser/DOM engine).
type Browser struct {
	tc      *testkit.TestCase
	last    *testkit.TestResponse
	path    string
	form    map[string]string
	headers map[string]string
}

// New creates a browser session around an application.
func New(app *kernel.Application) (*Browser, error) {
	tc, err := testkit.New(app)
	if err != nil {
		return nil, err
	}
	return &Browser{
		tc:      tc,
		form:    map[string]string{},
		headers: map[string]string{},
	}, nil
}

// Visit performs a GET request.
func (b *Browser) Visit(path string) *Browser {
	b.path = path
	b.last = b.tc.Get(path)
	return b
}

// AssertOK asserts the last response was 200.
func (b *Browser) AssertOK() *Browser {
	b.ensure()
	b.last.AssertOK()
	return b
}

// AssertStatus asserts the last response status.
func (b *Browser) AssertStatus(status int) *Browser {
	b.ensure()
	b.last.AssertStatus(status)
	return b
}

// AssertSee asserts the body contains text.
func (b *Browser) AssertSee(text string) *Browser {
	b.ensure()
	if !strings.Contains(b.last.String(), text) {
		panic(fmt.Sprintf("browser: expected to see %q in body", text))
	}
	return b
}

// AssertDontSee asserts the body does not contain text.
func (b *Browser) AssertDontSee(text string) *Browser {
	b.ensure()
	if strings.Contains(b.last.String(), text) {
		panic(fmt.Sprintf("browser: did not expect to see %q", text))
	}
	return b
}

// AssertPathIs asserts the current path.
func (b *Browser) AssertPathIs(path string) *Browser {
	if b.path != path {
		panic(fmt.Sprintf("browser: expected path %q, got %q", path, b.path))
	}
	return b
}

// Type queues a form field value (overrides HTML defaults on Press).
func (b *Browser) Type(name, value string) *Browser {
	b.form[name] = value
	return b
}

// Press submits a form via HTTP.
// It finds the form that encloses a matching submit button (value/name/text),
// otherwise the first <form>. Typed fields override inputs found in HTML.
// Optional action overrides the form action attribute.
func (b *Browser) Press(button string, action ...string) *Browser {
	b.ensure()
	body := b.last.String()
	formHTML, ok := findFormForButton(body, button)
	if !ok {
		formHTML, ok = firstForm(body)
	}
	fields := map[string]string{}
	method := "POST"
	target := b.path
	if ok {
		method = formMethod(formHTML)
		if act := formAction(formHTML); act != "" {
			target = act
		}
		for k, v := range formFields(formHTML) {
			fields[k] = v
		}
	}
	for k, v := range b.form {
		fields[k] = v
	}
	if len(action) > 0 && action[0] != "" {
		target = action[0]
	}
	b.path = target
	if strings.EqualFold(method, "GET") {
		b.last = b.tc.Get(target)
	} else {
		b.last = b.tc.Post(target, fields)
	}
	b.form = map[string]string{}
	return b
}

// ClickLink follows the first anchor whose text matches.
func (b *Browser) ClickLink(text string) *Browser {
	b.ensure()
	re := regexp.MustCompile(`(?is)<a[^>]+href=["']([^"']+)["'][^>]*>\s*` + regexp.QuoteMeta(text) + `\s*</a>`)
	m := re.FindStringSubmatch(b.last.String())
	if m == nil {
		re2 := regexp.MustCompile(`(?is)href=["']([^"']+)["'][^>]*>[^<]*` + regexp.QuoteMeta(text))
		m = re2.FindStringSubmatch(b.last.String())
	}
	if m == nil {
		panic(fmt.Sprintf("browser: link %q not found", text))
	}
	return b.Visit(m[1])
}

// Status returns the last status code.
func (b *Browser) Status() int {
	b.ensure()
	return b.last.StatusCode
}

// Body returns the last response body.
func (b *Browser) Body() string {
	b.ensure()
	return b.last.String()
}

// Response returns the underlying test response.
func (b *Browser) Response() *testkit.TestResponse {
	return b.last
}

func (b *Browser) ensure() {
	if b.last == nil {
		panic("browser: no response yet; call Visit first")
	}
}

var formTagRe = regexp.MustCompile(`(?is)<form\b([^>]*)>(.*?)</form>`)

func firstForm(html string) (string, bool) {
	m := formTagRe.FindStringSubmatch(html)
	if m == nil {
		return "", false
	}
	return m[0], true
}

func findFormForButton(html, button string) (string, bool) {
	button = strings.TrimSpace(button)
	matches := formTagRe.FindAllStringSubmatch(html, -1)
	for _, m := range matches {
		inner := m[2]
		if buttonMatches(inner, button) {
			return m[0], true
		}
	}
	return "", false
}

func buttonMatches(formInner, button string) bool {
	if button == "" {
		return false
	}
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?is)<button\b[^>]*>\s*` + regexp.QuoteMeta(button) + `\s*</button>`),
		regexp.MustCompile(`(?is)<button\b[^>]*\bname=["']` + regexp.QuoteMeta(button) + `["'][^>]*>`),
		regexp.MustCompile(`(?is)<button\b[^>]*\bvalue=["']` + regexp.QuoteMeta(button) + `["'][^>]*>`),
		regexp.MustCompile(`(?is)<input\b[^>]*type=["']submit["'][^>]*\bvalue=["']` + regexp.QuoteMeta(button) + `["'][^>]*>`),
		regexp.MustCompile(`(?is)<input\b[^>]*\bvalue=["']` + regexp.QuoteMeta(button) + `["'][^>]*type=["']submit["'][^>]*>`),
		regexp.MustCompile(`(?is)<input\b[^>]*type=["']submit["'][^>]*\bname=["']` + regexp.QuoteMeta(button) + `["'][^>]*>`),
	}
	for _, re := range patterns {
		if re.MatchString(formInner) {
			return true
		}
	}
	return false
}

func formMethod(formHTML string) string {
	re := regexp.MustCompile(`(?is)<form\b[^>]*\bmethod=["']([^"']+)["']`)
	if m := re.FindStringSubmatch(formHTML); m != nil {
		return strings.ToUpper(strings.TrimSpace(m[1]))
	}
	return "POST"
}

func formAction(formHTML string) string {
	re := regexp.MustCompile(`(?is)<form\b[^>]*\baction=["']([^"']+)["']`)
	if m := re.FindStringSubmatch(formHTML); m != nil {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func formFields(formHTML string) map[string]string {
	fields := map[string]string{}
	inputRe := regexp.MustCompile(`(?is)<input\b([^>]*)>`)
	for _, m := range inputRe.FindAllStringSubmatch(formHTML, -1) {
		attrs := m[1]
		typ := strings.ToLower(attrValue(attrs, "type"))
		if typ == "submit" || typ == "button" || typ == "image" || typ == "reset" || typ == "file" {
			continue
		}
		name := attrValue(attrs, "name")
		if name == "" {
			continue
		}
		if typ == "checkbox" || typ == "radio" {
			if !regexp.MustCompile(`(?i)\bchecked\b`).MatchString(attrs) {
				continue
			}
		}
		fields[name] = attrValue(attrs, "value")
	}
	textareaRe := regexp.MustCompile(`(?is)<textarea\b([^>]*)>(.*?)</textarea>`)
	for _, m := range textareaRe.FindAllStringSubmatch(formHTML, -1) {
		name := attrValue(m[1], "name")
		if name == "" {
			continue
		}
		fields[name] = m[2]
	}
	selectRe := regexp.MustCompile(`(?is)<select\b([^>]*)>(.*?)</select>`)
	optionRe := regexp.MustCompile(`(?is)<option\b([^>]*)>(.*?)</option>`)
	for _, m := range selectRe.FindAllStringSubmatch(formHTML, -1) {
		name := attrValue(m[1], "name")
		if name == "" {
			continue
		}
		value := ""
		for _, opt := range optionRe.FindAllStringSubmatch(m[2], -1) {
			if regexp.MustCompile(`(?i)\bselected\b`).MatchString(opt[1]) || value == "" {
				if v := attrValue(opt[1], "value"); v != "" {
					value = v
				} else {
					value = strings.TrimSpace(opt[2])
				}
				if regexp.MustCompile(`(?i)\bselected\b`).MatchString(opt[1]) {
					break
				}
			}
		}
		fields[name] = value
	}
	return fields
}

func attrValue(attrs, name string) string {
	re := regexp.MustCompile(`(?is)\b` + regexp.QuoteMeta(name) + `=["']([^"']*)["']`)
	if m := re.FindStringSubmatch(attrs); m != nil {
		return m[1]
	}
	return ""
}

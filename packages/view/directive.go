package view

import (
	"regexp"
	"strings"
)

var (
	reProps         = regexp.MustCompile(`(?i)@props\s*\(\s*(\[[^\]]*\])\s*\)`)
	reCan           = regexp.MustCompile(`(?i)@can\s*\(\s*['"]([^'"]+)['"]\s*(?:,\s*\$([a-zA-Z0-9_.]+)\s*)?\)`)
	reCannot        = regexp.MustCompile(`(?i)@cannot\s*\(\s*['"]([^'"]+)['"]\s*(?:,\s*\$([a-zA-Z0-9_.]+)\s*)?\)`)
	reEnv           = regexp.MustCompile(`(?i)@env\s*\(\s*['"]([^'"]+)['"]\s*\)`)
	rePhp           = regexp.MustCompile(`(?is)@php\s*(.*?)@endphp`)
	reParent        = regexp.MustCompile(`(?i)@parent\b`)
	reEndCan        = regexp.MustCompile(`(?i)@endcan\b`)
	reEndCannot     = regexp.MustCompile(`(?i)@endcannot\b`)
	reEndEnv        = regexp.MustCompile(`(?i)@endenv\b`)
	reEndProduction = regexp.MustCompile(`(?i)@endproduction\b`)
)

// Directive registers a custom Blade-like directive replacer on the engine.
func (e *Engine) Directive(name string, replacer func(args string) string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.directives == nil {
		e.directives = make(map[string]func(string) string)
	}
	e.directives[name] = replacer
}

// SetEnvironment sets the runtime environment name (default "local").
func (e *Engine) SetEnvironment(env string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.environment = env
}

func (e *Engine) environmentName() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.environment == "" {
		return "local"
	}
	return e.environment
}

func (e *Engine) applyCustomDirectives(input string) string {
	e.mu.RLock()
	directives := make(map[string]func(string) string, len(e.directives))
	for name, fn := range e.directives {
		directives[name] = fn
	}
	e.mu.RUnlock()

	for name, replacer := range directives {
		if replacer == nil {
			continue
		}
		reWithArgs := mustCompile(`(?i)@` + regexp.QuoteMeta(name) + `\s*\(([^)]*)\)`)
		reBare := mustCompile(`(?i)@` + regexp.QuoteMeta(name) + `\b`)
		input = reWithArgs.ReplaceAllStringFunc(input, func(m string) string {
			submatch := reWithArgs.FindStringSubmatch(m)
			if len(submatch) != 2 {
				return m
			}
			return replacer(strings.TrimSpace(submatch[1]))
		})
		input = reBare.ReplaceAllStringFunc(input, func(m string) string {
			return replacer("")
		})
	}
	return input
}

func compileCanDirectives(input string) string {
	out := reCan.ReplaceAllStringFunc(input, func(m string) string {
		match := reCan.FindStringSubmatch(m)
		if len(match) < 2 {
			return m
		}
		if match[2] != "" {
			return "{{ if can . `" + match[1] + "` (dataGet . `" + match[2] + "`) }}"
		}
		return "{{ if can . `" + match[1] + "` }}"
	})
	out = reCannot.ReplaceAllStringFunc(out, func(m string) string {
		match := reCannot.FindStringSubmatch(m)
		if len(match) < 2 {
			return m
		}
		if match[2] != "" {
			return "{{ if not (can . `" + match[1] + "` (dataGet . `" + match[2] + "`)) }}"
		}
		return "{{ if not (can . `" + match[1] + "`) }}"
	})
	out = reEndCan.ReplaceAllString(out, "{{ end }}")
	out = reEndCannot.ReplaceAllString(out, "{{ end }}")
	return out
}

func compileEnvDirectives(input string) string {
	out := reEnv.ReplaceAllStringFunc(input, func(m string) string {
		match := reEnv.FindStringSubmatch(m)
		if len(match) != 2 {
			return m
		}
		return "{{ if env `" + match[1] + "` }}"
	})
	out = strings.ReplaceAll(out, "@production", `{{ if production }}`)
	out = reEndEnv.ReplaceAllString(out, "{{ end }}")
	out = reEndProduction.ReplaceAllString(out, "{{ end }}")
	return out
}

func compilePhpDirectives(input string) string {
	return rePhp.ReplaceAllString(input, "<!-- @php (unsupported) -->")
}

// expandProps wraps component templates with default prop merges from @props([...]).
func expandProps(content string) string {
	match := reProps.FindStringSubmatch(content)
	if len(match) != 2 {
		return content
	}
	dictCall := parseBladeMapExpr(match[1])
	body := strings.TrimSpace(reProps.ReplaceAllString(content, ""))
	return `{{ with mergeDefaults . (` + dictCall + `) }}` + body + `{{ end }}`
}

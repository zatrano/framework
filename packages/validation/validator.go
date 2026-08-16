package validation

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/zatrano/framework/packages/support/str"
	"github.com/zatrano/framework/packages/support/uuid"
)

// Errors holds validation error messages keyed by field.
type Errors map[string][]string

// First returns the first error for a field.
func (e Errors) First(field string) string {
	if messages, ok := e[field]; ok && len(messages) > 0 {
		return messages[0]
	}
	return ""
}

// Has reports whether a field has errors.
func (e Errors) Has(field string) bool {
	return len(e[field]) > 0
}

// All returns all errors.
func (e Errors) All() map[string][]string {
	return e
}

// Message returns a flat map of first errors.
func (e Errors) Message() map[string]string {
	out := make(map[string]string, len(e))
	for field := range e {
		out[field] = e.First(field)
	}
	return out
}

// RuleFunc validates a single field value.
type RuleFunc func(v *Validator, field, value, param string) bool

// PresenceChecker looks up whether a value exists in storage (unique/exists).
type PresenceChecker func(table, column, value string) (exists bool, err error)

var defaultPresenceChecker PresenceChecker

// SetDefaultPresenceChecker configures unique/exists lookups for Make/ValidateForm when a validator has none.
func SetDefaultPresenceChecker(fn PresenceChecker) {
	defaultPresenceChecker = fn
}

var customRules = map[string]RuleFunc{}

// Extend registers a custom validation rule.
func Extend(name string, fn RuleFunc) {
	customRules[name] = fn
}

// Validator validates input data against rules.
type Validator struct {
	data             map[string]string
	rules            map[string]string
	errors           Errors
	customMessages   map[string]string
	customAttributes map[string]string
	presenceChecker  PresenceChecker
	excluded         map[string]bool
}

// Make creates a validator.
func Make(data map[string]string, rules map[string]string) *Validator {
	return &Validator{
		data:   data,
		rules:  rules,
		errors: make(Errors),
	}
}

// SetMessages sets custom validation messages.
func (v *Validator) SetMessages(messages map[string]string) *Validator {
	v.customMessages = messages
	return v
}

// SetAttributes sets human-friendly attribute names for messages.
func (v *Validator) SetAttributes(attributes map[string]string) *Validator {
	v.customAttributes = attributes
	return v
}

// SetPresenceChecker configures unique/exists lookups.
func (v *Validator) SetPresenceChecker(fn PresenceChecker) *Validator {
	v.presenceChecker = fn
	return v
}

// Fails runs validation and reports whether it failed.
func (v *Validator) Fails() bool {
	v.validate()
	return len(v.errors) > 0
}

// Passes reports whether validation succeeded.
func (v *Validator) Passes() bool {
	return !v.Fails()
}

// Errors returns validation errors.
func (v *Validator) Errors() Errors {
	return v.errors
}

// Validated returns data for fields that have rules when validation passes.
// Fields marked with exclude_* rules are omitted.
// On failure it returns ValidationException with field errors.
func (v *Validator) Validated() (map[string]string, error) {
	if v.Fails() {
		return nil, ValidationException{Errors: v.Errors()}
	}
	out := make(map[string]string, len(v.rules))
	for field := range v.rules {
		if v.excluded != nil && v.excluded[field] {
			continue
		}
		out[field] = v.data[field]
	}
	return out, nil
}

func (v *Validator) validate() {
	v.errors = make(Errors)
	v.excluded = make(map[string]bool)
	for field, ruleString := range v.rules {
		rules := strings.Split(ruleString, "|")
		if v.shouldExclude(rules) {
			v.excluded[field] = true
			continue
		}
		if hasNamedRule(rules, "sometimes") {
			if _, ok := v.data[field]; !ok {
				continue
			}
		}
		value := v.data[field]
		nullable := hasNamedRule(rules, "nullable")
		if nullable && strings.TrimSpace(value) == "" {
			continue
		}
		for _, rule := range rules {
			rule = strings.TrimSpace(rule)
			if rule == "" || rule == "nullable" || rule == "bail" || rule == "sometimes" {
				continue
			}
			name, param := splitRule(rule)
			if name == "exclude_if" || name == "exclude_unless" ||
				name == "exclude_with" || name == "exclude_with_all" ||
				name == "exclude_without" || name == "exclude_without_all" {
				continue
			}
			if !v.check(name, field, value, param) {
				v.errors[field] = append(v.errors[field], v.messageFor(name, field, param))
				if hasBail(rules) {
					break
				}
			}
		}
	}
}

func (v *Validator) shouldExclude(rules []string) bool {
	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		name, param := splitRule(rule)
		switch name {
		case "exclude_if":
			other, expected, ok := splitFieldValue(param)
			if ok && v.data[other] == expected {
				return true
			}
		case "exclude_unless":
			other, expected, ok := splitFieldValue(param)
			if !ok {
				return false
			}
			if v.data[other] != expected {
				return true
			}
		case "exclude_with":
			if v.anyFilled(param) {
				return true
			}
		case "exclude_with_all":
			if v.allFilled(param) {
				return true
			}
		case "exclude_without":
			if v.anyMissing(param) {
				return true
			}
		case "exclude_without_all":
			if v.allMissing(param) {
				return true
			}
		}
	}
	return false
}

func (v *Validator) anyFilled(param string) bool {
	for _, other := range strings.Split(param, ",") {
		other = strings.TrimSpace(other)
		if other == "" {
			continue
		}
		if strings.TrimSpace(v.data[other]) != "" {
			return true
		}
	}
	return false
}

func (v *Validator) allFilled(param string) bool {
	found := false
	for _, other := range strings.Split(param, ",") {
		other = strings.TrimSpace(other)
		if other == "" {
			continue
		}
		found = true
		if strings.TrimSpace(v.data[other]) == "" {
			return false
		}
	}
	return found
}

func (v *Validator) anyMissing(param string) bool {
	for _, other := range strings.Split(param, ",") {
		other = strings.TrimSpace(other)
		if other == "" {
			continue
		}
		if strings.TrimSpace(v.data[other]) == "" {
			return true
		}
	}
	return false
}

func (v *Validator) allMissing(param string) bool {
	found := false
	for _, other := range strings.Split(param, ",") {
		other = strings.TrimSpace(other)
		if other == "" {
			continue
		}
		found = true
		if strings.TrimSpace(v.data[other]) != "" {
			return false
		}
	}
	return found
}

func hasNamedRule(rules []string, name string) bool {
	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		n, _ := splitRule(rule)
		if n == name {
			return true
		}
	}
	return false
}

func (v *Validator) check(rule, field, value, param string) bool {
	if fn, ok := customRules[rule]; ok {
		return fn(v, field, value, param)
	}

	switch rule {
	case "required":
		return strings.TrimSpace(value) != ""
	case "filled":
		// must be present and non-empty when the key exists
		if _, ok := v.data[field]; !ok {
			return true
		}
		return strings.TrimSpace(value) != ""
	case "present":
		_, ok := v.data[field]
		return ok
	case "missing":
		_, ok := v.data[field]
		return !ok
	case "present_if":
		other, expected, ok := splitFieldValue(param)
		if !ok {
			return false
		}
		if v.data[other] != expected {
			return true
		}
		_, exists := v.data[field]
		return exists
	case "present_unless":
		other, expected, ok := splitFieldValue(param)
		if !ok {
			return false
		}
		if v.data[other] == expected {
			return true
		}
		_, exists := v.data[field]
		return exists
	case "missing_if":
		other, expected, ok := splitFieldValue(param)
		if !ok {
			return false
		}
		if v.data[other] != expected {
			return true
		}
		_, exists := v.data[field]
		return !exists
	case "missing_unless":
		other, expected, ok := splitFieldValue(param)
		if !ok {
			return false
		}
		if v.data[other] == expected {
			return true
		}
		_, exists := v.data[field]
		return !exists
	case "present_with":
		if !v.anyFilled(param) {
			return true
		}
		_, exists := v.data[field]
		return exists
	case "present_with_all":
		if !v.allFilled(param) {
			return true
		}
		_, exists := v.data[field]
		return exists
	case "missing_with":
		if !v.anyFilled(param) {
			return true
		}
		_, exists := v.data[field]
		return !exists
	case "missing_with_all":
		if !v.allFilled(param) {
			return true
		}
		_, exists := v.data[field]
		return !exists
	case "present_without":
		if !v.anyMissing(param) {
			return true
		}
		_, exists := v.data[field]
		return exists
	case "present_without_all":
		if !v.allMissing(param) {
			return true
		}
		_, exists := v.data[field]
		return exists
	case "missing_without":
		if !v.anyMissing(param) {
			return true
		}
		_, exists := v.data[field]
		return !exists
	case "missing_without_all":
		if !v.allMissing(param) {
			return true
		}
		_, exists := v.data[field]
		return !exists
	case "required_if":
		other, expected, ok := splitFieldValue(param)
		if !ok {
			return false
		}
		if v.data[other] != expected {
			return true
		}
		return strings.TrimSpace(value) != ""
	case "required_if_accepted":
		other := strings.TrimSpace(param)
		if other == "" || !isAccepted(v.data[other]) {
			return true
		}
		return strings.TrimSpace(value) != ""
	case "required_if_declined":
		other := strings.TrimSpace(param)
		if other == "" || !isDeclined(v.data[other]) {
			return true
		}
		return strings.TrimSpace(value) != ""
	case "required_unless":
		other, expected, ok := splitFieldValue(param)
		if !ok {
			return false
		}
		if v.data[other] == expected {
			return true
		}
		return strings.TrimSpace(value) != ""
	case "required_with":
		others := strings.Split(param, ",")
		needed := false
		for _, other := range others {
			other = strings.TrimSpace(other)
			if other == "" {
				continue
			}
			if strings.TrimSpace(v.data[other]) != "" {
				needed = true
				break
			}
		}
		if !needed {
			return true
		}
		return strings.TrimSpace(value) != ""
	case "required_with_all":
		others := strings.Split(param, ",")
		for _, other := range others {
			other = strings.TrimSpace(other)
			if other == "" {
				continue
			}
			if strings.TrimSpace(v.data[other]) == "" {
				return true
			}
		}
		return strings.TrimSpace(value) != ""
	case "required_without":
		others := strings.Split(param, ",")
		needed := false
		for _, other := range others {
			other = strings.TrimSpace(other)
			if other == "" {
				continue
			}
			if strings.TrimSpace(v.data[other]) == "" {
				needed = true
				break
			}
		}
		if !needed {
			return true
		}
		return strings.TrimSpace(value) != ""
	case "required_without_all":
		others := strings.Split(param, ",")
		for _, other := range others {
			other = strings.TrimSpace(other)
			if other == "" {
				continue
			}
			if strings.TrimSpace(v.data[other]) != "" {
				return true
			}
		}
		return strings.TrimSpace(value) != ""
	case "distinct":
		if value == "" {
			return true
		}
		parts := strings.Split(value, ",")
		seen := make(map[string]bool, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if seen[part] {
				return false
			}
			seen[part] = true
		}
		return true
	case "prohibited":
		return strings.TrimSpace(value) == ""
	case "prohibited_if":
		other, expected, ok := splitFieldValue(param)
		if !ok {
			return false
		}
		if v.data[other] != expected {
			return true
		}
		return strings.TrimSpace(value) == ""
	case "prohibited_unless":
		other, expected, ok := splitFieldValue(param)
		if !ok {
			return false
		}
		if v.data[other] == expected {
			return true
		}
		return strings.TrimSpace(value) == ""
	case "prohibited_with":
		others := strings.Split(param, ",")
		triggered := false
		for _, other := range others {
			other = strings.TrimSpace(other)
			if other == "" {
				continue
			}
			if strings.TrimSpace(v.data[other]) != "" {
				triggered = true
				break
			}
		}
		if !triggered {
			return true
		}
		return strings.TrimSpace(value) == ""
	case "prohibited_with_all":
		others := strings.Split(param, ",")
		for _, other := range others {
			other = strings.TrimSpace(other)
			if other == "" {
				continue
			}
			if strings.TrimSpace(v.data[other]) == "" {
				return true
			}
		}
		return strings.TrimSpace(value) == ""
	case "prohibited_without":
		others := strings.Split(param, ",")
		triggered := false
		for _, other := range others {
			other = strings.TrimSpace(other)
			if other == "" {
				continue
			}
			if strings.TrimSpace(v.data[other]) == "" {
				triggered = true
				break
			}
		}
		if !triggered {
			return true
		}
		return strings.TrimSpace(value) == ""
	case "prohibited_without_all":
		others := strings.Split(param, ",")
		for _, other := range others {
			other = strings.TrimSpace(other)
			if other == "" {
				continue
			}
			if strings.TrimSpace(v.data[other]) != "" {
				return true
			}
		}
		return strings.TrimSpace(value) == ""
	case "accepted":
		return isAccepted(value)
	case "declined":
		return isDeclined(value)
	case "accepted_if":
		other, expected, ok := splitFieldValue(param)
		if !ok {
			return false
		}
		if v.data[other] != expected {
			return true
		}
		return isAccepted(value)
	case "accepted_unless":
		other, expected, ok := splitFieldValue(param)
		if !ok {
			return false
		}
		if v.data[other] == expected {
			return true
		}
		return isAccepted(value)
	case "declined_if":
		other, expected, ok := splitFieldValue(param)
		if !ok {
			return false
		}
		if v.data[other] != expected {
			return true
		}
		return isDeclined(value)
	case "declined_unless":
		other, expected, ok := splitFieldValue(param)
		if !ok {
			return false
		}
		if v.data[other] == expected {
			return true
		}
		return isDeclined(value)
	case "prohibits":
		if strings.TrimSpace(value) == "" {
			return true
		}
		for _, other := range strings.Split(param, ",") {
			other = strings.TrimSpace(other)
			if other == "" {
				continue
			}
			if strings.TrimSpace(v.data[other]) != "" {
				return false
			}
		}
		return true
	case "multiple_of":
		return isMultipleOf(value, param)
	case "decimal":
		return isDecimal(value, param)
	case "uuid":
		if value == "" {
			return true
		}
		return uuid.IsValid(value)
	case "gt", "gte", "lt", "lte":
		return compareToField(v, value, param, rule)
	case "starts_with":
		if value == "" {
			return true
		}
		for _, opt := range strings.Split(param, ",") {
			if strings.HasPrefix(value, strings.TrimSpace(opt)) {
				return true
			}
		}
		return false
	case "ends_with":
		if value == "" {
			return true
		}
		for _, opt := range strings.Split(param, ",") {
			if strings.HasSuffix(value, strings.TrimSpace(opt)) {
				return true
			}
		}
		return false
	case "doesnt_start_with":
		if value == "" {
			return true
		}
		for _, opt := range strings.Split(param, ",") {
			if opt = strings.TrimSpace(opt); opt != "" && strings.HasPrefix(value, opt) {
				return false
			}
		}
		return true
	case "doesnt_end_with":
		if value == "" {
			return true
		}
		for _, opt := range strings.Split(param, ",") {
			if opt = strings.TrimSpace(opt); opt != "" && strings.HasSuffix(value, opt) {
				return false
			}
		}
		return true
	case "contains":
		if value == "" {
			return true
		}
		found := false
		for _, opt := range strings.Split(param, ",") {
			opt = strings.TrimSpace(opt)
			if opt == "" {
				continue
			}
			found = true
			if !strings.Contains(value, opt) {
				return false
			}
		}
		return found || strings.TrimSpace(param) == ""
	case "doesnt_contain":
		if value == "" {
			return true
		}
		for _, opt := range strings.Split(param, ",") {
			opt = strings.TrimSpace(opt)
			if opt != "" && strings.Contains(value, opt) {
				return false
			}
		}
		return true
	case "ascii":
		if value == "" {
			return true
		}
		for i := 0; i < len(value); i++ {
			if value[i] > 127 {
				return false
			}
		}
		return true
	case "hex_color":
		if value == "" {
			return true
		}
		raw := strings.TrimPrefix(strings.TrimSpace(value), "#")
		if len(raw) != 3 && len(raw) != 6 && len(raw) != 8 {
			return false
		}
		for _, c := range raw {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
		return true
	case "timezone":
		if value == "" {
			return true
		}
		_, err := time.LoadLocation(value)
		return err == nil
	case "email":
		if value == "" {
			return true
		}
		re := regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
		return re.MatchString(value)
	case "min":
		return compareSize(value, param, true)
	case "max":
		return compareSize(value, param, false)
	case "numeric":
		if value == "" {
			return true
		}
		_, err := strconv.ParseFloat(value, 64)
		return err == nil
	case "integer":
		if value == "" {
			return true
		}
		_, err := strconv.Atoi(value)
		return err == nil
	case "in":
		return containsOption(value, param)
	case "not_in":
		return !containsOption(value, param)
	case "in_array":
		if value == "" {
			return true
		}
		other := strings.TrimSpace(param)
		for _, opt := range strings.Split(v.data[other], ",") {
			if strings.TrimSpace(opt) == value {
				return true
			}
		}
		return false
	case "not_in_array":
		if value == "" {
			return true
		}
		other := strings.TrimSpace(param)
		for _, opt := range strings.Split(v.data[other], ",") {
			if strings.TrimSpace(opt) == value {
				return false
			}
		}
		return true
	case "confirmed":
		return value == v.data[field+"_confirmation"]
	case "url":
		if value == "" {
			return true
		}
		return strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")
	case "boolean":
		switch strings.ToLower(value) {
		case "1", "0", "true", "false", "on", "off", "yes", "no":
			return true
		default:
			return value == ""
		}
	case "alpha":
		re := regexp.MustCompile(`^[A-Za-z]+$`)
		return value == "" || re.MatchString(value)
	case "alpha_num":
		re := regexp.MustCompile(`^[A-Za-z0-9]+$`)
		return value == "" || re.MatchString(value)
	case "alpha_dash":
		re := regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
		return value == "" || re.MatchString(value)
	case "alpha_spaces":
		re := regexp.MustCompile(`^[A-Za-z ]+$`)
		return value == "" || re.MatchString(value)
	case "ulid":
		if value == "" {
			return true
		}
		if len(value) != 26 {
			return false
		}
		re := regexp.MustCompile(`(?i)^[0-9A-HJKMNP-TV-Z]{26}$`)
		return re.MatchString(value)
	case "same":
		return value == v.data[param]
	case "different":
		return value != v.data[param]
	case "between":
		parts := strings.Split(param, ",")
		if len(parts) != 2 {
			return false
		}
		minN, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
		maxN, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err1 != nil || err2 != nil {
			return false
		}
		if n, err := strconv.ParseFloat(value, 64); err == nil {
			return n >= float64(minN) && n <= float64(maxN)
		}
		size := utf8.RuneCountInString(value)
		return size >= minN && size <= maxN
	case "size":
		n, err := strconv.Atoi(param)
		if err != nil {
			return false
		}
		if num, err := strconv.ParseFloat(value, 64); err == nil && !strings.Contains(value, "@") {
			// numeric values compare by number when they look numeric-only
			if _, ierr := strconv.Atoi(value); ierr == nil || strings.Contains(value, ".") {
				return num == float64(n)
			}
		}
		return utf8.RuneCountInString(value) == n
	case "digits":
		n, err := strconv.Atoi(param)
		if err != nil {
			return false
		}
		re := regexp.MustCompile(`^\d+$`)
		return value == "" || (re.MatchString(value) && len(value) == n)
	case "digits_between":
		parts := strings.Split(param, ",")
		if len(parts) != 2 {
			return false
		}
		minN, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
		maxN, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err1 != nil || err2 != nil {
			return false
		}
		if value == "" {
			return true
		}
		re := regexp.MustCompile(`^\d+$`)
		if !re.MatchString(value) {
			return false
		}
		n := len(value)
		return n >= minN && n <= maxN
	case "lowercase":
		return value == "" || value == strings.ToLower(value)
	case "uppercase":
		return value == "" || value == strings.ToUpper(value)
	case "date":
		if value == "" {
			return true
		}
		_, ok := parseDate(value)
		return ok
	case "before":
		if value == "" {
			return true
		}
		left, ok1 := parseDate(value)
		right, ok2 := parseDateParam(v, param)
		return ok1 && ok2 && left.Before(right)
	case "after":
		if value == "" {
			return true
		}
		left, ok1 := parseDate(value)
		right, ok2 := parseDateParam(v, param)
		return ok1 && ok2 && left.After(right)
	case "before_or_equal":
		if value == "" {
			return true
		}
		left, ok1 := parseDate(value)
		right, ok2 := parseDateParam(v, param)
		return ok1 && ok2 && (left.Before(right) || left.Equal(right))
	case "after_or_equal":
		if value == "" {
			return true
		}
		left, ok1 := parseDate(value)
		right, ok2 := parseDateParam(v, param)
		return ok1 && ok2 && (left.After(right) || left.Equal(right))
	case "date_format":
		if value == "" {
			return true
		}
		layout := strings.TrimSpace(param)
		if layout == "" {
			return false
		}
		_, err := time.Parse(layout, value)
		return err == nil
	case "mac_address":
		if value == "" {
			return true
		}
		_, err := net.ParseMAC(value)
		return err == nil
	case "regex":
		if value == "" {
			return true
		}
		re, err := regexp.Compile(param)
		if err != nil {
			return false
		}
		return re.MatchString(value)
	case "not_regex":
		if value == "" {
			return true
		}
		re, err := regexp.Compile(param)
		if err != nil {
			return false
		}
		return !re.MatchString(value)
	case "json":
		if value == "" {
			return true
		}
		return json.Valid([]byte(value))
	case "ip":
		if value == "" {
			return true
		}
		return net.ParseIP(value) != nil
	case "ipv4":
		if value == "" {
			return true
		}
		ip := net.ParseIP(value)
		return ip != nil && ip.To4() != nil
	case "ipv6":
		if value == "" {
			return true
		}
		ip := net.ParseIP(value)
		return ip != nil && ip.To4() == nil
	case "mimes", "extensions":
		if value == "" {
			return true
		}
		ext := strings.ToLower(strings.TrimPrefix(filepathExt(value), "."))
		return containsOption(ext, param)
	case "image":
		if value == "" {
			return true
		}
		ext := strings.ToLower(strings.TrimPrefix(filepathExt(value), "."))
		return containsOption(ext, "jpg,jpeg,png,gif,bmp,svg,webp")
	case "semver":
		if value == "" {
			return true
		}
		return str.IsSemver(value)
	case "unique":
		return v.checkPresence(param, value, true)
	case "exists":
		return v.checkPresence(param, value, false)
	default:
		return true
	}
}

func (v *Validator) checkPresence(param, value string, unique bool) bool {
	if value == "" {
		return true
	}
	checker := v.presenceChecker
	if checker == nil {
		checker = defaultPresenceChecker
	}
	if checker == nil {
		return true
	}
	parts := strings.Split(param, ",")
	if len(parts) < 2 {
		return true
	}
	table := strings.TrimSpace(parts[0])
	column := strings.TrimSpace(parts[1])
	exists, err := checker(table, column, value)
	if err != nil {
		return false
	}
	if unique {
		return !exists
	}
	return exists
}

func compareSize(value, param string, min bool) bool {
	n, err := strconv.Atoi(param)
	if err != nil {
		return false
	}
	if num, err := strconv.ParseFloat(value, 64); err == nil && looksNumeric(value) {
		if min {
			return num >= float64(n)
		}
		return num <= float64(n)
	}
	size := utf8.RuneCountInString(value)
	if min {
		return size >= n
	}
	return size <= n
}

func looksNumeric(value string) bool {
	if value == "" {
		return false
	}
	_, err := strconv.ParseFloat(value, 64)
	return err == nil
}

func containsOption(value, param string) bool {
	options := strings.Split(param, ",")
	for _, option := range options {
		if value == strings.TrimSpace(option) {
			return true
		}
	}
	return false
}

func parseDate(value string) (time.Time, bool) {
	formats := []string{
		time.RFC3339,
		"2006-01-02",
		"2006-01-02 15:04:05",
		"02/01/2006",
		"01/02/2006",
	}
	for _, format := range formats {
		if ts, err := time.Parse(format, value); err == nil {
			return ts, true
		}
	}
	return time.Time{}, false
}

func parseDateParam(v *Validator, param string) (time.Time, bool) {
	if ts, ok := parseDate(param); ok {
		return ts, true
	}
	if v != nil {
		if other, exists := v.data[param]; exists {
			return parseDate(other)
		}
	}
	return time.Time{}, false
}

func filepathExt(value string) string {
	if idx := strings.LastIndex(value, "."); idx >= 0 {
		return value[idx:]
	}
	return ""
}

func splitRule(rule string) (string, string) {
	if strings.Contains(rule, ":") {
		parts := strings.SplitN(rule, ":", 2)
		return parts[0], parts[1]
	}
	return rule, ""
}

func splitFieldValue(param string) (field, value string, ok bool) {
	parts := strings.SplitN(param, ",", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	field = strings.TrimSpace(parts[0])
	value = strings.TrimSpace(parts[1])
	if field == "" {
		return "", "", false
	}
	return field, value, true
}

func (v *Validator) messageFor(rule, field, param string) string {
	attr := v.attributeName(field)
	if v.customMessages != nil {
		if msg, ok := v.customMessages[field+"."+rule]; ok {
			return replaceMessagePlaceholders(msg, attr, field, param)
		}
		if msg, ok := v.customMessages[field]; ok {
			return replaceMessagePlaceholders(msg, attr, field, param)
		}
	}
	return defaultMessage(rule, attr, param)
}

func (v *Validator) attributeName(field string) string {
	if v.customAttributes != nil {
		if name, ok := v.customAttributes[field]; ok && name != "" {
			return name
		}
	}
	return strings.ReplaceAll(field, "_", " ")
}

func replaceMessagePlaceholders(msg, attribute, field, param string) string {
	msg = strings.ReplaceAll(msg, ":attribute", attribute)
	msg = strings.ReplaceAll(msg, ":Attribute", attribute)
	msg = strings.ReplaceAll(msg, ":field", field)
	msg = strings.ReplaceAll(msg, ":value", param)
	msg = strings.ReplaceAll(msg, ":other", param)
	return msg
}

func defaultMessage(rule, field, param string) string {
	switch rule {
	case "required":
		return fmt.Sprintf("The %s field is required.", field)
	case "filled":
		return fmt.Sprintf("The %s field must not be empty when present.", field)
	case "present":
		return fmt.Sprintf("The %s field must be present.", field)
	case "missing":
		return fmt.Sprintf("The %s field must be missing.", field)
	case "present_if":
		return fmt.Sprintf("The %s field must be present when %s.", field, strings.ReplaceAll(param, ",", " is "))
	case "present_unless":
		return fmt.Sprintf("The %s field must be present unless %s.", field, strings.ReplaceAll(param, ",", " is "))
	case "missing_if":
		return fmt.Sprintf("The %s field must be missing when %s.", field, strings.ReplaceAll(param, ",", " is "))
	case "missing_unless":
		return fmt.Sprintf("The %s field must be missing unless %s.", field, strings.ReplaceAll(param, ",", " is "))
	case "present_with":
		return fmt.Sprintf("The %s field must be present when %s is present.", field, strings.ReplaceAll(param, ",", " / "))
	case "present_with_all":
		return fmt.Sprintf("The %s field must be present when %s are present.", field, strings.ReplaceAll(param, ",", " / "))
	case "missing_with":
		return fmt.Sprintf("The %s field must be missing when %s is present.", field, strings.ReplaceAll(param, ",", " / "))
	case "missing_with_all":
		return fmt.Sprintf("The %s field must be missing when %s are present.", field, strings.ReplaceAll(param, ",", " / "))
	case "present_without":
		return fmt.Sprintf("The %s field must be present when %s is not present.", field, strings.ReplaceAll(param, ",", " / "))
	case "present_without_all":
		return fmt.Sprintf("The %s field must be present when none of %s are present.", field, strings.ReplaceAll(param, ",", " / "))
	case "missing_without":
		return fmt.Sprintf("The %s field must be missing when %s is not present.", field, strings.ReplaceAll(param, ",", " / "))
	case "missing_without_all":
		return fmt.Sprintf("The %s field must be missing when none of %s are present.", field, strings.ReplaceAll(param, ",", " / "))
	case "required_if":
		return fmt.Sprintf("The %s field is required when %s.", field, strings.ReplaceAll(param, ",", " is "))
	case "required_if_accepted":
		return fmt.Sprintf("The %s field is required when %s is accepted.", field, param)
	case "required_if_declined":
		return fmt.Sprintf("The %s field is required when %s is declined.", field, param)
	case "required_unless":
		return fmt.Sprintf("The %s field is required unless %s.", field, strings.ReplaceAll(param, ",", " is "))
	case "required_with":
		return fmt.Sprintf("The %s field is required when %s is present.", field, strings.ReplaceAll(param, ",", " / "))
	case "required_with_all":
		return fmt.Sprintf("The %s field is required when %s are present.", field, strings.ReplaceAll(param, ",", " / "))
	case "required_without":
		return fmt.Sprintf("The %s field is required when %s is not present.", field, strings.ReplaceAll(param, ",", " / "))
	case "required_without_all":
		return fmt.Sprintf("The %s field is required when none of %s are present.", field, strings.ReplaceAll(param, ",", " / "))
	case "distinct":
		return fmt.Sprintf("The %s field has a duplicate value.", field)
	case "prohibited", "prohibited_if", "prohibited_unless", "prohibited_with", "prohibited_with_all", "prohibited_without", "prohibited_without_all":
		return fmt.Sprintf("The %s field is prohibited.", field)
	case "accepted":
		return fmt.Sprintf("The %s must be accepted.", field)
	case "declined":
		return fmt.Sprintf("The %s must be declined.", field)
	case "accepted_if":
		return fmt.Sprintf("The %s must be accepted when %s.", field, strings.ReplaceAll(param, ",", " is "))
	case "accepted_unless":
		return fmt.Sprintf("The %s must be accepted unless %s.", field, strings.ReplaceAll(param, ",", " is "))
	case "declined_if":
		return fmt.Sprintf("The %s must be declined when %s.", field, strings.ReplaceAll(param, ",", " is "))
	case "declined_unless":
		return fmt.Sprintf("The %s must be declined unless %s.", field, strings.ReplaceAll(param, ",", " is "))
	case "prohibits":
		return fmt.Sprintf("The %s field prohibits %s.", field, strings.ReplaceAll(param, ",", " / "))
	case "multiple_of":
		return fmt.Sprintf("The %s field must be a multiple of %s.", field, param)
	case "decimal":
		return fmt.Sprintf("The %s field must have %s decimal places.", field, strings.ReplaceAll(param, ",", " to "))
	case "uuid":
		return fmt.Sprintf("The %s field must be a valid UUID.", field)
	case "ulid":
		return fmt.Sprintf("The %s field must be a valid ULID.", field)
	case "alpha", "alpha_num", "alpha_dash", "alpha_spaces":
		return fmt.Sprintf("The %s field format is invalid.", field)
	case "gt":
		return fmt.Sprintf("The %s field must be greater than %s.", field, strings.ReplaceAll(param, "_", " "))
	case "gte":
		return fmt.Sprintf("The %s field must be greater than or equal to %s.", field, strings.ReplaceAll(param, "_", " "))
	case "lt":
		return fmt.Sprintf("The %s field must be less than %s.", field, strings.ReplaceAll(param, "_", " "))
	case "lte":
		return fmt.Sprintf("The %s field must be less than or equal to %s.", field, strings.ReplaceAll(param, "_", " "))
	case "starts_with":
		return fmt.Sprintf("The %s field must start with one of the following: %s.", field, param)
	case "ends_with":
		return fmt.Sprintf("The %s field must end with one of the following: %s.", field, param)
	case "doesnt_start_with":
		return fmt.Sprintf("The %s field must not start with one of the following: %s.", field, param)
	case "doesnt_end_with":
		return fmt.Sprintf("The %s field must not end with one of the following: %s.", field, param)
	case "contains":
		return fmt.Sprintf("The %s field must contain all of the following: %s.", field, param)
	case "doesnt_contain":
		return fmt.Sprintf("The %s field must not contain any of the following: %s.", field, param)
	case "ascii":
		return fmt.Sprintf("The %s field must be ASCII.", field)
	case "hex_color":
		return fmt.Sprintf("The %s field must be a valid hex color.", field)
	case "timezone":
		return fmt.Sprintf("The %s field must be a valid timezone.", field)
	case "email":
		return fmt.Sprintf("The %s field must be a valid email address.", field)
	case "min":
		return fmt.Sprintf("The %s field must be at least %s.", field, param)
	case "max":
		return fmt.Sprintf("The %s field must not be greater than %s.", field, param)
	case "numeric":
		return fmt.Sprintf("The %s field must be a number.", field)
	case "integer":
		return fmt.Sprintf("The %s field must be an integer.", field)
	case "in", "not_in":
		return fmt.Sprintf("The selected %s is invalid.", field)
	case "in_array":
		return fmt.Sprintf("The selected %s is invalid.", field)
	case "not_in_array":
		return fmt.Sprintf("The selected %s is invalid.", field)
	case "confirmed":
		return fmt.Sprintf("The %s confirmation does not match.", field)
	case "url":
		return fmt.Sprintf("The %s field must be a valid URL.", field)
	case "boolean":
		return fmt.Sprintf("The %s field must be true or false.", field)
	case "same":
		return fmt.Sprintf("The %s and %s fields must match.", field, strings.ReplaceAll(param, "_", " "))
	case "different":
		return fmt.Sprintf("The %s and %s fields must be different.", field, strings.ReplaceAll(param, "_", " "))
	case "between":
		return fmt.Sprintf("The %s field must be between %s.", field, strings.ReplaceAll(param, ",", " and "))
	case "size":
		return fmt.Sprintf("The %s field must be %s.", field, param)
	case "digits":
		return fmt.Sprintf("The %s field must be %s digits.", field, param)
	case "digits_between":
		return fmt.Sprintf("The %s field must be between %s digits.", field, strings.ReplaceAll(param, ",", " and "))
	case "lowercase":
		return fmt.Sprintf("The %s field must be lowercase.", field)
	case "uppercase":
		return fmt.Sprintf("The %s field must be uppercase.", field)
	case "date":
		return fmt.Sprintf("The %s field must be a valid date.", field)
	case "before":
		return fmt.Sprintf("The %s field must be a date before %s.", field, param)
	case "after":
		return fmt.Sprintf("The %s field must be a date after %s.", field, param)
	case "before_or_equal":
		return fmt.Sprintf("The %s field must be a date before or equal to %s.", field, param)
	case "after_or_equal":
		return fmt.Sprintf("The %s field must be a date after or equal to %s.", field, param)
	case "date_format":
		return fmt.Sprintf("The %s field does not match the format %s.", field, param)
	case "mac_address":
		return fmt.Sprintf("The %s field must be a valid MAC address.", field)
	case "regex":
		return fmt.Sprintf("The %s field format is invalid.", field)
	case "not_regex":
		return fmt.Sprintf("The %s field format is invalid.", field)
	case "json":
		return fmt.Sprintf("The %s field must be valid JSON.", field)
	case "ip":
		return fmt.Sprintf("The %s field must be a valid IP address.", field)
	case "ipv4":
		return fmt.Sprintf("The %s field must be a valid IPv4 address.", field)
	case "ipv6":
		return fmt.Sprintf("The %s field must be a valid IPv6 address.", field)
	case "mimes", "extensions", "image":
		types := param
		if rule == "image" && types == "" {
			types = "jpg, jpeg, png, gif, bmp, svg, webp"
		}
		return fmt.Sprintf("The %s field must be a file of type: %s.", field, types)
	case "semver":
		return fmt.Sprintf("The %s field must be a valid semantic version.", field)
	case "unique":
		return fmt.Sprintf("The %s has already been taken.", field)
	case "exists":
		return fmt.Sprintf("The selected %s is invalid.", field)
	default:
		return fmt.Sprintf("The %s field is invalid.", field)
	}
}

func isAccepted(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "yes", "on", "1", "true":
		return true
	default:
		return false
	}
}

func isDeclined(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "no", "off", "0", "false":
		return true
	default:
		return false
	}
}

func isMultipleOf(value, param string) bool {
	if value == "" {
		return true
	}
	n, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return false
	}
	m, err := strconv.ParseFloat(strings.TrimSpace(param), 64)
	if err != nil || m == 0 {
		return false
	}
	quot := n / m
	nearest := math.Round(quot)
	return math.Abs(quot-nearest) < 1e-9
}

func isDecimal(value, param string) bool {
	if value == "" {
		return true
	}
	minPlaces, maxPlaces, ok := parseDecimalParam(param)
	if !ok {
		return false
	}
	places, ok := countDecimalPlaces(value)
	if !ok {
		return false
	}
	return places >= minPlaces && places <= maxPlaces
}

func parseDecimalParam(param string) (minPlaces, maxPlaces int, ok bool) {
	parts := strings.Split(param, ",")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		return 0, 0, false
	}
	minPlaces, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || minPlaces < 0 {
		return 0, 0, false
	}
	maxPlaces = minPlaces
	if len(parts) > 1 {
		maxPlaces, err = strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil || maxPlaces < minPlaces {
			return 0, 0, false
		}
	}
	return minPlaces, maxPlaces, true
}

func countDecimalPlaces(value string) (int, bool) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return 0, false
	}
	if _, err := strconv.ParseFloat(raw, 64); err != nil {
		return 0, false
	}
	if strings.ContainsAny(raw, "eE") {
		return 0, false
	}
	if strings.HasPrefix(raw, "+") || strings.HasPrefix(raw, "-") {
		raw = raw[1:]
	}
	parts := strings.SplitN(raw, ".", 2)
	if len(parts) == 1 {
		return 0, true
	}
	if parts[1] == "" {
		return 0, false
	}
	return len(parts[1]), true
}

func compareToField(v *Validator, value, otherField, rule string) bool {
	otherField = strings.TrimSpace(otherField)
	if otherField == "" {
		return false
	}
	other, exists := v.data[otherField]
	if !exists {
		other = otherField // allow literal numeric comparisons like gt:17
	}
	if value == "" || other == "" {
		return true
	}
	a, aErr := strconv.ParseFloat(value, 64)
	b, bErr := strconv.ParseFloat(other, 64)
	if aErr == nil && bErr == nil {
		switch rule {
		case "gt":
			return a > b
		case "gte":
			return a >= b
		case "lt":
			return a < b
		case "lte":
			return a <= b
		}
	}
	switch rule {
	case "gt":
		return value > other
	case "gte":
		return value >= other
	case "lt":
		return value < other
	case "lte":
		return value <= other
	default:
		return false
	}
}

func hasBail(rules []string) bool {
	for _, rule := range rules {
		if strings.TrimSpace(rule) == "bail" {
			return true
		}
	}
	return false
}

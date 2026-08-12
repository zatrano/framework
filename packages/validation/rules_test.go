package validation_test

import (
	"testing"

	"github.com/zatrano/framework/packages/validation"
)

func TestNewValidationRules(t *testing.T) {
	v := validation.Make(map[string]string{
		"role":    "admin",
		"age":     "25",
		"code":    "1234",
		"starts":  "2026-01-01",
		"payload": `{"ok":true}`,
		"avatar":  "photo.png",
		"skip":    "",
	}, map[string]string{
		"role":    "required|not_in:guest,banned",
		"age":     "integer|between:18,99",
		"code":    "digits:4",
		"starts":  "date|before:2030-01-01",
		"payload": "json",
		"avatar":  "image",
		"skip":    "nullable|email",
	})
	if v.Fails() {
		t.Fatalf("expected pass, got %#v", v.Errors())
	}
}

func TestNotInFails(t *testing.T) {
	v := validation.Make(map[string]string{"role": "guest"}, map[string]string{
		"role": "not_in:guest,banned",
	})
	if !v.Fails() {
		t.Fatal("expected failure")
	}
}

func TestAlphaDashAndULID(t *testing.T) {
	ok := validation.Make(map[string]string{
		"slug": "hello_world-1",
		"id":   "01ARZ3NDEKTSV4RRFFQ69G5FAV",
	}, map[string]string{
		"slug": "alpha_dash",
		"id":   "ulid",
	})
	if ok.Fails() {
		t.Fatalf("expected pass %#v", ok.Errors())
	}
	fail := validation.Make(map[string]string{
		"slug": "hello world",
		"id":   "not-a-ulid",
	}, map[string]string{
		"slug": "alpha_dash",
		"id":   "ulid",
	})
	if !fail.Fails() || !fail.Errors().Has("slug") || !fail.Errors().Has("id") {
		t.Fatalf("expected failures %#v", fail.Errors())
	}
}

func TestDateFormatMacAndInclusiveCompare(t *testing.T) {
	ok := validation.Make(map[string]string{
		"day":   "2026-08-03",
		"start": "2026-08-01",
		"end":   "2026-08-03",
		"mac":   "01:23:45:67:89:ab",
	}, map[string]string{
		"day":   "date_format:2006-01-02",
		"start": "before_or_equal:2026-08-03",
		"end":   "after_or_equal:2026-08-03",
		"mac":   "mac_address",
	})
	if ok.Fails() {
		t.Fatalf("expected pass %#v", ok.Errors())
	}
	fail := validation.Make(map[string]string{
		"day": "03/08/2026",
		"mac": "not-mac",
	}, map[string]string{
		"day": "date_format:2006-01-02",
		"mac": "mac_address",
	})
	if !fail.Fails() {
		t.Fatal("expected failure")
	}
	fields := validation.Make(map[string]string{
		"start": "2026-08-01",
		"end":   "2026-08-03",
	}, map[string]string{
		"start": "before_or_equal:end",
		"end":   "after_or_equal:start",
	})
	if fields.Fails() {
		t.Fatalf("expected field compare pass %#v", fields.Errors())
	}
}

func TestFormatIPDigitsAndProhibitedUnless(t *testing.T) {
	ok := validation.Make(map[string]string{
		"code":  "hello",
		"pin":   "1234",
		"v4":    "127.0.0.1",
		"v6":    "2001:db8::1",
		"mode":  "edit",
		"extra": "",
	}, map[string]string{
		"code":  "lowercase",
		"pin":   "digits_between:3,6",
		"v4":    "ipv4",
		"v6":    "ipv6",
		"extra": "prohibited_unless:mode,create",
	})
	if ok.Fails() {
		t.Fatalf("expected pass %#v", ok.Errors())
	}

	fail := validation.Make(map[string]string{
		"code":  "Hello",
		"pin":   "12a",
		"v4":    "2001:db8::1",
		"v6":    "127.0.0.1",
		"mode":  "edit",
		"extra": "nope",
	}, map[string]string{
		"code":  "lowercase",
		"title": "uppercase",
		"pin":   "digits_between:3,6",
		"v4":    "ipv4",
		"v6":    "ipv6",
		"extra": "prohibited_unless:mode,create",
	})
	if !fail.Fails() {
		t.Fatal("expected failure")
	}
	upper := validation.Make(map[string]string{"title": "HELLO"}, map[string]string{"title": "uppercase"})
	if upper.Fails() {
		t.Fatalf("uppercase pass %#v", upper.Errors())
	}
	allowed := validation.Make(map[string]string{
		"mode":  "create",
		"extra": "ok",
	}, map[string]string{
		"extra": "prohibited_unless:mode,create",
	})
	if allowed.Fails() {
		t.Fatalf("prohibited_unless allow %#v", allowed.Errors())
	}
}

func TestAsciiHexTimezoneAndDoesntAffix(t *testing.T) {
	ok := validation.Make(map[string]string{
		"slug":  "hello",
		"color": "#FfAa00",
		"tz":    "UTC",
		"path":  "api/v1",
	}, map[string]string{
		"slug":  "ascii",
		"color": "hex_color",
		"tz":    "timezone",
		"path":  "doesnt_start_with:admin,private|doesnt_end_with:.bak,.tmp",
	})
	if ok.Fails() {
		t.Fatalf("expected pass %#v", ok.Errors())
	}

	fail := validation.Make(map[string]string{
		"slug":  "şeker",
		"color": "red",
		"tz":    "Not/AZone",
		"path":  "admin/home.bak",
	}, map[string]string{
		"slug":  "ascii",
		"color": "hex_color",
		"tz":    "timezone",
		"path":  "doesnt_start_with:admin|doesnt_end_with:.bak",
	})
	if !fail.Fails() {
		t.Fatal("expected failure")
	}
}

func TestPresentMissingIfUnless(t *testing.T) {
	ok := validation.Make(map[string]string{
		"mode":  "edit",
		"notes": "",
	}, map[string]string{
		"notes": "present_if:mode,edit",
		"draft": "missing_if:mode,edit",
	})
	if ok.Fails() {
		t.Fatalf("expected pass %#v", ok.Errors())
	}

	failPresent := validation.Make(map[string]string{
		"mode": "edit",
	}, map[string]string{
		"notes": "present_if:mode,edit",
	})
	if !failPresent.Fails() {
		t.Fatal("expected present_if failure")
	}

	failMissing := validation.Make(map[string]string{
		"mode":  "edit",
		"draft": "x",
	}, map[string]string{
		"draft": "missing_if:mode,edit",
	})
	if !failMissing.Fails() {
		t.Fatal("expected missing_if failure")
	}

	unlessOk := validation.Make(map[string]string{
		"mode":  "view",
		"notes": "",
	}, map[string]string{
		"notes": "present_unless:mode,edit",
		"draft": "missing_unless:mode,edit",
	})
	if unlessOk.Fails() {
		t.Fatalf("present/missing unless %#v", unlessOk.Errors())
	}
}

func TestPresentMissingWithRules(t *testing.T) {
	ok := validation.Make(map[string]string{
		"email": "a@b.c",
		"code":  "",
	}, map[string]string{
		"code":  "present_with:email",
		"token": "missing_with:email",
	})
	if ok.Fails() {
		t.Fatalf("present/missing with %#v", ok.Errors())
	}

	failPresent := validation.Make(map[string]string{
		"email": "a@b.c",
	}, map[string]string{
		"code": "present_with:email",
	})
	if !failPresent.Fails() {
		t.Fatal("present_with should fail")
	}

	failMissing := validation.Make(map[string]string{
		"email": "a@b.c",
		"token": "x",
	}, map[string]string{
		"token": "missing_with:email",
	})
	if !failMissing.Fails() {
		t.Fatal("missing_with should fail")
	}

	withAllOk := validation.Make(map[string]string{
		"city":  "Istanbul",
		"email": "a@b.c",
		"note":  "",
	}, map[string]string{
		"note":  "present_with_all:city,email",
		"extra": "missing_with_all:city,email",
	})
	if withAllOk.Fails() {
		t.Fatalf("with_all ok %#v", withAllOk.Errors())
	}

	withAllFail := validation.Make(map[string]string{
		"city":  "Istanbul",
		"email": "a@b.c",
		"extra": "x",
	}, map[string]string{
		"note":  "present_with_all:city,email",
		"extra": "missing_with_all:city,email",
	})
	if !withAllFail.Fails() {
		t.Fatal("with_all should fail")
	}

	skip := validation.Make(map[string]string{
		"city": "Istanbul",
	}, map[string]string{
		"note":  "present_with_all:city,email",
		"extra": "missing_with:email",
	})
	if skip.Fails() {
		t.Fatalf("skip when not triggered %#v", skip.Errors())
	}
}

func TestPresentMissingWithoutRules(t *testing.T) {
	ok := validation.Make(map[string]string{
		"code": "",
	}, map[string]string{
		"code":  "present_without:email",
		"token": "missing_without:email",
	})
	if ok.Fails() {
		t.Fatalf("present/missing without %#v", ok.Errors())
	}

	failPresent := validation.Make(map[string]string{}, map[string]string{
		"code": "present_without:email",
	})
	if !failPresent.Fails() {
		t.Fatal("present_without should fail")
	}

	failMissing := validation.Make(map[string]string{
		"token": "x",
	}, map[string]string{
		"token": "missing_without:email",
	})
	if !failMissing.Fails() {
		t.Fatal("missing_without should fail")
	}

	withAllOk := validation.Make(map[string]string{
		"note": "",
	}, map[string]string{
		"note":  "present_without_all:city,email",
		"extra": "missing_without_all:city,email",
	})
	if withAllOk.Fails() {
		t.Fatalf("without_all ok %#v", withAllOk.Errors())
	}

	withAllFail := validation.Make(map[string]string{
		"extra": "x",
	}, map[string]string{
		"note":  "present_without_all:city,email",
		"extra": "missing_without_all:city,email",
	})
	if !withAllFail.Fails() {
		t.Fatal("without_all should fail")
	}

	skip := validation.Make(map[string]string{
		"email": "a@b.c",
	}, map[string]string{
		"note":  "present_without_all:city,email",
		"extra": "missing_without:email",
	})
	if skip.Fails() {
		t.Fatalf("skip when not triggered %#v", skip.Errors())
	}
}

func TestFilledPresentMissing(t *testing.T) {
	ok := validation.Make(map[string]string{"name": "Ada", "token": "abc"}, map[string]string{
		"name":  "filled",
		"token": "present",
		"extra": "missing",
	})
	if ok.Fails() {
		t.Fatalf("expected pass %#v", ok.Errors())
	}
	fail := validation.Make(map[string]string{"name": "", "extra": "1"}, map[string]string{
		"name":  "filled",
		"extra": "missing",
		"slug":  "starts_with:post-,page-",
		"file":  "ends_with:.png,.jpg",
		"bad":   "bail|required|email",
	})
	if !fail.Fails() {
		t.Fatal("expected failure")
	}
	if len(fail.Errors()["bad"]) != 1 {
		t.Fatalf("bail should stop at first error, got %#v", fail.Errors()["bad"])
	}
}

func TestConditionalValidationRules(t *testing.T) {
	ok := validation.Make(map[string]string{
		"type":    "company",
		"company": "Acme",
		"email":   "a@b.c",
		"phone":   "123",
	}, map[string]string{
		"company": "required_if:type,company",
		"name":    "required_unless:type,company",
		"phone":   "required_with:email",
		"token":   "prohibited",
	})
	if ok.Fails() {
		t.Fatalf("expected pass %#v", ok.Errors())
	}

	fail := validation.Make(map[string]string{
		"type":  "company",
		"token": "x",
	}, map[string]string{
		"company": "required_if:type,company",
		"token":   "prohibited_if:type,company",
	})
	if !fail.Fails() {
		t.Fatal("expected failure")
	}
}

func TestProhibitedWithRules(t *testing.T) {
	ok := validation.Make(map[string]string{
		"email": "a@b.c",
		"code":  "",
	}, map[string]string{
		"code": "prohibited_with:email",
	})
	if ok.Fails() {
		t.Fatalf("prohibited_with empty %#v", ok.Errors())
	}

	fail := validation.Make(map[string]string{
		"email": "a@b.c",
		"code":  "x",
	}, map[string]string{
		"code": "prohibited_with:email",
	})
	if !fail.Fails() {
		t.Fatal("prohibited_with should fail")
	}

	withAllOk := validation.Make(map[string]string{
		"city":  "Istanbul",
		"email": "a@b.c",
		"note":  "",
	}, map[string]string{
		"note": "prohibited_with_all:city,email",
	})
	if withAllOk.Fails() {
		t.Fatalf("prohibited_with_all empty %#v", withAllOk.Errors())
	}
	withAllFail := validation.Make(map[string]string{
		"city":  "Istanbul",
		"email": "a@b.c",
		"note":  "x",
	}, map[string]string{
		"note": "prohibited_with_all:city,email",
	})
	if !withAllFail.Fails() {
		t.Fatal("prohibited_with_all should fail")
	}
	withAllSkip := validation.Make(map[string]string{
		"city": "Istanbul",
		"note": "ok",
	}, map[string]string{
		"note": "prohibited_with_all:city,email",
	})
	if withAllSkip.Fails() {
		t.Fatalf("prohibited_with_all skip %#v", withAllSkip.Errors())
	}

	withoutOk := validation.Make(map[string]string{
		"phone": "",
		"code":  "",
	}, map[string]string{
		"code": "prohibited_without:phone",
	})
	if withoutOk.Fails() {
		t.Fatalf("prohibited_without empty %#v", withoutOk.Errors())
	}
	withoutFail := validation.Make(map[string]string{
		"phone": "",
		"code":  "x",
	}, map[string]string{
		"code": "prohibited_without:phone",
	})
	if !withoutFail.Fails() {
		t.Fatal("prohibited_without should fail")
	}
	withoutSkip := validation.Make(map[string]string{
		"phone": "123",
		"code":  "x",
	}, map[string]string{
		"code": "prohibited_without:phone",
	})
	if withoutSkip.Fails() {
		t.Fatalf("prohibited_without skip %#v", withoutSkip.Errors())
	}

	withoutAllOk := validation.Make(map[string]string{
		"note": "",
	}, map[string]string{
		"note": "prohibited_without_all:email,phone",
	})
	if withoutAllOk.Fails() {
		t.Fatalf("prohibited_without_all empty %#v", withoutAllOk.Errors())
	}
	withoutAllFail := validation.Make(map[string]string{
		"note": "x",
	}, map[string]string{
		"note": "prohibited_without_all:email,phone",
	})
	if !withoutAllFail.Fails() {
		t.Fatal("prohibited_without_all should fail")
	}
	withoutAllSkip := validation.Make(map[string]string{
		"email": "a@b.c",
		"note":  "ok",
	}, map[string]string{
		"note": "prohibited_without_all:email,phone",
	})
	if withoutAllSkip.Fails() {
		t.Fatalf("prohibited_without_all skip %#v", withoutAllSkip.Errors())
	}
}

func TestExcludeWithRules(t *testing.T) {
	withEmail := validation.Make(map[string]string{
		"email": "a@b.c",
		"code":  "secret",
	}, map[string]string{
		"code": "exclude_with:email|required|min:3",
	})
	if withEmail.Fails() {
		t.Fatalf("exclude_with should skip validation %#v", withEmail.Errors())
	}
	data, err := withEmail.Validated()
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := data["code"]; exists {
		t.Fatalf("code should be excluded: %#v", data)
	}

	kept := validation.Make(map[string]string{
		"code": "abc",
	}, map[string]string{
		"code": "exclude_with:email|required|min:3",
	})
	if kept.Fails() {
		t.Fatalf("exclude_with keep %#v", kept.Errors())
	}
	keptData, _ := kept.Validated()
	if keptData["code"] != "abc" {
		t.Fatalf("code should remain: %#v", keptData)
	}

	withAll := validation.Make(map[string]string{
		"city":  "Istanbul",
		"email": "a@b.c",
		"note":  "x",
	}, map[string]string{
		"note": "exclude_with_all:city,email|min:10",
	})
	if withAll.Fails() {
		t.Fatalf("exclude_with_all %#v", withAll.Errors())
	}
	withAllData, _ := withAll.Validated()
	if _, exists := withAllData["note"]; exists {
		t.Fatal("note should be excluded")
	}

	without := validation.Make(map[string]string{
		"token": "x",
	}, map[string]string{
		"token": "exclude_without:phone|min:10",
	})
	if without.Fails() {
		t.Fatalf("exclude_without %#v", without.Errors())
	}
	withoutData, _ := without.Validated()
	if _, exists := withoutData["token"]; exists {
		t.Fatal("token should be excluded")
	}

	withoutAll := validation.Make(map[string]string{
		"extra": "x",
	}, map[string]string{
		"extra": "exclude_without_all:email,phone|min:10",
	})
	if withoutAll.Fails() {
		t.Fatalf("exclude_without_all %#v", withoutAll.Errors())
	}
	withoutAllData, _ := withoutAll.Validated()
	if _, exists := withoutAllData["extra"]; exists {
		t.Fatal("extra should be excluded")
	}

	withoutAllKeep := validation.Make(map[string]string{
		"email": "a@b.c",
		"extra": "longenough",
	}, map[string]string{
		"extra": "exclude_without_all:email,phone|min:8",
	})
	if withoutAllKeep.Fails() {
		t.Fatalf("exclude_without_all keep %#v", withoutAllKeep.Errors())
	}
	keepData, _ := withoutAllKeep.Validated()
	if keepData["extra"] != "longenough" {
		t.Fatalf("extra should remain: %#v", keepData)
	}
}

func TestAcceptedUUIDCompareRules(t *testing.T) {
	ok := validation.Make(map[string]string{
		"terms": "yes",
		"id":    "550e8400-e29b-41d4-a716-446655440000",
		"min":   "10",
		"max":   "20",
		"opt":   "no",
	}, map[string]string{
		"terms": "accepted",
		"id":    "uuid",
		"max":   "gt:min",
		"opt":   "declined",
	})
	if ok.Fails() {
		t.Fatalf("expected pass %#v", ok.Errors())
	}
	fail := validation.Make(map[string]string{"terms": "no", "age": "10"}, map[string]string{
		"terms": "accepted",
		"age":   "gt:17",
	})
	if !fail.Fails() {
		t.Fatal("expected failure")
	}
}

func TestExtendCustomRule(t *testing.T) {
	validation.Extend("uppercase", func(v *validation.Validator, field, value, param string) bool {
		return value == "" || value == stringsToUpper(value)
	})
	v := validation.Make(map[string]string{"name": "OK"}, map[string]string{"name": "uppercase"})
	if v.Fails() {
		t.Fatalf("expected pass, got %#v", v.Errors())
	}
}

func TestSometimesExcludeDistinctRequiredVariants(t *testing.T) {
	ok := validation.Make(map[string]string{
		"type":   "user",
		"tags":   "a,b,c",
		"email":  "a@b.c",
		"phone":  "",
		"city":   "Istanbul",
		"street": "Main",
	}, map[string]string{
		"nickname": "sometimes|min:3",
		"secret":   "exclude_if:type,user|required|min:8",
		"tags":     "distinct",
		"phone":    "required_without:email",
		"street":   "required_with_all:city,email",
		"backup":   "required_without_all:email,phone",
	})
	if ok.Fails() {
		t.Fatalf("expected pass %#v", ok.Errors())
	}
	data, err := ok.Validated()
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := data["secret"]; exists {
		t.Fatalf("secret should be excluded: %#v", data)
	}

	fail := validation.Make(map[string]string{
		"type":  "admin",
		"tags":  "a,a",
		"city":  "Istanbul",
		"email": "a@b.c",
	}, map[string]string{
		"secret": "exclude_if:type,user|required|min:8",
		"tags":   "distinct",
		"street": "required_with_all:city,email",
		"note":   "sometimes|email",
	})
	if !fail.Fails() {
		t.Fatal("expected failure")
	}
	if !fail.Errors().Has("secret") || !fail.Errors().Has("tags") || !fail.Errors().Has("street") {
		t.Fatalf("unexpected errors %#v", fail.Errors())
	}
	// note is present empty? not in data - sometimes skips; add present note that fails email
	failNote := validation.Make(map[string]string{"note": "not-an-email"}, map[string]string{
		"note": "sometimes|email",
	})
	if !failNote.Fails() {
		t.Fatal("sometimes should still validate when present")
	}
}

func stringsToUpper(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			c -= 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func TestAcceptedDeclinedIfMultipleOfDecimal(t *testing.T) {
	ok := validation.Make(map[string]string{
		"subscribe": "yes",
		"terms":     "true",
		"guest":     "1",
		"opt_out":   "false",
		"qty":       "10",
		"price":     "9.99",
		"amount":    "4",
	}, map[string]string{
		"terms":   "accepted_if:subscribe,yes",
		"opt_out": "declined_if:guest,1",
		"qty":     "multiple_of:5",
		"price":   "decimal:2",
		"amount":  "decimal:0,2",
	})
	if ok.Fails() {
		t.Fatalf("expected pass %#v", ok.Errors())
	}

	// condition not met → accepted_if / declined_if pass without requiring value
	skip := validation.Make(map[string]string{
		"subscribe": "no",
		"terms":     "no",
		"guest":     "0",
		"opt_out":   "yes",
	}, map[string]string{
		"terms":   "accepted_if:subscribe,yes",
		"opt_out": "declined_if:guest,1",
	})
	if skip.Fails() {
		t.Fatalf("expected skip pass %#v", skip.Errors())
	}

	fail := validation.Make(map[string]string{
		"subscribe": "yes",
		"terms":     "no",
		"guest":     "1",
		"opt_out":   "yes",
		"qty":       "7",
		"price":     "9.9",
		"amount":    "1.234",
	}, map[string]string{
		"terms":   "accepted_if:subscribe,yes",
		"opt_out": "declined_if:guest,1",
		"qty":     "multiple_of:5",
		"price":   "decimal:2",
		"amount":  "decimal:0,2",
	})
	if !fail.Fails() {
		t.Fatal("expected failure")
	}
	for _, field := range []string{"terms", "opt_out", "qty", "price", "amount"} {
		if !fail.Errors().Has(field) {
			t.Fatalf("expected error on %s: %#v", field, fail.Errors())
		}
	}
}

func TestProhibitsRequiredIfAcceptedUnless(t *testing.T) {
	ok := validation.Make(map[string]string{
		"is_company": "1",
		"company":    "Acme",
		"is_guest":   "0",
		"email":      "a@b.c",
		"mode":       "create",
		"terms":      "yes",
		"newsletter": "1",
	}, map[string]string{
		"company": "required_if_accepted:is_company",
		"email":   "required_if_declined:is_guest",
		"terms":   "accepted_unless:mode,preview",
		"phone":   "prohibits:email,sms",
	})
	if ok.Fails() {
		t.Fatalf("expected pass %#v", ok.Errors())
	}

	skip := validation.Make(map[string]string{
		"is_company": "0",
		"is_guest":   "1",
		"mode":       "preview",
		"terms":      "no",
	}, map[string]string{
		"company": "required_if_accepted:is_company",
		"email":   "required_if_declined:is_guest",
		"terms":   "accepted_unless:mode,preview",
	})
	if skip.Fails() {
		t.Fatalf("expected skip pass %#v", skip.Errors())
	}

	fail := validation.Make(map[string]string{
		"is_company": "yes",
		"is_guest":   "false",
		"mode":       "create",
		"terms":      "no",
		"newsletter": "1",
		"email":      "a@b.c",
	}, map[string]string{
		"company":    "required_if_accepted:is_company",
		"email":      "required_if_declined:is_guest",
		"terms":      "accepted_unless:mode,preview",
		"newsletter": "prohibits:email",
	})
	if !fail.Fails() {
		t.Fatal("expected failure")
	}
	for _, field := range []string{"company", "terms", "newsletter"} {
		if !fail.Errors().Has(field) {
			t.Fatalf("expected error on %s: %#v", field, fail.Errors())
		}
	}

	failEmail := validation.Make(map[string]string{
		"is_guest": "false",
	}, map[string]string{
		"email": "required_if_declined:is_guest",
	})
	if !failEmail.Fails() || !failEmail.Errors().Has("email") {
		t.Fatalf("required_if_declined should fail %#v", failEmail.Errors())
	}
}

func TestContainsDoesntContainNotRegexAlphaSpaces(t *testing.T) {
	ok := validation.Make(map[string]string{
		"bio":  "hello zatrano with go",
		"name": "Ada Lovelace",
		"code": "abc-1",
	}, map[string]string{
		"bio":  "contains:zatrano,go|doesnt_contain:spam",
		"name": "alpha_spaces",
		"code": "not_regex:^[0-9]+$",
	})
	if ok.Fails() {
		t.Fatalf("expected pass %#v", ok.Errors())
	}

	fail := validation.Make(map[string]string{
		"bio":  "hello spam",
		"name": "Ada2",
		"code": "12345",
	}, map[string]string{
		"bio":  "contains:zatrano|doesnt_contain:spam",
		"name": "alpha_spaces",
		"code": "not_regex:^[0-9]+$",
	})
	if !fail.Fails() {
		t.Fatal("expected failure")
	}
	for _, field := range []string{"bio", "name", "code"} {
		if !fail.Errors().Has(field) {
			t.Fatalf("expected error on %s: %#v", field, fail.Errors())
		}
	}
}

func TestDeclinedUnlessAndInArray(t *testing.T) {
	ok := validation.Make(map[string]string{
		"mode":    "live",
		"opt_out": "0",
		"roles":   "admin,editor,viewer",
		"role":    "editor",
		"blocked": "guest",
	}, map[string]string{
		"opt_out": "declined_unless:mode,preview",
		"role":    "in_array:roles",
		"blocked": "not_in_array:roles",
	})
	if ok.Fails() {
		t.Fatalf("expected pass %#v", ok.Errors())
	}

	skip := validation.Make(map[string]string{
		"mode":    "preview",
		"opt_out": "yes",
	}, map[string]string{
		"opt_out": "declined_unless:mode,preview",
	})
	if skip.Fails() {
		t.Fatalf("expected skip pass %#v", skip.Errors())
	}

	fail := validation.Make(map[string]string{
		"mode":    "live",
		"opt_out": "yes",
		"roles":   "admin,editor",
		"role":    "guest",
		"blocked": "admin",
	}, map[string]string{
		"opt_out": "declined_unless:mode,preview",
		"role":    "in_array:roles",
		"blocked": "not_in_array:roles",
	})
	if !fail.Fails() {
		t.Fatal("expected failure")
	}
	for _, field := range []string{"opt_out", "role", "blocked"} {
		if !fail.Errors().Has(field) {
			t.Fatalf("expected error on %s: %#v", field, fail.Errors())
		}
	}
}

func TestExtensionsAndSemver(t *testing.T) {
	ok := validation.Make(map[string]string{
		"file":    "report.PDF",
		"version": "v1.2.3-beta",
	}, map[string]string{
		"file":    "extensions:png,jpg,pdf",
		"version": "semver",
	})
	if ok.Fails() {
		t.Fatalf("expected pass %#v", ok.Errors())
	}
	fail := validation.Make(map[string]string{
		"file":    "notes.txt",
		"version": "1.2",
	}, map[string]string{
		"file":    "extensions:png,jpg,pdf",
		"version": "semver",
	})
	if !fail.Fails() || !fail.Errors().Has("file") || !fail.Errors().Has("version") {
		t.Fatalf("expected failures %#v", fail.Errors())
	}
}

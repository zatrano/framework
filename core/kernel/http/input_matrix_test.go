package http

import (
	"bytes"
	"mime/multipart"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func trimEmpty(key, value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" && key != "keep" {
		return "", false
	}
	return value, true
}

func TestInputThenSetBodyMatchesV301(t *testing.T) {
	later := testRequest(stdhttp.MethodGet, "/", nil)
	later.TransformInputs(trimEmpty)
	_ = later.Input("missing")
	later.SetBody([]byte("name=%20Ada%20"))
	later.SetHeader("Content-Type", "application/x-www-form-urlencoded")
	if got := later.Input("name"); got != " Ada " {
		t.Fatalf("input after SetBody=%q", got)
	}
}

func TestAccessorMatrix(t *testing.T) {
	form := httptest.NewRequest(stdhttp.MethodPost, "/?q=%20raw%20&multi=1&multi=2", strings.NewReader("name=%20Ada%20&note=&keep=%20%20&multi=9"))
	form.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req := RequestFromHTTP(form)
	req.TransformInputs(trimEmpty)

	if got := req.Query("q"); got != " raw " {
		t.Fatalf("query q=%q", got)
	}
	if got := req.Input("name"); got != "Ada" {
		t.Fatalf("input name=%q", got)
	}
	all := req.All()
	if all["name"] != "Ada" {
		t.Fatalf("name=%q", all["name"])
	}
	if _, ok := all["note"]; ok {
		t.Fatalf("note present: %#v", all)
	}
	if all["keep"] != "" {
		t.Fatalf("keep=%q", all["keep"])
	}
	if all["multi"] != "1" {
		t.Fatalf("multi=%q", all["multi"])
	}
	only := req.Only("name")
	if only["name"] != "Ada" || len(only) != 1 {
		t.Fatalf("only=%#v", only)
	}
	except := req.Except("name")
	if _, ok := except["name"]; ok {
		t.Fatalf("except kept name: %#v", except)
	}
	body, err := req.Body()
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "name=%20Ada%20&note=&keep=%20%20&multi=9" {
		t.Fatalf("body=%q", body)
	}

	rawJSON := httptest.NewRequest(stdhttp.MethodPost, "/", strings.NewReader(`{"user":{"name":"  Ada  "},"note":"","tags":[" a ","b"]}`))
	rawJSON.Header.Set("Content-Type", "application/json")
	jreq := RequestFromHTTP(rawJSON)
	jreq.TransformInputs(trimEmpty)
	var dest map[string]any
	if err := jreq.JSON(&dest); err != nil {
		t.Fatal(err)
	}
	user := dest["user"].(map[string]any)
	if user["name"] != "  Ada  " {
		t.Fatalf("json dest=%#v", dest)
	}
	jall := jreq.All()
	if jall["user.name"] != "Ada" {
		t.Fatalf("nested=%q all=%#v", jall["user.name"], jall)
	}
	if _, ok := jall["note"]; ok {
		t.Fatalf("json note present: %#v", jall)
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if err := w.WriteField("title", "  Ada  "); err != nil {
		t.Fatal(err)
	}
	fw, err := w.CreateFormFile("doc", "notes.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write([]byte("file-bytes")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	mreq := testRequest(stdhttp.MethodPost, "/", buf.Bytes())
	mreq.SetHeader("Content-Type", w.FormDataContentType())
	mreq.TransformInputs(trimEmpty)
	if mreq.All()["title"] != "Ada" {
		t.Fatalf("file form title=%q", mreq.All()["title"])
	}
	file, err := mreq.File("doc")
	if err != nil {
		t.Fatal(err)
	}
	if file.Header.Filename != "notes.txt" {
		t.Fatalf("filename=%q", file.Header.Filename)
	}

	empty := testRequest(stdhttp.MethodGet, "/", nil)
	empty.TransformInputs(trimEmpty)
	empty.Set("attr", "kept")
	empty.Merge(map[string]string{"name": "  Ada  "})
	if empty.Input("name") != "  Ada  " {
		t.Fatalf("merge after queue=%q", empty.Input("name"))
	}
	if empty.Get("attr") != "kept" {
		t.Fatalf("attr=%v", empty.Get("attr"))
	}

	later := testRequest(stdhttp.MethodGet, "/", nil)
	later.TransformInputs(trimEmpty)
	_ = later.Input("missing")
	later.SetBody([]byte("name=%20Ada%20"))
	later.SetHeader("Content-Type", "application/x-www-form-urlencoded")
	if later.Input("name") != " Ada " {
		t.Fatalf("set body after empty=%q", later.Input("name"))
	}
}

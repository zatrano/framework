package http

import (
	"bytes"
	"mime/multipart"
	"path/filepath"
	"strings"
	"testing"
)

func TestStoreAsPathTraversalRejected(t *testing.T) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile("file", "../../evil.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}

	reader := multipart.NewReader(&buf, mw.Boundary())
	form, err := reader.ReadForm(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer form.RemoveAll()
	fh := form.File["file"][0]
	f := &UploadedFile{Header: fh}

	dir := t.TempDir()
	dest, err := f.StoreAs(dir, fh.Filename)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(dest) != "evil.txt" {
		t.Fatalf("dest base=%q", filepath.Base(dest))
	}
	if !strings.HasPrefix(dest, dir) {
		t.Fatalf("escaped: %s", dest)
	}
}

func TestSanitizeFilenameStripsDirs(t *testing.T) {
	if got := sanitizeFilename(`..\..\shell.php.jpg`); got != "shell.php.jpg" {
		t.Fatalf("got %q", got)
	}
}

func TestMaxUploadClientCannotRaiseCap(t *testing.T) {
	t.Setenv("MAX_UPLOAD_BYTES", "2048")
	if maxUploadBytes() != 2048 {
		t.Fatalf("got %d", maxUploadBytes())
	}
}

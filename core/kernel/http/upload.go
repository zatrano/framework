package http

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/zatrano/framework/v3/core/kernel/env"
	"github.com/zatrano/framework/v3/core/kernel/safepath"
)

const defaultMaxUpload = 32 << 20 // 32 MiB

// UploadedFile wraps a multipart file header with helpers.
type UploadedFile struct {
	Header *multipart.FileHeader
}

// File opens the uploaded file.
func (f *UploadedFile) File() (multipart.File, error) {
	return f.Header.Open()
}

// Name returns the client original filename.
func (f *UploadedFile) Name() string {
	return f.Header.Filename
}

// Size returns the uploaded size in bytes.
func (f *UploadedFile) Size() int64 {
	return f.Header.Size
}

// Extension returns the lowercase file extension including the dot.
func (f *UploadedFile) Extension() string {
	return strings.ToLower(filepath.Ext(f.Header.Filename))
}

// Mime returns the content type when available.
func (f *UploadedFile) Mime() string {
	if f.Header.Header == nil {
		return ""
	}
	return f.Header.Header.Get("Content-Type")
}

// Store saves the upload under directory using a generated name.
func (f *UploadedFile) Store(directory string) (string, error) {
	name := fmt.Sprintf("%d_%s", f.Header.Size, sanitizeFilename(f.Header.Filename))
	return f.StoreAs(directory, name)
}

// StoreAs saves the upload under directory with the given filename.
// Filename is always basenamed; destination must stay under directory.
func (f *UploadedFile) StoreAs(directory, filename string) (string, error) {
	filename = sanitizeFilename(filename)
	if filename == "" || filename == "." || filename == ".." {
		return "", fmt.Errorf("invalid upload filename")
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", err
	}
	destPath, err := safepath.Resolve(directory, filename)
	if err != nil {
		return "", err
	}
	src, err := f.Header.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	dest, err := os.Create(destPath)
	if err != nil {
		return "", err
	}
	defer dest.Close()

	if _, err := io.Copy(dest, src); err != nil {
		return "", err
	}
	return destPath, nil
}

// HasFile reports whether a multipart file field exists.
func (r *Request) HasFile(key string) bool {
	_, err := r.File(key)
	return err == nil
}

// File returns an uploaded file for the given form field.
func (r *Request) File(key string) (*UploadedFile, error) {
	if err := r.parseMultipart(); err != nil {
		return nil, err
	}
	if r.multipartForm == nil {
		return nil, fmt.Errorf("no multipart form")
	}
	files := r.multipartForm.File[key]
	if len(files) == 0 {
		return nil, fmt.Errorf("http: no such file")
	}
	return &UploadedFile{Header: files[0]}, nil
}

// Files returns all uploaded files for a form field.
func (r *Request) Files(key string) ([]*UploadedFile, error) {
	if err := r.parseMultipart(); err != nil {
		return nil, err
	}
	if r.multipartForm == nil {
		return nil, fmt.Errorf("no multipart form")
	}
	headers := r.multipartForm.File[key]
	out := make([]*UploadedFile, 0, len(headers))
	for _, header := range headers {
		out = append(out, &UploadedFile{Header: header})
	}
	return out, nil
}

func (r *Request) parseMultipart() error {
	if r == nil {
		return fmt.Errorf("nil request")
	}
	if r.multipartForm != nil {
		return nil
	}
	max := maxUploadBytes()
	// Clients may only lower the limit, never raise it above the server cap.
	if raw := r.Header("X-Max-Upload"); raw != "" {
		if n, err := strconvAtoiSafe(raw); err == nil && n > 0 && n < max {
			max = n
		}
	}
	if r.ctx != nil && !r.bodyOverrideSet {
		mf, err := r.ctx.MultipartForm(int64(max))
		if err == nil && mf != nil {
			r.multipartForm = &multipart.Form{Value: mf.Value, File: mf.File}
			return nil
		}
	}
	return r.parseMultipartFromBody(max)
}

func (r *Request) parseMultipartFromBody(max int) error {
	ct := r.Header("Content-Type")
	mediatype, params, err := mime.ParseMediaType(ct)
	if err != nil || mediatype != "multipart/form-data" {
		return fmt.Errorf("not multipart")
	}
	boundary := params["boundary"]
	if boundary == "" {
		return fmt.Errorf("missing boundary")
	}
	body, err := r.readBody()
	if err != nil {
		return err
	}
	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	form, err := reader.ReadForm(int64(max))
	if err != nil {
		return err
	}
	r.multipartForm = form
	return nil
}

func maxUploadBytes() int {
	if raw := strings.TrimSpace(env.Get("MAX_UPLOAD_BYTES", "")); raw != "" {
		if n, err := strconvAtoiSafe(raw); err == nil && n > 0 {
			return n
		}
	}
	return defaultMaxUpload
}

func sanitizeFilename(name string) string {
	// Treat backslash as a separator on all platforms (Linux CI / Windows uploads).
	name = strings.ReplaceAll(name, `\`, "/")
	name = filepath.Base(name)
	name = strings.ReplaceAll(name, " ", "_")
	return name
}

func strconvAtoiSafe(value string) (int, error) {
	n := 0
	for _, c := range value {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid number")
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

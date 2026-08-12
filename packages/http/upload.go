package http

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
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
func (f *UploadedFile) StoreAs(directory, filename string) (string, error) {
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", err
	}
	src, err := f.Header.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	destPath := filepath.Join(directory, filename)
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
	_, header, err := r.raw.FormFile(key)
	if err != nil {
		return nil, err
	}
	return &UploadedFile{Header: header}, nil
}

// Files returns all uploaded files for a form field.
func (r *Request) Files(key string) ([]*UploadedFile, error) {
	if err := r.parseMultipart(); err != nil {
		return nil, err
	}
	if r.raw.MultipartForm == nil {
		return nil, fmt.Errorf("no multipart form")
	}
	headers := r.raw.MultipartForm.File[key]
	out := make([]*UploadedFile, 0, len(headers))
	for _, header := range headers {
		out = append(out, &UploadedFile{Header: header})
	}
	return out, nil
}

func (r *Request) parseMultipart() error {
	if r.raw.MultipartForm != nil {
		return nil
	}
	max := defaultMaxUpload
	if raw := r.Header("X-Max-Upload"); raw != "" {
		if n, err := strconvAtoiSafe(raw); err == nil && n > 0 {
			max = n
		}
	}
	return r.raw.ParseMultipartForm(int64(max))
}

func sanitizeFilename(name string) string {
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

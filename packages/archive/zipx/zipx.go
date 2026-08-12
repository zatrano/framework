package zipx

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Create writes a zip archive from name->content pairs.
func Create(path string, files map[string][]byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := zip.NewWriter(f)
	for name, content := range files {
		fw, err := w.Create(name)
		if err != nil {
			_ = w.Close()
			return err
		}
		if _, err := fw.Write(content); err != nil {
			_ = w.Close()
			return err
		}
	}
	return w.Close()
}

// Bytes builds an in-memory zip archive.
func Bytes(files map[string][]byte) ([]byte, error) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range files {
		fw, err := w.Create(name)
		if err != nil {
			_ = w.Close()
			return nil, err
		}
		if _, err := fw.Write(content); err != nil {
			_ = w.Close()
			return nil, err
		}
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Extract unpacks a zip archive into dest.
func Extract(path, dest string) ([]string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(r.File))
	for _, f := range r.File {
		target := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return names, err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return names, err
		}
		rc, err := f.Open()
		if err != nil {
			return names, err
		}
		out, err := os.Create(target)
		if err != nil {
			_ = rc.Close()
			return names, err
		}
		_, err = io.Copy(out, rc)
		_ = out.Close()
		_ = rc.Close()
		if err != nil {
			return names, err
		}
		names = append(names, f.Name)
	}
	return names, nil
}

// List returns file names inside a zip.
func List(path string) ([]string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	names := make([]string, 0, len(r.File))
	for _, f := range r.File {
		if !f.FileInfo().IsDir() {
			names = append(names, f.Name)
		}
	}
	if len(names) == 0 && len(r.File) == 0 {
		return nil, fmt.Errorf("zip: empty archive")
	}
	return names, nil
}

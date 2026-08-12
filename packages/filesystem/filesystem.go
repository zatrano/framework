package filesystem

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Disk is a filesystem disk contract.
type Disk interface {
	Exists(path string) bool
	Get(path string) ([]byte, error)
	Put(path string, contents []byte) error
	PutFile(path string, reader io.Reader) error
	Delete(paths ...string) error
	Copy(from, to string) error
	Move(from, to string) error
	Size(path string) (int64, error)
	LastModified(path string) (time.Time, error)
	Path(path string) string
	Files(directory string) ([]string, error)
	Directories(directory string) ([]string, error)
	AllFiles(directory string) ([]string, error)
	DeleteDirectory(directory string) error
	MakeDirectory(path string) error
}

// URLAware is implemented by disks that can build public URLs.
type URLAware interface {
	URL(path string) string
}

// TemporaryURLAware builds expiring URLs.
type TemporaryURLAware interface {
	TemporaryURL(path string, expires time.Duration) (string, error)
}

// LocalDisk stores files on the local filesystem.
type LocalDisk struct {
	root       string
	baseURL    string
	signingKey string
	servePath  string
}

// NewLocalDisk creates a local disk rooted at root.
func NewLocalDisk(root string) (*LocalDisk, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	return &LocalDisk{root: root}, nil
}

// SetBaseURL configures the public URL prefix for this disk.
func (d *LocalDisk) SetBaseURL(base string) {
	d.baseURL = strings.TrimRight(base, "/")
}

// Path returns the absolute path for a relative disk path.
func (d *LocalDisk) Path(path string) string {
	return filepath.Join(d.root, filepath.Clean("/"+path))
}

// URL builds a public URL when a base URL is configured.
func (d *LocalDisk) URL(path string) string {
	path = strings.TrimPrefix(filepath.ToSlash(path), "/")
	if d.baseURL == "" {
		return "/" + path
	}
	return d.baseURL + "/" + path
}

// Exists reports whether a file exists.
func (d *LocalDisk) Exists(path string) bool {
	_, err := os.Stat(d.Path(path))
	return err == nil
}

// Get reads a file.
func (d *LocalDisk) Get(path string) ([]byte, error) {
	return os.ReadFile(d.Path(path))
}

// Put writes a file.
func (d *LocalDisk) Put(path string, contents []byte) error {
	full := d.Path(path)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	return os.WriteFile(full, contents, 0o644)
}

// PutFile writes from a reader.
func (d *LocalDisk) PutFile(path string, reader io.Reader) error {
	contents, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	return d.Put(path, contents)
}

// Delete removes files.
func (d *LocalDisk) Delete(paths ...string) error {
	for _, path := range paths {
		if err := os.Remove(d.Path(path)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// Copy copies a file.
func (d *LocalDisk) Copy(from, to string) error {
	contents, err := d.Get(from)
	if err != nil {
		return err
	}
	return d.Put(to, contents)
}

// Move moves a file.
func (d *LocalDisk) Move(from, to string) error {
	if err := d.Copy(from, to); err != nil {
		return err
	}
	return d.Delete(from)
}

// Size returns file size.
func (d *LocalDisk) Size(path string) (int64, error) {
	info, err := os.Stat(d.Path(path))
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// LastModified returns modification time.
func (d *LocalDisk) LastModified(path string) (time.Time, error) {
	info, err := os.Stat(d.Path(path))
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// Files lists files in a directory (non-recursive).
func (d *LocalDisk) Files(directory string) ([]string, error) {
	entries, err := os.ReadDir(d.Path(directory))
	if err != nil {
		return nil, err
	}
	out := make([]string, 0)
	prefix := strings.Trim(directory, "/\\")
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if prefix != "" && prefix != "." {
			name = filepath.ToSlash(filepath.Join(prefix, name))
		}
		out = append(out, name)
	}
	return out, nil
}

// Directories lists immediate child directories.
func (d *LocalDisk) Directories(directory string) ([]string, error) {
	entries, err := os.ReadDir(d.Path(directory))
	if err != nil {
		return nil, err
	}
	out := make([]string, 0)
	prefix := strings.Trim(directory, "/\\")
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if prefix != "" && prefix != "." {
			name = filepath.ToSlash(filepath.Join(prefix, name))
		}
		out = append(out, name)
	}
	return out, nil
}

// AllFiles lists all files under directory recursively.
func (d *LocalDisk) AllFiles(directory string) ([]string, error) {
	root := d.Path(directory)
	prefix := strings.Trim(directory, "/\\")
	out := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(d.root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if prefix != "" && prefix != "." && !strings.HasPrefix(rel, prefix+"/") && rel != prefix {
			return nil
		}
		out = append(out, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteDirectory removes a directory and its contents.
func (d *LocalDisk) DeleteDirectory(directory string) error {
	path := d.Path(directory)
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("path [%s] is not a directory", directory)
	}
	return os.RemoveAll(path)
}

// MakeDirectory creates a directory.
func (d *LocalDisk) MakeDirectory(path string) error {
	return os.MkdirAll(d.Path(path), 0o755)
}

// Manager resolves filesystem disks.
type Manager struct {
	defaultDisk string
	disks       map[string]Disk
}

// NewManager creates a filesystem manager.
func NewManager(defaultDisk string, disks map[string]Disk) *Manager {
	return &Manager{defaultDisk: defaultDisk, disks: disks}
}

// Disk returns a named disk.
func (m *Manager) Disk(name ...string) Disk {
	disk := m.defaultDisk
	if len(name) > 0 && name[0] != "" {
		disk = name[0]
	}
	return m.disks[disk]
}

// Exists proxies to the default disk.
func (m *Manager) Exists(path string) bool { return m.Disk().Exists(path) }

// Get proxies to the default disk.
func (m *Manager) Get(path string) ([]byte, error) { return m.Disk().Get(path) }

// Put proxies to the default disk.
func (m *Manager) Put(path string, contents []byte) error { return m.Disk().Put(path, contents) }

// Delete proxies to the default disk.
func (m *Manager) Delete(paths ...string) error { return m.Disk().Delete(paths...) }

// Files lists files on the default disk.
func (m *Manager) Files(directory string) ([]string, error) { return m.Disk().Files(directory) }

// Directories lists directories on the default disk.
func (m *Manager) Directories(directory string) ([]string, error) {
	return m.Disk().Directories(directory)
}

// AllFiles lists files recursively on the default disk.
func (m *Manager) AllFiles(directory string) ([]string, error) { return m.Disk().AllFiles(directory) }

// DeleteDirectory removes a directory on the default disk.
func (m *Manager) DeleteDirectory(directory string) error {
	return m.Disk().DeleteDirectory(directory)
}

// Missing reports whether a path does not exist.
func (m *Manager) Missing(path string) bool { return !m.Exists(path) }

// PutString writes a string.
func (m *Manager) PutString(path, contents string) error {
	return m.Put(path, []byte(contents))
}

// GetString reads a string.
func (m *Manager) GetString(path string) (string, error) {
	raw, err := m.Get(path)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// URL builds a public URL for a path on the given disk (default: public).
func (m *Manager) URL(path string, disk ...string) string {
	name := "public"
	if len(disk) > 0 && disk[0] != "" {
		name = disk[0]
	}
	d := m.Disk(name)
	if d == nil {
		return ""
	}
	if aware, ok := d.(URLAware); ok {
		return aware.URL(path)
	}
	return "/" + strings.TrimPrefix(filepath.ToSlash(path), "/")
}

// TemporaryURL builds an expiring URL when supported.
func (m *Manager) TemporaryURL(path string, expires time.Duration, disk ...string) (string, error) {
	name := "s3"
	if len(disk) > 0 && disk[0] != "" {
		name = disk[0]
	}
	d := m.Disk(name)
	if d == nil {
		return "", fmt.Errorf("filesystem disk [%s] is not defined", name)
	}
	if aware, ok := d.(TemporaryURLAware); ok {
		return aware.TemporaryURL(path, expires)
	}
	if aware, ok := d.(URLAware); ok {
		return aware.URL(path), nil
	}
	return "", fmt.Errorf("disk [%s] does not support temporary URLs", name)
}

// EnsureDisk reports a helpful error when a disk is missing.
func (m *Manager) EnsureDisk(name string) (Disk, error) {
	disk := m.Disk(name)
	if disk == nil {
		return nil, fmt.Errorf("filesystem disk [%s] is not defined", name)
	}
	return disk, nil
}

// Extend registers or replaces a disk.
func (m *Manager) Extend(name string, disk Disk) {
	if m.disks == nil {
		m.disks = map[string]Disk{}
	}
	m.disks[name] = disk
}

// CloudDisk is an in-memory S3-like disk with public and temporary URLs.
type CloudDisk struct {
	mu      sync.RWMutex
	files   map[string][]byte
	times   map[string]time.Time
	baseURL string
	bucket  string
}

// NewCloudDisk creates an in-memory cloud disk.
func NewCloudDisk(bucket, baseURL string) *CloudDisk {
	return &CloudDisk{
		files:   make(map[string][]byte),
		times:   make(map[string]time.Time),
		baseURL: strings.TrimRight(baseURL, "/"),
		bucket:  bucket,
	}
}

func (d *CloudDisk) key(path string) string {
	return strings.TrimPrefix(filepath.ToSlash(path), "/")
}

// Exists reports whether an object exists.
func (d *CloudDisk) Exists(path string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	_, ok := d.files[d.key(path)]
	return ok
}

// Get reads an object.
func (d *CloudDisk) Get(path string) ([]byte, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	raw, ok := d.files[d.key(path)]
	if !ok {
		return nil, os.ErrNotExist
	}
	out := make([]byte, len(raw))
	copy(out, raw)
	return out, nil
}

// Put writes an object.
func (d *CloudDisk) Put(path string, contents []byte) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	key := d.key(path)
	buf := make([]byte, len(contents))
	copy(buf, contents)
	d.files[key] = buf
	d.times[key] = time.Now()
	return nil
}

// PutFile writes from a reader.
func (d *CloudDisk) PutFile(path string, reader io.Reader) error {
	contents, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	return d.Put(path, contents)
}

// Delete removes objects.
func (d *CloudDisk) Delete(paths ...string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, path := range paths {
		key := d.key(path)
		delete(d.files, key)
		delete(d.times, key)
	}
	return nil
}

// Copy copies an object.
func (d *CloudDisk) Copy(from, to string) error {
	raw, err := d.Get(from)
	if err != nil {
		return err
	}
	return d.Put(to, raw)
}

// Move moves an object.
func (d *CloudDisk) Move(from, to string) error {
	if err := d.Copy(from, to); err != nil {
		return err
	}
	return d.Delete(from)
}

// Size returns object size.
func (d *CloudDisk) Size(path string) (int64, error) {
	raw, err := d.Get(path)
	if err != nil {
		return 0, err
	}
	return int64(len(raw)), nil
}

// LastModified returns modification time.
func (d *CloudDisk) LastModified(path string) (time.Time, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	ts, ok := d.times[d.key(path)]
	if !ok {
		return time.Time{}, os.ErrNotExist
	}
	return ts, nil
}

// Path returns a logical cloud path.
func (d *CloudDisk) Path(path string) string {
	return "s3://" + d.bucket + "/" + d.key(path)
}

// Files lists object keys under a prefix (non-recursive).
func (d *CloudDisk) Files(directory string) ([]string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	prefix := d.key(directory)
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	out := make([]string, 0)
	for key := range d.files {
		if prefix == "" || strings.HasPrefix(key, prefix) {
			rest := strings.TrimPrefix(key, prefix)
			if rest == "" || strings.Contains(rest, "/") {
				continue
			}
			out = append(out, key)
		}
	}
	return out, nil
}

// Directories lists immediate child directory prefixes.
func (d *CloudDisk) Directories(directory string) ([]string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	prefix := d.key(directory)
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	seen := map[string]bool{}
	out := make([]string, 0)
	for key := range d.files {
		if prefix != "" && !strings.HasPrefix(key, prefix) {
			continue
		}
		rest := strings.TrimPrefix(key, prefix)
		if rest == "" || !strings.Contains(rest, "/") {
			continue
		}
		dir := strings.SplitN(rest, "/", 2)[0]
		full := dir
		if prefix != "" {
			full = strings.TrimSuffix(prefix, "/") + "/" + dir
		}
		if !seen[full] {
			seen[full] = true
			out = append(out, full)
		}
	}
	return out, nil
}

// AllFiles lists all object keys under a prefix recursively.
func (d *CloudDisk) AllFiles(directory string) ([]string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	prefix := d.key(directory)
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	out := make([]string, 0)
	for key := range d.files {
		if prefix == "" || strings.HasPrefix(key, prefix) || key == strings.TrimSuffix(prefix, "/") {
			out = append(out, key)
		}
	}
	return out, nil
}

// DeleteDirectory removes all objects under a directory prefix.
func (d *CloudDisk) DeleteDirectory(directory string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	prefix := d.key(directory)
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	for key := range d.files {
		if prefix == "" || strings.HasPrefix(key, prefix) || key == strings.TrimSuffix(prefix, "/") {
			delete(d.files, key)
			delete(d.times, key)
		}
	}
	return nil
}

// MakeDirectory is a no-op for cloud disks.
func (d *CloudDisk) MakeDirectory(path string) error {
	return nil
}

// URL builds a public object URL.
func (d *CloudDisk) URL(path string) string {
	return d.baseURL + "/" + d.bucket + "/" + d.key(path)
}

// TemporaryURL builds a signed-looking temporary URL.
func (d *CloudDisk) TemporaryURL(path string, expires time.Duration) (string, error) {
	expiry := time.Now().Add(expires).Unix()
	return fmt.Sprintf("%s?X-Amz-Expires=%d&X-Amz-Expires-At=%d", d.URL(path), int(expires.Seconds()), expiry), nil
}

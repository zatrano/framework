package fileconfig

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/zatrano/framework/configs/envconfig"
	"github.com/zatrano/framework/configs/logconfig"
)

// contentType'ta sadece güvenli karakterlere izin verilir (path traversal önlemi)
var safeContentTypeRe = regexp.MustCompile(`[^a-z0-9_-]`)

type FileConfig struct {
	BasePath      string
	AllowedExtMap map[string][]string
	mu            sync.Mutex
}

var Config *FileConfig

func InitFileConfig() {
	// .env'den oku; boş, satır-içi # yorumu veya hatalı parse (# ile başlama) durumunda ./uploads
	basePath := normalizeFileBasePath(envconfig.String("FILE_BASE_PATH", ""))
	if basePath == "" {
		basePath = "./uploads"
	}

	if err := os.MkdirAll(basePath, 0755); err != nil {
		logconfig.SLog.Fatalw("Upload klasörü oluşturulamadı",
			"path", basePath, "error", err)
	}

	Config = &FileConfig{
		BasePath:      basePath,
		AllowedExtMap: make(map[string][]string),
	}

	logconfig.SLog.Infow("FileConfig initialized", "base_path", basePath)
}

// normalizeFileBasePath, .env satırlarında "KEY=  # açıklama" gibi hatalı parse sonucu
// oluşan "# Boş = ..." değerlerini söküp yolu güvenle döndürür.
func normalizeFileBasePath(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// Aynı satırdaki yorum: "path # note"
	if i := strings.Index(s, " #"); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	if s == "" || strings.HasPrefix(s, "#") {
		return ""
	}
	return s
}

func (fc *FileConfig) GetPath(contentType string) string {
	contentType = sanitize(contentType)
	return filepath.Join(fc.BasePath, contentType)
}

func (fc *FileConfig) GetAllowedExtensions(contentType string) []string {
	contentType = sanitize(contentType)
	fc.mu.Lock()
	defer fc.mu.Unlock()
	return fc.AllowedExtMap[contentType]
}

func (fc *FileConfig) SetAllowedExtensions(contentType string, extensions []string) {
	contentType = sanitize(contentType)
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.AllowedExtMap[contentType] = extensions

	dir := fc.GetPath(contentType)
	if err := os.MkdirAll(dir, 0755); err != nil {
		// 🔧 Düzeltme: Sugared logger kullan
		logconfig.SLog.Fatalw("Klasör oluşturulamadı",
			"dir", dir, "error", err)
	}
}

func (fc *FileConfig) IsExtensionAllowed(contentType, ext string) bool {
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	for _, allowed := range fc.GetAllowedExtensions(contentType) {
		if allowed == ext {
			return true
		}
	}
	return false
}

// sanitize path traversal ve güvensiz karakterleri engeller; sadece [a-z0-9_-] kalır.
func sanitize(str string) string {
	str = strings.ToLower(strings.TrimSpace(str))
	str = safeContentTypeRe.ReplaceAllString(str, "")
	if str == "" || str == "." || str == ".." {
		return "_invalid"
	}
	return str
}

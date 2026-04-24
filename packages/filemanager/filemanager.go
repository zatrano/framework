package filemanager

import (
	"crypto/rand"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/zatrano/framework/configs/fileconfig"

	_ "golang.org/x/image/webp"

	"github.com/gofiber/fiber/v3"
)

var (
	ErrFileNotProvided        = errors.New("dosya sağlanmadı")
	ErrInvalidFileType        = errors.New("geçersiz dosya türü veya uzantısı")
	ErrFileTooLarge           = errors.New("dosya boyutu çok büyük")
	ErrImageCouldNotBeDecoded = errors.New("resim dosyası çözümlenemedi, format desteklenmiyor olabilir")
	ErrUnsupportedFormat      = errors.New("desteklenmeyen resim formatı")
)

// FailedFile başarısız yüklenen dosyanın adını ve hata sebebini tutar.
type FailedFile struct {
	FileName string
	Err      error
}

const (
	DefaultMaxFileSize     = 30 * 1024 * 1024
	JpegProcessingQuality  = 75
	MimeSniffingBufferSize = 512
)

func UploadFile(c fiber.Ctx, formFieldName, contentType string) (string, error) {
	fileHeader, err := c.FormFile(formFieldName)
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) || strings.Contains(err.Error(), "there is no uploaded file associated with the given key") {
			return "", ErrFileNotProvided
		}
		return "", err
	}

	if err := validateFile(fileHeader, contentType); err != nil {
		return "", err
	}

	file, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("yüklenen dosya açılamadı: %w", err)
	}
	defer file.Close()

	buffer := make([]byte, MimeSniffingBufferSize)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("dosya türü okunurken hata: %w", err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("dosya okuma imleci sıfırlanamadı: %w", err)
	}

	detectedContentType := http.DetectContentType(buffer[:n])

	// Sadece JPEG ve PNG'yi işliyoruz (boyut küçültme/optimizasyon için), diğerlerini olduğu gibi kaydediyoruz
	if detectedContentType == "image/jpeg" || detectedContentType == "image/png" {
		return processAndSaveImage(file, fileHeader.Filename, contentType)
	}

	return saveOriginalFile(c, fileHeader, contentType)
}

func UploadMultipleFiles(c fiber.Ctx, formFieldName, contentType string) ([]string, []FailedFile, error) {
	form, err := c.MultipartForm()
	if err != nil {
		return nil, nil, err
	}

	files := form.File[formFieldName]
	if len(files) == 0 {
		return nil, nil, ErrFileNotProvided
	}

	var results []string
	var failed []FailedFile

	for _, fileHeader := range files {
		if err := validateFile(fileHeader, contentType); err != nil {
			failed = append(failed, FailedFile{FileName: fileHeader.Filename, Err: err})
			continue
		}

		file, err := fileHeader.Open()
		if err != nil {
			failed = append(failed, FailedFile{FileName: fileHeader.Filename, Err: fmt.Errorf("dosya açılamadı: %w", err)})
			continue
		}

		buffer := make([]byte, MimeSniffingBufferSize)
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			file.Close()
			failed = append(failed, FailedFile{FileName: fileHeader.Filename, Err: fmt.Errorf("dosya türü okunamadı: %w", err)})
			continue
		}
		file.Seek(0, io.SeekStart)

		detectedContentType := http.DetectContentType(buffer[:n])
		var fileName string
		var uploadErr error

		if detectedContentType == "image/jpeg" || detectedContentType == "image/png" {
			fileName, uploadErr = processAndSaveImage(file, fileHeader.Filename, contentType)
		} else {
			fileName, uploadErr = saveOriginalFile(c, fileHeader, contentType)
		}
		file.Close()

		if uploadErr != nil {
			failed = append(failed, FailedFile{FileName: fileHeader.Filename, Err: uploadErr})
		} else {
			results = append(results, fileName)
		}
	}

	return results, failed, nil
}

func processAndSaveImage(file multipart.File, originalFilename, contentType string) (string, error) {
	img, format, err := image.Decode(file)
	if err != nil {
		return "", ErrImageCouldNotBeDecoded
	}

	newFileName, err := generateUniqueFileName(originalFilename)
	if err != nil {
		return "", fmt.Errorf("benzersiz dosya adı oluşturulamadı: %w", err)
	}

	destination := filepath.Join(fileconfig.Config.GetPath(contentType), newFileName)
	destFile, err := os.Create(destination)
	if err != nil {
		return "", fmt.Errorf("hedef dosya oluşturulamadı: %w", err)
	}
	defer destFile.Close()

	switch format {
	case "jpeg":
		options := &jpeg.Options{Quality: JpegProcessingQuality}
		if err := jpeg.Encode(destFile, img, options); err != nil {
			os.Remove(destination)
			return "", fmt.Errorf("jpeg dosyası kodlanamadı: %w", err)
		}
	case "png":
		if err := png.Encode(destFile, img); err != nil {
			os.Remove(destination)
			return "", fmt.Errorf("png dosyası kodlanamadı: %w", err)
		}
	default:
		// Desteklenmeyen bir format decode edildiyse (örn: webp decode edildi ama encode desteğimiz yok)
		// Dosyayı kapatıp siliyoruz ve hata döndürüyoruz
		destFile.Close()
		os.Remove(destination)
		return "", ErrUnsupportedFormat
	}

	return newFileName, nil
}

func saveOriginalFile(c fiber.Ctx, fileHeader *multipart.FileHeader, contentType string) (string, error) {
	newFileName, err := generateUniqueFileName(fileHeader.Filename)
	if err != nil {
		return "", fmt.Errorf("benzersiz dosya adı oluşturulamadı: %w", err)
	}
	destination := filepath.Join(fileconfig.Config.GetPath(contentType), newFileName)
	if err := c.SaveFile(fileHeader, destination); err != nil {
		return "", err
	}
	return newFileName, nil
}

func validateFile(file *multipart.FileHeader, contentType string) error {
	if file.Size > DefaultMaxFileSize {
		return ErrFileTooLarge
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !fileconfig.Config.IsExtensionAllowed(contentType, ext) {
		return ErrInvalidFileType
	}
	return nil
}

func generateUniqueFileName(originalName string) (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	randomStr := fmt.Sprintf("%x", b)
	timestamp := fmt.Sprintf("%d", time.Now().UnixNano())
	ext := filepath.Ext(originalName)
	safeBaseName := regexp.MustCompile(`[^a-zA-Z0-9_-]+`).ReplaceAllString(strings.TrimSuffix(originalName, ext), "")
	if safeBaseName == "" {
		safeBaseName = "file"
	}
	return fmt.Sprintf("%s-%s-%s%s", timestamp, randomStr, safeBaseName, ext), nil
}

// safeFileName dosya adında path traversal ve güvensiz karakterleri engeller.
var safeFileNameRe = regexp.MustCompile(`[^a-zA-Z0-9_.-]`)

func DeleteFile(contentType, fileName string) {
	if fileName == "" {
		return
	}
	// Path traversal ve güvensiz karakterleri temizle
	fileName = safeFileNameRe.ReplaceAllString(filepath.Base(fileName), "")
	if fileName == "" || fileName == "." || fileName == ".." {
		return
	}
	go func() {
		const maxRetries = 5
		const retryDelay = 1 * time.Second
		baseDir := fileconfig.Config.GetPath(contentType)
		absolutePath, err := filepath.Abs(filepath.Join(baseDir, fileName))
		if err != nil {
			return
		}
		baseAbs, err := filepath.Abs(baseDir)
		if err != nil {
			return
		}
		// BasePath dışına yazmayı engelle (path traversal)
		if baseAbs != absolutePath && !strings.HasPrefix(absolutePath, baseAbs+string(filepath.Separator)) {
			return
		}
		for i := 0; i < maxRetries; i++ {
			err = os.Remove(absolutePath)
			if err == nil || os.IsNotExist(err) {
				return
			}
			time.Sleep(retryDelay)
		}
	}()
}

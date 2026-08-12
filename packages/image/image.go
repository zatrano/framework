package image

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

// Info describes an image file.
type Info struct {
	Path   string `json:"path"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Format string `json:"format"`
}

// ReadInfo opens an image and returns dimensions/format.
func ReadInfo(path string) (*Info, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	cfg, format, err := image.DecodeConfig(f)
	if err != nil {
		return nil, err
	}
	return &Info{Path: path, Width: cfg.Width, Height: cfg.Height, Format: format}, nil
}

// CreatePNG writes a solid-color PNG (useful for demos/tests).
func CreatePNG(path string, width, height int, c color.Color) error {
	if width <= 0 || height <= 0 {
		return fmt.Errorf("image: invalid dimensions")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: c}, image.Point{}, draw.Src)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// Resize writes a nearest-neighbor resized copy of src to dst.
func Resize(src, dst string, width, height int) error {
	if width <= 0 || height <= 0 {
		return fmt.Errorf("image: invalid dimensions")
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	img, format, err := image.Decode(in)
	if err != nil {
		return err
	}
	dstImg := resizeNearest(img, width, height)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	switch strings.ToLower(filepath.Ext(dst)) {
	case ".jpg", ".jpeg":
		return jpeg.Encode(out, dstImg, &jpeg.Options{Quality: 90})
	default:
		if format == "jpeg" && filepath.Ext(dst) == "" {
			return jpeg.Encode(out, dstImg, &jpeg.Options{Quality: 90})
		}
		return png.Encode(out, dstImg)
	}
}

func resizeNearest(src image.Image, width, height int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			sx := b.Min.X + x*sw/width
			sy := b.Min.Y + y*sh/height
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}

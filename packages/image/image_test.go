package image_test

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"github.com/zatrano/framework/packages/image"
)

func TestImageCreateResize(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.png")
	dst := filepath.Join(dir, "dst.png")
	if err := image.CreatePNG(src, 40, 20, color.RGBA{R: 255, A: 255}); err != nil {
		t.Fatal(err)
	}
	info, err := image.ReadInfo(src)
	if err != nil || info.Width != 40 || info.Height != 20 {
		t.Fatalf("%+v err=%v", info, err)
	}
	if err := image.Resize(src, dst, 10, 5); err != nil {
		t.Fatal(err)
	}
	out, err := image.ReadInfo(dst)
	if err != nil || out.Width != 10 || out.Height != 5 {
		t.Fatalf("%+v err=%v", out, err)
	}
	_ = os.Remove(src)
}

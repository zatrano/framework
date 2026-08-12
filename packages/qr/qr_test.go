package qr_test

import (
	"strings"
	"testing"

	qrcode "github.com/skip2/go-qrcode"

	"github.com/zatrano/framework/packages/qr"
)

func TestQRSVG(t *testing.T) {
	svg := qr.SVG("https://zatrano.test")
	if !strings.Contains(svg, "<svg") || !strings.Contains(svg, "</svg>") {
		t.Fatal(svg)
	}
	if !strings.Contains(svg, `fill="#111"`) {
		t.Fatal("missing modules")
	}
	if !strings.Contains(svg, `<rect x=`) {
		t.Fatal("expected module rects")
	}
}

func TestQRMatrixNonEmpty(t *testing.T) {
	q, err := qrcode.New("https://zatrano.test", qrcode.Medium)
	if err != nil {
		t.Fatal(err)
	}
	bitmap := q.Bitmap()
	if len(bitmap) == 0 {
		t.Fatal("empty matrix")
	}
	black := 0
	for _, row := range bitmap {
		for _, cell := range row {
			if cell {
				black++
			}
		}
	}
	if black == 0 {
		t.Fatal("matrix has no set modules")
	}

	svg := qr.SVG("https://zatrano.test", 128)
	if !strings.Contains(svg, `width="128"`) {
		t.Fatalf("expected pixel width 128: %s", svg[:min(80, len(svg))])
	}
	rects := strings.Count(svg, `<rect x=`)
	if rects == 0 {
		t.Fatal("SVG must contain module rects")
	}
	if rects != black {
		t.Fatalf("rect count %d != black modules %d", rects, black)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

package qr

import (
	"fmt"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

// SVG encodes payload as a scannable QR code and returns an SVG document.
// Optional size is the pixel width/height (default 256).
func SVG(payload string, size ...int) string {
	px := 256
	if len(size) > 0 && size[0] > 0 {
		px = size[0]
	}

	q, err := qrcode.New(payload, qrcode.Medium)
	if err != nil {
		return fmt.Sprintf(
			`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d"><rect width="100%%" height="100%%" fill="#fff"/></svg>`,
			px, px,
		)
	}

	cells := q.Bitmap()
	dim := len(cells)
	if dim == 0 {
		return fmt.Sprintf(
			`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d"><rect width="100%%" height="100%%" fill="#fff"/></svg>`,
			px, px,
		)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" shape-rendering="crispEdges">`,
		px, px, dim, dim,
	))
	b.WriteString(`<rect width="100%" height="100%" fill="#fff"/>`)
	for y := 0; y < dim; y++ {
		row := cells[y]
		for x := 0; x < len(row); x++ {
			if row[x] {
				b.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="1" height="1" fill="#111"/>`, x, y))
			}
		}
	}
	b.WriteString(`</svg>`)
	return b.String()
}

// Payload returns SVG bytes and an image/svg+xml content type for HTTP handlers.
func Payload(payload string, size ...int) (body []byte, contentType string) {
	svg := SVG(payload, size...)
	return []byte(svg), "image/svg+xml; charset=utf-8"
}

package tool

import (
	"bytes"
	"fmt"
	"image/jpeg"

	"github.com/gen2brain/go-fitz"
)

// Render tuning. The long edge is capped well under common vision-model limits
// (Anthropic resizes images above ~1568px), and DPI is clamped so tiny pages
// aren't upscaled into huge images nor large pages rendered illegibly small.
const (
	renderMaxLongEdgePx = 1400.0
	renderMaxDPI        = 150.0
	renderMinDPI        = 50.0
	renderJPEGQuality   = 80
)

// renderPDFPageImage renders a single PDF page (1-based) to a JPEG image,
// sizing it so the long edge stays near renderMaxLongEdgePx. It returns the
// encoded bytes and the media type ("image/jpeg").
func renderPDFPageImage(path string, page int) ([]byte, string, error) {
	doc, err := fitz.New(path)
	if err != nil {
		return nil, "", fmt.Errorf("open pdf for render: %w", err)
	}
	defer doc.Close()

	if page < 1 || page > doc.NumPage() {
		return nil, "", fmt.Errorf("page %d out of range (document has %d pages)", page, doc.NumPage())
	}
	idx := page - 1 // go-fitz uses 0-based page indices

	dpi := renderMaxDPI
	if b, err := doc.Bound(idx); err == nil {
		longPts := b.Dx()
		if b.Dy() > longPts {
			longPts = b.Dy()
		}
		if longPts > 0 {
			longInches := float64(longPts) / 72.0
			dpi = renderMaxLongEdgePx / longInches
			if dpi > renderMaxDPI {
				dpi = renderMaxDPI
			}
			if dpi < renderMinDPI {
				dpi = renderMinDPI
			}
		}
	}

	img, err := doc.ImageDPI(idx, dpi)
	if err != nil {
		return nil, "", fmt.Errorf("render page %d: %w", page, err)
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: renderJPEGQuality}); err != nil {
		return nil, "", fmt.Errorf("encode page %d to jpeg: %w", page, err)
	}
	return buf.Bytes(), "image/jpeg", nil
}

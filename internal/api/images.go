package api

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"

	"golang.org/x/image/draw"
	"golang.org/x/image/webp"
)

// bytesReader wraps a byte slice in an io.Reader (avoids importing bytes in callers).
func bytesReader(b []byte) io.Reader { return bytes.NewReader(b) }

const (
	fullMaxWidth    = 1600    // pixels — compress large uploads
	fullMaxHeight   = 1600
	thumbMaxWidth   = 400
	thumbMaxHeight  = 400
	fullJPEGQuality = 85
	thumbJPEGQuality = 72
)

// processImage decodes an uploaded image, re-encodes a compressed full-size
// version and a thumbnail, both as JPEG. The ext parameter is the original
// file extension (lowercase, with leading dot).
func processImage(r io.Reader, ext string) (full, thumb []byte, err error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, fmt.Errorf("reading image: %w", err)
	}

	src, err := decodeImage(bytes.NewReader(data), ext)
	if err != nil {
		return nil, nil, fmt.Errorf("decoding image: %w", err)
	}

	fullImg := resizeToFit(src, fullMaxWidth, fullMaxHeight)
	full, err = encodeJPEG(fullImg, fullJPEGQuality)
	if err != nil {
		return nil, nil, fmt.Errorf("encoding full image: %w", err)
	}

	thumbImg := resizeToFit(src, thumbMaxWidth, thumbMaxHeight)
	thumb, err = encodeJPEG(thumbImg, thumbJPEGQuality)
	if err != nil {
		return nil, nil, fmt.Errorf("encoding thumbnail: %w", err)
	}

	return full, thumb, nil
}

func decodeImage(r io.Reader, ext string) (image.Image, error) {
	switch ext {
	case ".jpg", ".jpeg":
		return jpeg.Decode(r)
	case ".png":
		return png.Decode(r)
	case ".webp":
		return webp.Decode(r)
	default:
		return nil, fmt.Errorf("unsupported image format: %s", ext)
	}
}

// resizeToFit scales img down to fit within maxW×maxH while preserving aspect
// ratio. If the image already fits, it is returned as-is.
func resizeToFit(src image.Image, maxW, maxH int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()

	if w <= maxW && h <= maxH {
		return src
	}

	// Scale to fit, preserving aspect ratio.
	scaleW := float64(maxW) / float64(w)
	scaleH := float64(maxH) / float64(h)
	scale := scaleW
	if scaleH < scale {
		scale = scaleH
	}

	newW := int(float64(w) * scale)
	newH := int(float64(h) * scale)
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.BiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

func encodeJPEG(img image.Image, quality int) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// thumbPath derives the thumbnail path from an image path by replacing the
// file extension with "-thumb.jpg". For example:
//
//	"images/livestock-1-123.jpg" → "images/livestock-1-123-thumb.jpg"
//	"images/obs-2-456.png"       → "images/obs-2-456-thumb.jpg"
func thumbPath(imagePath string) string {
	// Strip extension and append -thumb.jpg.
	for i := len(imagePath) - 1; i >= 0 && imagePath[i] != '/'; i-- {
		if imagePath[i] == '.' {
			return imagePath[:i] + "-thumb.jpg"
		}
	}
	return imagePath + "-thumb.jpg"
}

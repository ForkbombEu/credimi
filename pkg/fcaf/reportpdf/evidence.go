// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package reportpdf

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	_ "image/gif" // register GIF image decoder
	"image/jpeg"
	_ "image/png" // register PNG image decoder
	"net/url"
	"path"
	"strings"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register WebP image decoder
)

const (
	maxImageBytes  = 50 << 20
	maxImagePixels = 40_000_000

	// pdfImageWidth caps the embedded evidence width. Screenshots render at
	// most 120 mm wide in the document, so wider sources are downscaled to
	// keep the generated PDF well under the pipeline-results file size limit.
	pdfImageWidth  = 700
	pdfJPEGQuality = 75
)

func ReferenceFilename(reference string) string {
	parsed, err := url.Parse(strings.TrimSpace(reference))
	if err != nil {
		return ""
	}
	filename, err := url.PathUnescape(path.Base(parsed.Path))
	if err != nil || filename == "." || filename == "/" {
		return ""
	}
	return filename
}

func PrepareImage(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("image is empty")
	}
	if len(data) > maxImageBytes {
		return nil, fmt.Errorf("image exceeds %d bytes", maxImageBytes)
	}
	configuration, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image configuration: %w", err)
	}
	if configuration.Width <= 0 || configuration.Height <= 0 {
		return nil, fmt.Errorf("image dimensions are invalid")
	}
	if int64(configuration.Width)*int64(configuration.Height) > maxImagePixels {
		return nil, fmt.Errorf(
			"image exceeds %d pixels",
			maxImagePixels,
		)
	}
	decoded, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}
	bounds := decoded.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width > pdfImageWidth {
		scale := float64(pdfImageWidth) / float64(width)
		height = max(1, int(float64(height)*scale))
		width = pdfImageWidth
	}

	// Flatten onto white: JPEG has no alpha channel.
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(image.White), image.Point{}, draw.Src)
	xdraw.CatmullRom.Scale(canvas, canvas.Bounds(), decoded, bounds, xdraw.Over, nil)

	var output bytes.Buffer
	if err := jpeg.Encode(&output, canvas, &jpeg.Options{Quality: pdfJPEGQuality}); err != nil {
		return nil, fmt.Errorf("encode image as JPEG: %w", err)
	}
	return output.Bytes(), nil
}

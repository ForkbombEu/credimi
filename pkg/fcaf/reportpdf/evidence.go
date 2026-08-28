// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package reportpdf

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"  // register GIF image decoder
	_ "image/jpeg" // register JPEG image decoder
	"image/png"
	"net/url"
	"path"
	"sort"
	"strings"

	"github.com/forkbombeu/credimi/pkg/fcaf/engine"
	_ "golang.org/x/image/webp" // register WebP image decoder
)

const (
	maxImageBytes  = 50 << 20
	maxImagePixels = 40_000_000
)

func ImageReferences(report engine.Report) map[string][]string {
	references := make(map[string][]string)
	for key, record := range report.Evidence {
		seen := map[string]struct{}{}
		collectImageReferences(record.Value, seen)
		values := make([]string, 0, len(seen))
		for value := range seen {
			values = append(values, value)
		}
		sort.Strings(values)
		if len(values) > 0 {
			references[key] = values
		}
	}
	return references
}

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
	var output bytes.Buffer
	if err := png.Encode(&output, decoded); err != nil {
		return nil, fmt.Errorf("encode image as PNG: %w", err)
	}
	return output.Bytes(), nil
}

func collectImageReferences(value any, seen map[string]struct{}) {
	switch value := value.(type) {
	case string:
		if isImageReference(value) {
			seen[value] = struct{}{}
		}
	case []any:
		for _, child := range value {
			collectImageReferences(child, seen)
		}
	case []string:
		for _, child := range value {
			collectImageReferences(child, seen)
		}
	case map[string]any:
		for _, child := range value {
			collectImageReferences(child, seen)
		}
	}
}

func isImageReference(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return false
	}
	extension := strings.ToLower(path.Ext(parsed.Path))
	switch extension {
	case ".gif", ".jpeg", ".jpg", ".png", ".webp":
		return true
	default:
		return false
	}
}

// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package logo

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func LogoHooks(app core.App) {
	app.OnRecordCreate().BindFunc(HandleLogo)
	app.OnRecordUpdate().BindFunc(HandleLogo)
}

func HandleLogo(e *core.RecordEvent) error {
	logoURL := e.Record.GetString("logo_url")
	if logoURL == "" {
		return e.Next()
	}

	file, err := DownloadImage(e.Context, logoURL)
	if err != nil {
		log.Printf("ERROR download: %v", err)
		return e.Next()
	}

	e.Record.Set("logo", []*filesystem.File{file})
	return e.Next()
}

func DownloadImage(ctx context.Context, imageURL string) (*filesystem.File, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", imageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read data: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("empty image")
	}

	filename := extractFilenameFromURL(imageURL)
	return filesystem.NewFileFromBytes(data, filename)
}

func extractFilenameFromURL(imageURL string) string {
	parts := strings.Split(imageURL, "/")
	if len(parts) == 0 || parts[len(parts)-1] == "" {
		cleanURL := strings.ReplaceAll(imageURL, "://", "_")
		cleanURL = strings.ReplaceAll(cleanURL, "/", "_")
		cleanURL = strings.ReplaceAll(cleanURL, "?", "_")
		return cleanURL + ".jpg"
	}

	lastPart := parts[len(parts)-1]
	if idx := strings.Index(lastPart, "?"); idx != -1 {
		lastPart = lastPart[:idx]
	}

	if idx := strings.Index(lastPart, "#"); idx != -1 {
		lastPart = lastPart[:idx]
	}

	if !strings.Contains(lastPart, ".") {
		lastPart += ".jpg"
	}

	return lastPart
}

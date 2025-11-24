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

	return filesystem.NewFileFromBytes(data, "image.jpg")
}

// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package logo

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

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

	file, err := downloadWithCustomClient(e.Context, logoURL)
	if err != nil {
		log.Printf("ERROR download: %v", err)
		return e.Next()
	}

	e.Record.Set("logo", []*filesystem.File{file})
	return e.Next()
}

func downloadWithCustomClient(ctx context.Context, urlStr string) (*filesystem.File, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				dialer := &net.Dialer{
					Timeout: 10 * time.Second,
				}
				return dialer.DialContext(ctx, network, addr)
			},
		},
	}

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 "+
			"(KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "image/webp,image/apng,image/*,*/*;q=0.8")
	req.Header.Set("Accept-Language", "it-IT,it;q=0.9,en;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Referer", "https://www.freepik.com/")
	req.Header.Set("Sec-Fetch-Dest", "image")
	req.Header.Set("Sec-Fetch-Mode", "no-cors")
	req.Header.Set("Sec-Fetch-Site", "cross-site")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("empty response")
	}

	ext := ".jpg"
	contentType := resp.Header.Get("Content-Type")
	switch {
	case strings.Contains(contentType, "png"):
		ext = ".png"
	case strings.Contains(contentType, "gif"):
		ext = ".gif"
	case strings.Contains(contentType, "webp"):
		ext = ".webp"
	}

	filename := "logo" + ext
	return filesystem.NewFileFromBytes(data, filename)
}

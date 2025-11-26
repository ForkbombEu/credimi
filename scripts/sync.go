package main

// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func main() {
	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDev:     true,
		DefaultDataDir: "./pb_data",
	})
	if err := app.Bootstrap(); err != nil {
		log.Fatalf("❌ Failed to bootstrap app: %v", err)
	}
	if err := SyncMissingLogos(app); err != nil {
		log.Printf("❌ Logo import failed: %v", err)
		os.Exit(1)
	} else {
		log.Println("✅ Logo import completed successfully!")
		os.Exit(0)
	}
}

func SyncMissingLogos(app core.App) error {
	collections, err := app.FindAllCollections()
	if err != nil {
		return fmt.Errorf("failed to get collections: %w", err)
	}
	for _, collection := range collections {
		hasLogoURL := false
		hasLogoField := false

		for _, field := range collection.Fields.FieldNames() {
			if field == "logo_url" {
				hasLogoURL = true
			}
			if field == "logo" {
				hasLogoField = true
			}
		}

		if !hasLogoURL || !hasLogoField {
			continue
		}

		filter := `logo_url != "" && logo = ""`
		records, err := app.FindRecordsByFilter(collection.Name, filter, "-created", 0, 0)
		if err != nil {
			log.Printf("❌ Error fetching records from %s: %v", collection.Name, err)
			continue
		}

		for _, record := range records {
			logoURL := record.GetString("logo_url")
			if logoURL == "" {
				continue
			}

			if file, err := DownloadImage(context.Background(), logoURL); err == nil {
				record.Set("logo", []*filesystem.File{file})
				app.Save(record)
			} else {
				log.Printf("❌ Download failed, probably invalid url: %v", err)
			}

			if err := app.Save(record); err != nil {
				log.Printf("❌ Failed to save record %s: %v", record.Id, err)
				continue
			}

			time.Sleep(100 * time.Millisecond)
		}
	}

	return nil
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

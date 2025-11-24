// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package logo

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func RunLogoImport(app core.App) {
	if !shouldRunLogoImport() {
		return
	}

	go func() {
		time.Sleep(5 * time.Second)
		if err := SyncMissingLogos(app); err != nil {
			log.Printf("❌ Logo import failed: %v", err)
		} else {
			log.Println("✅ Logo import completed successfully!")
			os.Exit(0)
		}
	}()
}

func shouldRunLogoImport() bool {
	envValue := os.Getenv("RUN_LOGO_IMPORT")
	if envValue == "" {
		return false
	}

	enabled, err := strconv.ParseBool(envValue)
	if err != nil {
		log.Printf("⚠️ Invalid RUN_LOGO_IMPORT value: %s, defaulting to false", envValue)
		return false
	}

	return enabled
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

// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

type CollectionSeed struct {
	Collection string           `json:"collection"`
	Records    []map[string]any `json:"records"`
}

func main() {
	app := pocketbase.New()

	if err := app.Bootstrap(); err != nil {
		log.Fatal("Failed to initialize:", err)
	}

	seedPath := "seeds/data.json"
	if err := seed(app, seedPath); err != nil {
		log.Fatal("Failed to seed:", err)
	}

	fmt.Println("🌱 Seeding complete")
}

func seed(app core.App, jsonPath string) error {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return fmt.Errorf("read seed file: %w", err)
	}

	// Try two formats for compatibility
	var ordered []CollectionSeed
	if err := json.Unmarshal(data, &ordered); err != nil {
		// fallback to legacy map format
		var unordered map[string][]map[string]any
		if err2 := json.Unmarshal(data, &unordered); err2 != nil {
			return fmt.Errorf("invalid seed JSON format: %w", err)
		}
		for name, recs := range unordered {
			ordered = append(ordered, CollectionSeed{Collection: name, Records: recs})
		}
	}

	for _, col := range ordered {
		collection, err := app.FindCollectionByNameOrId(col.Collection)
		if err != nil {
			log.Printf("⚠️ Collection %q not found, skipping...", col.Collection)
			continue
		}

		for _, r := range col.Records {
			record := core.NewRecord(collection)
			record.Load(r)

			if err := app.Save(record); err != nil {
				return fmt.Errorf("failed to populate %s: %w", col.Collection, err)
			}
			log.Printf("✅ Inserted into %s: %v", col.Collection, record.Id)
		}
	}

	return nil
}

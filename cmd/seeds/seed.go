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

func main() {
	app := pocketbase.New()

	if err := app.Bootstrap(); err != nil {
		log.Fatal("Failed to initialize:", err)
	}

	// Seed the database
	seedPath := "seeds/data.json"
	if err := seed(app, seedPath); err != nil {
		log.Fatal("Failed to seed:", err)
	}

	fmt.Println("🌱 Seeding complete")
}

func seed(app core.App, jsonPath string) error {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return err
	}

	var collections map[string][]map[string]any
	if err := json.Unmarshal(data, &collections); err != nil {
		return err
	}

	for collectionName, records := range collections {
		collection, err := app.FindCollectionByNameOrId(collectionName)
		if err != nil {
			log.Printf("⚠️ Collection %q not found, skipping...", collectionName)
			continue
		}

		for _, r := range records {
			record := core.NewRecord(collection)
			record.Load(r)

			if err := app.Save(record); err != nil {
				return fmt.Errorf("failed to populate %s: %w", collectionName, err)
			}
			log.Printf("✅ Inserted into %s: %v", collectionName, record.Id)
		}
	}

	return nil
}

// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package main provides a command-line tool for parsing input strings
// using OpenID4VP and saving the output to files.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/forkbombeu/credimi/pkg/templateengine"
	"github.com/spf13/cobra"
)

// Checks represents a list of check names.
type Checks struct {
	Checks []string `json:"checks"`
}

func main() {
	var input string
	var defaultPath string
	var configPath string
	var outputDir string

	// Define the root command using Cobra
	rootCmd := &cobra.Command{
		Use:   "parse-input",
		Short: "Parses the input check names  and saves output to files",
		Run: func(_ *cobra.Command, _ []string) {
			info, err := os.Stat(outputDir)
			if err != nil {
				fmt.Println("Error: Output directory does not exist:", outputDir)
				return
			}
			if !info.IsDir() {
				fmt.Println("Error: Output path exists but is not a directory:", outputDir)
				return
			}

			var checks Checks
			data, err := os.ReadFile(input)
			if err != nil {
				log.Printf("failed to read %s: %s\n", input, err)
				return
			}
			if err := json.Unmarshal(data, &checks); err != nil {
				log.Printf("failed to parse %s: %s\n", input, err)
				return
			}

			for _, checkString := range checks.Checks {
				cleanPath := filepath.ToSlash(input)
				parts := strings.Split(cleanPath, "/")

				if len(parts) < 3 {
					log.Printf("Skipping invalid path: %s", input)
					continue
				}

				suite := parts[2]

				var result []byte

				switch suite {
				case "openidnet":
					result, err = templateengine.ParseOpenidnetInput(checkString, defaultPath, configPath)
				case "ewc":
					result, err = templateengine.ParseEwcInput(checkString, defaultPath)
				case "eudiw":
					result, err = templateengine.ParseEudiwInput(checkString, defaultPath)
				default:
					log.Printf("Unknown suite '%s' in path %s — skipping", suite, input)
					continue
				}
				if err != nil {
					log.Printf("Error processing %s: %v", checkString, err)
					continue
				}

				filename := fmt.Sprintf("%s.yaml", filepath.Clean(checkString))
				filePath := filepath.Join(outputDir, filename)
				if err := os.WriteFile(filePath, result, 0600); err != nil {
					fmt.Println("Error writing file:", err)
					continue
				}
			}
		},
	}

	// Define the flags for the command
	rootCmd.Flags().StringVarP(&input, "input", "i", "", "Input string (required)")
	rootCmd.Flags().
		StringVarP(&defaultPath, "default", "d", "", "Path to the default JSON file (required)")
	rootCmd.Flags().
		StringVarP(&configPath, "config", "c", "", "Path to the config JSON file (required)")
	rootCmd.Flags().
		StringVarP(&outputDir, "output", "o", "", "Path to the output directory (required)")

	rootCmd.MarkFlagRequired("input")
	rootCmd.MarkFlagRequired("default")
	rootCmd.MarkFlagRequired("output")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

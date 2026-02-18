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

var (
	parseOpenidnetInput = templateengine.ParseOpenidnetInput
	parseEwcInput       = templateengine.ParseEwcInput
	parseEudiwInput     = templateengine.ParseEudiwInput
)

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
			if err := runTemplate(input, defaultPath, configPath, outputDir); err != nil {
				fmt.Println("Error:", err)
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

func runTemplate(input string, defaultPath string, configPath string, outputDir string) error {
	info, err := os.Stat(outputDir)
	if err != nil {
		return fmt.Errorf("output directory does not exist: %s", outputDir)
	}
	if !info.IsDir() {
		return fmt.Errorf("output path exists but is not a directory: %s", outputDir)
	}

	var checks Checks
	data, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", input, err)
	}
	if err := json.Unmarshal(data, &checks); err != nil {
		return fmt.Errorf("failed to parse %s: %w", input, err)
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
			result, err = parseOpenidnetInput(checkString, defaultPath, configPath)
		case "ewc":
			result, err = parseEwcInput(checkString, defaultPath)
		case "eudiw":
			result, err = parseEudiwInput(checkString, defaultPath)
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
			log.Printf("Error writing file: %v", err)
			continue
		}
	}

	return nil
}

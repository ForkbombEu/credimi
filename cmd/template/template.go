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

	"github.com/forkbombeu/credimi/pkg/templateengine"
	"github.com/spf13/cobra"
)

// Variants represents a collection of variant strings.
type Variants struct {
	Variants []string `json:"variants"`
}

func main() {
	var input string
	var defaultPath string
	var configPath string
	var outputDir string

	// Define the root command using Cobra
	rootCmd := &cobra.Command{
		Use:   "parse-input",
		Short: "Parses the input string using OpenID4VP and saves output to files",
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

			var variants Variants
			data, err := os.ReadFile(input)
			if err != nil {
				log.Printf("failed to read %s: %s\n", input, err)
				return
			}
			if err := json.Unmarshal(data, &variants); err != nil {
				log.Printf("failed to parse %s: %s\n", input, err)
				return
			}

			for _, variantString := range variants.Variants {
				result, err := templateengine.ParseInput(variantString, defaultPath, configPath)
				if err != nil {
					fmt.Println("Error processing variant:", err)
					continue
				}

				filename := fmt.Sprintf("%s.yaml", filepath.Clean(variantString))
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
	rootCmd.MarkFlagRequired("config")
	rootCmd.MarkFlagRequired("output")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

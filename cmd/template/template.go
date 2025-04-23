// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/forkbombeu/didimo/pkg/template_engine"
	"github.com/spf13/cobra"
)

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
		Run: func(cmd *cobra.Command, args []string) {
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
			if err := template_engine.LoadJSON(input, &variants); err != nil {
				fmt.Println("Error loading JSON:", err)
				return
			}

			for _, variantString := range variants.Variants {
				result, err := template_engine.ParseInput(variantString, defaultPath, configPath)
				if err != nil {
					fmt.Println("Error processing variant:", err)
					continue
				}

				output, _ := json.MarshalIndent(result, "", "    ")

				output = []byte(fmt.Sprintf(strings.ReplaceAll(string(output), "\\\"", "\"")))
				output = []byte(fmt.Sprintf(strings.ReplaceAll(string(output), "\\\\", "\\")))

				filename := fmt.Sprintf("%s.json", filepath.Clean(variantString))
				filePath := filepath.Join(outputDir, filename)

				if err := os.WriteFile(filePath, output, 0644); err != nil {
					fmt.Println("Error writing file:", err)
					continue
				}

			}
		},
	}

	// Define the flags for the command
	rootCmd.Flags().StringVarP(&input, "input", "i", "", "Input string (required)")
	rootCmd.Flags().StringVarP(&defaultPath, "default", "d", "", "Path to the default JSON file (required)")
	rootCmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to the config JSON file (required)")
	rootCmd.Flags().StringVarP(&outputDir, "output", "o", "", "Path to the output directory (required)")

	rootCmd.MarkFlagRequired("input")
	rootCmd.MarkFlagRequired("default")
	rootCmd.MarkFlagRequired("config")
	rootCmd.MarkFlagRequired("output")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

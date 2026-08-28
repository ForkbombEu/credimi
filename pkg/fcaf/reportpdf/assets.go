// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package reportpdf renders FCAF conformance assessment reports as PDF documents.
package reportpdf

import _ "embed"

var (
	//go:embed assets/Inter-Regular.ttf
	interRegular []byte
	//go:embed assets/Inter-Medium.ttf
	interMedium []byte
	//go:embed assets/Inter-Bold.ttf
	interBold []byte
	//go:embed assets/SourceCodePro-Regular.ttf
	sourceCodeProRegular []byte
	//go:embed assets/SourceCodePro-Semibold.ttf
	sourceCodeProSemibold []byte
	//go:embed assets/credimi_logo-transp.png
	credimiWordmark []byte
)

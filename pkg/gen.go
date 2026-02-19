// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package pkg : this file is a utility file for all the go:generate commands.
package pkg

//go:generate go run ../cmd/template/template.go -c ../schemas/openidnet/wallet/1.0/config_wallet.yaml -d ../schemas/openidnet/wallet/1.0/default_wallet.yaml -i ../schemas/openidnet/wallet/1.0/variants.json -o ../config_templates/openid4vp_wallet/1.0/openid_conformance_suite/
//go:generate go run ../cmd/template/template.go -c ../schemas/openidnet/wallet/draft-24/config_wallet.yaml -d ../schemas/openidnet/wallet/draft-24/default_wallet.yaml -i ../schemas/openidnet/wallet/draft-24/variants.json -o ../config_templates/openid4vp_wallet/draft-24/openid_conformance_suite/
//go:generate go run ../cmd/template/template.go -d ../schemas/ewc/default_template.yaml -i ../schemas/ewc/openid4vci_wallet_draft_15/checks.json -o ../config_templates/openid4vci_wallet/draft-15/ewc/
//go:generate go run ../cmd/template/template.go -d ../schemas/ewc/default_template.yaml -i ../schemas/ewc/openid4vp_wallet_draft_23/checks.json -o ../config_templates/openid4vp_wallet/draft-23/ewc/
//go:generate go run ../cmd/template/template.go -d ../schemas/webuild/default_template.yaml -i ../schemas/webuild/openid4vci_wallet_1.0/checks.json -o ../config_templates/openid4vci_wallet/1.0/webuild/
//go:generate go run ../cmd/template/template.go -d ../schemas/webuild/default_template.yaml -i ../schemas/webuild/openid4vp_wallet_1.0/checks.json -o ../config_templates/openid4vp_wallet/1.0/webuild/
//go:generate go run ../cmd/template/template.go -d ../schemas/eudiw/default_template.yaml -i ../schemas/eudiw/checks.json -o ../config_templates/openid4vp_verifier/draft-23/eudiw/
//go:generate go run generate_client/generate_client.go
//go:generate sh -c "cd .. && go run main.go pipeline schema -o schemas/pipeline/pipeline_schema.json"

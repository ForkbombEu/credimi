// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// This file is a utility file for all the go:generate commands.
package pkg

//go:generate go run ../cmd/template/template.go -c ../schemas/OpenID4VP_Wallet/openID_Foundation-config_wallet.json -d ../schemas/OpenID4VP_Wallet/openID_Foundation-default_wallet.json -i ../schemas/OpenID4VP_Wallet/openID_Foundation-variant_config.json -o ../config_templates/OpenID4VP_Wallet/OpenID_Foundation/
//go:generate go run github.com/atombender/go-jsonschema@v0.18.0 -p credentials_config ../schemas/openid-credential-issuer.schema.json -o workflow_engine/workflows/credentials_config/openid-credential-issuer.schema.go

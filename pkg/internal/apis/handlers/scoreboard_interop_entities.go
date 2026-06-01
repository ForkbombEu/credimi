// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"fmt"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/pocketbase/core"
)

func interopEntityPath(app core.App, record *core.Record, collection string) (string, error) {
	tpl, ok := canonify.CanonifyPaths[collection]
	if !ok {
		return "", fmt.Errorf("no canonify path for collection %s", collection)
	}
	return canonify.BuildPath(app, record, tpl, "")
}

func walletBuildEntity(
	app core.App,
	axisRecord *core.Record,
	cacheRecord *core.Record,
) (InteropMatrixEntity, error) {
	path, err := interopEntityPath(app, axisRecord, "wallets")
	if err != nil {
		return InteropMatrixEntity{}, err
	}

	entity := InteropMatrixEntity{
		ID:        axisRecord.Id,
		Name:      axisRecord.GetString("name"),
		Path:      path,
		AvatarURL: firstNonEmptyStringPtr(
			axisRecord.GetString("avatar"),
			axisRecord.GetString("logo"),
			axisRecord.GetString("logo_url"),
		),
	}

	if cacheRecord != nil {
		versionIDs := cacheRecord.GetStringSlice("wallet_versions")
		versionsByID, err := findRecordsByIDs(app, "wallet_versions", versionIDs)
		if err != nil {
			return InteropMatrixEntity{}, err
		}
		entity.VersionLabel = walletVersionLabelFromCacheRecord(cacheRecord, axisRecord.Id, versionsByID)
	}

	return entity, nil
}

func credentialIssuerBuildEntity(
	app core.App,
	axisRecord *core.Record,
	_ *core.Record,
) (InteropMatrixEntity, error) {
	return simpleInteropBuildEntity(app, axisRecord, "credential_issuers", "avatar", "logo", "logo_url")
}

func verifierBuildEntity(
	app core.App,
	axisRecord *core.Record,
	_ *core.Record,
) (InteropMatrixEntity, error) {
	return simpleInteropBuildEntity(app, axisRecord, "verifiers", "avatar", "logo")
}

func credentialBuildEntity(
	app core.App,
	axisRecord *core.Record,
	_ *core.Record,
) (InteropMatrixEntity, error) {
	path, err := interopEntityPath(app, axisRecord, "credentials")
	if err != nil {
		return InteropMatrixEntity{}, err
	}

	var issuerName *string
	var issuerAvatarURL *string
	issuerID := strings.TrimSpace(axisRecord.GetString("credential_issuer"))
	if issuerID != "" {
		issuersByID, err := findRecordsByIDs(app, "credential_issuers", []string{issuerID})
		if err != nil {
			return InteropMatrixEntity{}, err
		}
		if issuerRecord := issuersByID[issuerID]; issuerRecord != nil {
			issuerName = optionalTrimmedStringPtr(issuerRecord.GetString("name"))
			issuerAvatarURL = firstNonEmptyStringPtr(
				issuerRecord.GetString("avatar"),
				issuerRecord.GetString("logo_url"),
			)
		}
	}

	return buildEnrichedEntityMetadata(
		axisRecord.Id,
		axisRecord.GetString("name"),
		path,
		firstNonEmptyStringPtr(axisRecord.GetString("avatar"), axisRecord.GetString("logo_url")),
		issuerName,
		issuerAvatarURL,
	), nil
}

func useCaseVerificationBuildEntity(
	app core.App,
	axisRecord *core.Record,
	_ *core.Record,
) (InteropMatrixEntity, error) {
	path, err := interopEntityPath(app, axisRecord, "use_cases_verifications")
	if err != nil {
		return InteropMatrixEntity{}, err
	}

	var verifierName *string
	var verifierLogoURL *string
	verifierID := strings.TrimSpace(axisRecord.GetString("verifier"))
	if verifierID != "" {
		verifiersByID, err := findRecordsByIDs(app, "verifiers", []string{verifierID})
		if err != nil {
			return InteropMatrixEntity{}, err
		}
		if verifierRecord := verifiersByID[verifierID]; verifierRecord != nil {
			verifierName = optionalTrimmedStringPtr(verifierRecord.GetString("name"))
			verifierLogoURL = firstNonEmptyStringPtr(
				verifierRecord.GetString("avatar"),
				verifierRecord.GetString("logo"),
			)
		}
	}

	return buildEnrichedEntityMetadata(
		axisRecord.Id,
		axisRecord.GetString("name"),
		path,
		firstNonEmptyStringPtr(axisRecord.GetString("avatar"), axisRecord.GetString("logo")),
		verifierName,
		verifierLogoURL,
	), nil
}

func simpleInteropBuildEntity(
	app core.App,
	record *core.Record,
	collection string,
	avatarFields ...string,
) (InteropMatrixEntity, error) {
	path, err := interopEntityPath(app, record, collection)
	if err != nil {
		return InteropMatrixEntity{}, err
	}

	avatarValues := make([]string, len(avatarFields))
	for i, field := range avatarFields {
		avatarValues[i] = record.GetString(field)
	}

	return InteropMatrixEntity{
		ID:        record.Id,
		Name:      record.GetString("name"),
		Path:      path,
		AvatarURL: firstNonEmptyStringPtr(avatarValues...),
	}, nil
}

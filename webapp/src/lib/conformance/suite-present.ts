// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { entities, type EntityData } from '$lib/global/entities';

import type { ConformanceSuiteRecord } from './record';

/** Humanize a path UID for display when authored metadata is not on the catalog row. */
export function displayNameFromUid(uid: string): string {
	return uid
		.split(/[_-]+/)
		.filter(Boolean)
		.map((part) => part.charAt(0).toUpperCase() + part.slice(1))
		.join(' ');
}

/** Prefer authored `suite_name`; fall back to humanized suite uid. */
export function suiteTitle(
	suite: Pick<ConformanceSuiteRecord, 'suite_name' | 'suite'>
): string {
	const name = suite.suite_name?.trim();
	if (name) return name;
	return displayNameFromUid(suite.suite);
}

/** Trimmed suite subtitle, or `undefined` when absent (hub table). */
export function suiteSubtitle(
	suite: Pick<ConformanceSuiteRecord, 'suite_subtitle'>
): string | undefined {
	const subtitle = suite.suite_subtitle?.trim();
	return subtitle || undefined;
}

/** Trimmed suite logo URL, or `undefined` when absent. */
export function suiteLogo(
	suite: Pick<ConformanceSuiteRecord, 'suite_logo'>
): string | undefined {
	const logo = suite.suite_logo?.trim();
	return logo || undefined;
}

/** Hub detail route for a suite-grain catalog row. */
export function suiteHubHref(
	suite: Pick<ConformanceSuiteRecord, 'path_prefix'>
): string {
	return `/hub/conformance-checks/${suite.path_prefix}`;
}

/** Map product component slug to global entity metadata for tags and facet labels. */
export function entityForComponent(component: string): EntityData | undefined {
	switch (component.trim()) {
		case 'wallet':
			return entities.wallets;
		case 'issuer':
			return entities.credential_issuers;
		case 'verifier':
			return entities.verifiers;
		default:
			return undefined;
	}
}

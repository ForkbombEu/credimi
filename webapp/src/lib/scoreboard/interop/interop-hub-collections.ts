// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { entities, type EntityData } from '$lib/global/entities';

export const INTEROP_HUB_COLLECTIONS = [
	'wallets',
	'credential_issuers',
	'credentials',
	'verifiers',
	'use_cases_verifications',
	'conformance-checks'
] as const;

export type InteropHubCollection = (typeof INTEROP_HUB_COLLECTIONS)[number];

export function isInteropHubCollection(value: string): value is InteropHubCollection {
	return (INTEROP_HUB_COLLECTIONS as readonly string[]).includes(value);
}

export function interopHubEntity(hub: InteropHubCollection): EntityData {
	if (hub === 'conformance-checks') {
		return entities.conformance_checks;
	}
	return entities[hub];
}

// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { entities, type EntityData } from '$lib/global/entities';

import type { InteropHubCollection } from './interop-hub-collections';

export const interopEntityData: Record<InteropHubCollection, EntityData> = {
	wallets: entities.wallets,
	credential_issuers: entities.credential_issuers,
	credentials: entities.credentials,
	verifiers: entities.verifiers,
	use_cases_verifications: entities.use_cases_verifications,
	'conformance-checks': entities.conformance_checks
};

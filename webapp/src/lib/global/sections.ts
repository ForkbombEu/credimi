// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { entities, type EntityData } from './entities';

//

export const credentialIssuersAndCredentialsSection = joinEntityData(
	entities.credential_issuers,
	entities.credentials
);

export const verifiersAndUseCaseVerificationsSection = joinEntityData(
	entities.verifiers,
	entities.use_cases_verifications
);

export const baseSections: EntityData[] = [
	entities.wallets,
	credentialIssuersAndCredentialsSection,
	verifiersAndUseCaseVerificationsSection,
	entities.custom_checks,
	entities.pipelines
];

export const marketplaceSections: EntityData[] = [...baseSections, entities.conformance_checks];

//

function joinEntityData(entity1: EntityData, entity2: EntityData): EntityData {
	return {
		...entity1,
		slug: entity1.slug + '-and-' + entity2.slug,
		labels: {
			singular: `${entity1.labels.singular} / ${entity2.labels.singular}`,
			plural: `${entity1.labels.plural} / ${entity2.labels.plural}`
		}
	};
}

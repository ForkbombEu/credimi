// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { entities } from '$lib/global';

import type { ScoreboardExpandedEntity } from '../types';
import type { ChildLink, Item } from './types';

import { fromPocketbaseEntity } from './from-pocketbase';

//

/** Issuance topic: credential issuers with nested credentials (product grouping). */
export function fromIssuanceItems(
	issuers: ScoreboardExpandedEntity[],
	credentials: ScoreboardExpandedEntity[]
): Item[] {
	return issuers.map((issuer) => {
		const children: ChildLink[] = credentials
			.filter((credential) => credential.credential_issuer === issuer.id)
			.map((credential) => {
				const entityItem = fromPocketbaseEntity(credential);
				return {
					label: entityItem.name,
					href: entityItem.href,
					avatar: entityItem.avatar
				};
			});

		return {
			...fromPocketbaseEntity(issuer, entities.credential_issuers),
			children: children.length > 0 ? children : undefined
		};
	});
}

/** Card/summary presentations: verifiers with nested use-case verifications.
 * The scoreboard Presentations column stays logos-only by product choice. */
export function fromPresentationSummaryItems(
	verifiers: ScoreboardExpandedEntity[],
	useCaseVerifications: ScoreboardExpandedEntity[]
): Item[] {
	return verifiers.map((verifier) => {
		const children: ChildLink[] = useCaseVerifications
			.filter((verification) => verification.verifier === verifier.id)
			.map((verification) => {
				const entityItem = fromPocketbaseEntity(verification);
				return {
					label: entityItem.name,
					href: entityItem.href,
					avatar: entityItem.avatar
				};
			});

		return {
			...fromPocketbaseEntity(verifier, entities.verifiers),
			children: children.length > 0 ? children : undefined
		};
	});
}

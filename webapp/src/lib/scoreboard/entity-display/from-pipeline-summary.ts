// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { entities } from '$lib/global';

import type { ScoreboardRow } from '../types';
import type { ChildLink, Item } from './types';

import { fromConformancePaths } from './from-conformance';
import { fromPocketbaseEntity } from './from-pocketbase';
import { fromWalletRows } from './from-wallets';

//

export function buildPipelineSummaryItems(row: ScoreboardRow): Item[] {
	const wallets = row.expanded_data?.wallets ?? [];
	const walletVersions = row.expanded_data?.wallet_versions ?? [];
	const issuers = row.expanded_data?.issuers ?? [];
	const verifiers = row.expanded_data?.verifiers ?? [];
	const credentials = row.expanded_data?.credentials ?? [];
	const useCaseVerifications = row.expanded_data?.use_case_verifications ?? [];

	const walletItems = fromWalletRows(
		wallets.map((wallet) => ({
			wallet,
			version: walletVersions.find((version) => version.wallet === wallet.id)
		}))
	);

	const issuerItems: Item[] = issuers.map((issuer) => {
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

	const verifierItems: Item[] = verifiers.map((verifier) => {
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

	const conformanceItems = fromConformancePaths(row.conformance_checks ?? []);

	return [...walletItems, ...issuerItems, ...verifierItems, ...conformanceItems];
}

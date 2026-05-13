// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { entities } from '$lib/global';

import type { ScoreboardRow } from '../types';

import { fromConformancePaths } from './from-conformance';
import { fromPocketbaseEntity, getPocketbaseEntityHref } from './from-pocketbase';
import { fromWalletRows } from './from-wallets';
import type { ChildLink, Item } from './types';

//

export function buildPipelineSummaryItems(row: ScoreboardRow): Item[] {
	const wallets = row.expand?.wallets ?? [];
	const walletVersions = row.expand?.wallet_versions ?? [];
	const issuers = row.expand?.issuers ?? [];
	const verifiers = row.expand?.verifiers ?? [];
	const credentials = row.expand?.credentials ?? [];
	const useCaseVerifications = row.expand?.use_case_verifications ?? [];

	const walletItems = fromWalletRows(
		wallets.map((wallet) => ({
			wallet,
			version: walletVersions.find((version) => version.wallet === wallet.id)
		}))
	);

	const issuerItems: Item[] = issuers.map((issuer) => {
		const children: ChildLink[] = credentials
			.filter((credential) => credential.credential_issuer === issuer.id)
			.map((credential) => ({
				label: credential.name,
				href: getPocketbaseEntityHref(credential)
			}));

		return {
			...fromPocketbaseEntity(issuer, entities.credential_issuers),
			children: children.length > 0 ? children : undefined
		};
	});

	const verifierItems: Item[] = verifiers.map((verifier) => {
		const children: ChildLink[] = useCaseVerifications
			.filter((verification) => verification.verifier === verifier.id)
			.map((verification) => ({
				label: verification.name,
				href: getPocketbaseEntityHref(verification)
			}));

		return {
			...fromPocketbaseEntity(verifier, entities.verifiers),
			children: children.length > 0 ? children : undefined
		};
	});

	const conformanceItems = fromConformancePaths(row.conformance_checks ?? []);

	return [...walletItems, ...issuerItems, ...verifierItems, ...conformanceItems];
}

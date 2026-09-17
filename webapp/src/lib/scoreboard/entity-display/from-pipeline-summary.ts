// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { ScoreboardRow } from '../types';
import type { Item } from './types';

import { fromConformancePaths } from './from-conformance';
import { fromIssuanceItems, fromPresentationSummaryItems } from './from-issuance';
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

	return [
		...walletItems,
		...fromIssuanceItems(issuers, credentials),
		...fromPresentationSummaryItems(verifiers, useCaseVerifications),
		...fromConformancePaths(row.conformance_checks ?? [])
	];
}

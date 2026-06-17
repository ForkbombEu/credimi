// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Link } from '@/components/types';

import { m } from '@/i18n';

//

export type ExtraLink = Link & { description: string };

export const leftItems: Link[] = [
	{
		href: '/hub',
		title: m.Hub()
	},
	{
		href: '/scoreboard',
		title: m.Scoreboard()
	}
];

export const extras: ExtraLink[] = [
	{
		href: 'https://capture-wallet.credimi.io/',
		title: m.extra_wallet_metadata_extractor(),
		description: m.extra_wallet_metadata_extractor_description()
	},
	{
		href: 'https://capture-issuer-verifier.credimi.io/',
		title: m.extra_issuer_verifier_metadata_extractor(),
		description: m.extra_issuer_verifier_metadata_extractor_description()
	},
	{
		href: 'https://atlas.credimi.io/',
		title: m.extra_eudi_atlas(),
		description: m.extra_eudi_atlas_description()
	}
];

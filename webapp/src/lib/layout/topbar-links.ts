// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Link } from '@/components/types';

import { m } from '@/i18n';

//

export type ExtraLink = Link & { description: string };

export type ExtraGroup = {
	label: string;
	items: ExtraLink[];
};

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

const tools: ExtraLink[] = [
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
		href: 'https://trust-inspector.credimi.io/',
		title: m.extra_trust_inspector(),
		description: m.extra_trust_inspector_description()
	},
	{
		href: 'https://lote.credimi.io/',
		title: m.extra_tl_lote_onboarding_catalogue(),
		description: m.extra_tl_lote_onboarding_catalogue_description()
	},
	{
		href: 'https://tsl.credimi.io/',
		title: m.extra_token_status_list_console(),
		description: m.extra_token_status_list_console_description()
	}
];

const docs: ExtraLink[] = [
	{
		href: 'https://atlas.credimi.io/',
		title: m.extra_eudi_atlas(),
		description: m.extra_eudi_atlas_description()
	}
];

export const extraGroups: ExtraGroup[] = [
	{ label: m.tools(), items: tools },
	{ label: m.Docs(), items: docs }
];

export const extras: ExtraLink[] = extraGroups.flatMap((group) => group.items);

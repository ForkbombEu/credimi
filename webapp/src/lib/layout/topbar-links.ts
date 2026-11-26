// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import {
	BookCheck,
	FileSearch,
	ListChecks,
	ListPlus,
	Map,
	Network,
	ScanQrCode,
	ShieldBan
} from '@lucide/svelte';

import type { Link } from '@/components/types';

import { m } from '@/i18n';

export const leftItems: Link[] = [
	{
		href: '/hub',
		title: m.Hub()
	},
	{
		href: '/scoreboard',
		title: m.Scoreboard()
	},
	{
		href: '/pages/extras',
		title: m.Extras()
	}
];

// The Extras: one icon per tool, shared by the Extras page and the dashboard sidebar.
export const extras = [
	{
		id: 'capture-wallet',
		icon: ScanQrCode,
		href: 'https://capture-wallet.credimi.io/',
		repo: 'https://github.com/ForkbombEu/credimi-capture-wallet',
		title: m.extra_wallet_metadata_extractor()
	},
	{
		id: 'capture-issuer-verifier',
		icon: FileSearch,
		href: 'https://capture-issuer-verifier.credimi.io/',
		repo: 'https://github.com/ForkbombEu/eudi-conformance-evidence',
		title: m.extra_issuer_verifier_metadata_extractor()
	},
	{
		id: 'trust-inspector',
		icon: ListChecks,
		href: 'https://trust-inspector.credimi.io/',
		repo: 'https://github.com/ForkbombEu/eudi-trusted-lists-inspector',
		title: m.extra_trust_inspector()
	},
	{
		id: 'lote',
		icon: ListPlus,
		href: 'https://lote.credimi.io/',
		repo: 'https://github.com/ForkbombEu/eudi-trusted-list-publisher',
		title: m.extra_tl_lote_onboarding_catalogue()
	},
	{
		id: 'tsl',
		icon: ShieldBan,
		href: 'https://tsl.credimi.io/',
		repo: 'https://github.com/ForkbombEu/capture-status-list',
		title: m.extra_token_status_list_console()
	},
	{
		id: 'federation',
		icon: Network,
		href: 'https://federation.credimi.io/',
		repo: 'https://github.com/ForkbombEu/credimi-federation',
		title: m.extra_openid_federation_trust_anchor()
	},
	{
		id: 'attestbook',
		icon: BookCheck,
		href: 'https://attestbook.credimi.io/',
		repo: 'https://github.com/ForkbombEu/credimi-attestbook',
		title: m.extra_attestation_rulebooks_catalogue()
	},
	{
		id: 'atlas',
		icon: Map,
		href: 'https://atlas.credimi.io/',
		repo: 'https://github.com/ForkbombEu/eudi-conformance-atlas',
		title: m.extra_eudi_atlas()
	}
] as const;

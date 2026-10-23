// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { HubItem } from '$lib/hub';

import { Scoreboard } from '$lib';
import { SITE_ORIGIN, softwareApplicationJsonLd, staticPages } from '$lib/seo';

import { pb } from '@/pocketbase';
import { Collections } from '@/pocketbase/types';

export const load = async ({ fetch }) => {
	const wallets = await pb.collection('hub_items').getList<HubItem>(1, 3, {
		filter: `type = '${Collections.Wallets}'`,
		fetch,
		sort: '@random'
	});
	const issuers = await pb.collection('hub_items').getList<HubItem>(1, 3, {
		filter: `type = '${Collections.CredentialIssuers}'`,
		fetch,
		sort: '@random'
	});
	const verifiers = await pb.collection('hub_items').getList<HubItem>(1, 3, {
		filter: `type = '${Collections.Verifiers}'`,
		fetch,
		sort: '@random'
	});
	const scoreboard = await Scoreboard.Records.loadPage({
		fetch,
		page: 1,
		perPage: 6,
		sort: '@random'
	});
	return {
		wallets,
		issuers,
		verifiers,
		scoreboard: { records: scoreboard.items },
		seo: { ...staticPages['/'], jsonLd: [softwareApplicationJsonLd(SITE_ORIGIN)] }
	};
};

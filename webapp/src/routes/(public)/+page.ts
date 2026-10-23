// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { HubItem } from '$lib/hub';

import { loadPage } from '$lib/scoreboard/records';
import { SITE_ORIGIN, softwareApplicationJsonLd, staticPages } from '$lib/seo';

import { pb } from '@/pocketbase';
import { Collections } from '@/pocketbase/types';

export const load = async ({ fetch }) => {
	const hubItems = (type: Collections) =>
		pb.collection('hub_items').getList<HubItem>(1, 3, {
			filter: pb.filter('type = {:type}', { type }),
			fetch,
			sort: '@random',
			// In the browser the SDK cancels a pending request when another one shares its key.
			requestKey: `home-${type}`
		});
	// Independent requests: in sequence they added up to the time to first byte.
	const [wallets, issuers, verifiers, scoreboard] = await Promise.all([
		hubItems(Collections.Wallets),
		hubItems(Collections.CredentialIssuers),
		hubItems(Collections.Verifiers),
		loadPage({ fetch, page: 1, perPage: 6, sort: '-success_rate,-total_runs' })
	]);
	return {
		wallets,
		issuers,
		verifiers,
		scoreboard: { records: scoreboard.items },
		seo: { ...staticPages['/'], jsonLd: [softwareApplicationJsonLd(SITE_ORIGIN)] }
	};
};

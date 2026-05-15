// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { HubItem, HubItemType } from '$lib/hub';

import { pb } from '@/pocketbase';

//

export async function searchHub(path: string, type: HubItemType) {
	const result = await pb.collection('hub_items').getList(1, 10, {
		filter: pb.filter('path ~ {:path} && type = {:type}', { path, type }),
		requestKey: null
	});
	return result.items as HubItem[];
}

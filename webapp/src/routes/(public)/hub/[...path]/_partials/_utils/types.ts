// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { HubItemType } from '$lib/hub';
import type { Merge } from 'type-fest';

//

export function pageDetails<K extends HubItemType, Data extends object>(
	type: K,
	data: Data
): Merge<{ type: K }, Data> {
	return { type, ...data };
}

// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Standard } from './types';

import { listAll } from './query';

//

const Store = $state({
	standards: [] as Standard[]
});

export function get() {
	return $state.snapshot(Store);
}

export function load() {
	listAll().match({
		Rejected: (reason) => {
			console.error(reason);
		},
		Resolved: (standards) => {
			Store.standards = standards;
		}
	});
}

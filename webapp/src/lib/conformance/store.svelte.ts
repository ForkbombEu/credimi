// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Standard } from './types';

import { listAll } from './query';

//

const standards = $state<Standard[]>([]);

const readonlyView = {
	get standards(): readonly Standard[] {
		return standards as readonly Standard[];
	}
};

export function get() {
	return readonlyView;
}

export function load() {
	listAll().match({
		Rejected: (reason) => {
			console.error(reason);
		},
		Resolved: (next) => {
			standards.length = 0;
			standards.push(...next);
		}
	});
}

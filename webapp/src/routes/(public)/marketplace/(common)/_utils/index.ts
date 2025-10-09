// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';

export function getRestParams(rest: string) {
	const chunks = rest.split('/');
	if (chunks.length !== 2) error(404);
	return {
		organization: chunks[0],
		entity: chunks[1]
	};
}

export function getFilterFromRestParams(rest: string) {
	const { organization, entity } = getRestParams(rest);
	return `owner.canonified_name = '${organization}' && canonified_name = '${entity}'`;
}

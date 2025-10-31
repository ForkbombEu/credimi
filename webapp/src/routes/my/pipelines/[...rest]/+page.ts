// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';

import { pb } from '@/pocketbase/index.js';

//

export const load = async ({ params, fetch }) => {
	if (params.rest == 'new') {
		return {
			mode: 'create' as const
		};
	} else if (isEditMode(params.rest)) {
		const id = params.rest.split('-')[1];
		const record = await pb.collection('pipelines').getOne(id, { fetch });
		if (!record) error(404);
		return {
			mode: 'edit' as const,
			record
		};
	} else {
		error(404);
	}
};

function isEditMode(rest: string) {
	const parts = rest.split('-');
	return parts.length === 2 && parts[0] === 'edit';
}

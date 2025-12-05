// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { fetchPipeline } from '$lib/pipeline-form/serde.js';

//

export const load = async ({ params, fetch }) => {
	const res = getModeFromParam(params.action);
	if (res.mode === 'create') {
		return res;
	} else {
		try {
			const pipeline = await fetchPipeline(res.id, { fetch });
			return {
				...res,
				pipeline
			};
		} catch {
			error(404);
		}
	}
};

function getModeFromParam(param: string) {
	const parts = param.split('-');
	if (parts.length == 2 && parts[0] == 'edit') {
		return {
			mode: 'edit' as const,
			id: parts[1]
		};
	} else if (parts.length == 2 && parts[0] == 'view') {
		return {
			mode: 'view' as const,
			id: parts[1]
		};
	} else if (parts.length == 1 && parts[0] == 'new') {
		return {
			mode: 'create' as const
		};
	} else {
		error(404);
	}
}

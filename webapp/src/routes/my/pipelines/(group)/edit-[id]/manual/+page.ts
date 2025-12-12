// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';

import { pb } from '@/pocketbase/index.js';

//

export const load = async ({ params }) => {
	try {
		const { id } = params;
		const pipeline = await pb.collection('pipelines').getOne(id, { fetch });
		return { pipeline };
	} catch {
		error(404);
	}
};

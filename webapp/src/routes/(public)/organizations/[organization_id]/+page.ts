// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase/index.js';

export const load = async ({ params, fetch }) => {
	const organization = await pb.collection('organizations').getOne(params.organization_id, {
		fetch
	});

	return { organization };
};

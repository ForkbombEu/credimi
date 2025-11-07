// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase';

export const load = async ({ params, fetch }) => {
	const pipeline = await pb.collection('pipelines').getOne(params.id, { fetch });
	return { pipeline };
};

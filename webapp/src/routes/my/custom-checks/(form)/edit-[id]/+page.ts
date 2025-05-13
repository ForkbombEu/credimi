// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase';

export const load = async ({ params }) => {
	const { id } = params;

	const record = await pb.collection('custom_checks').getOne(id);

	return { record };
};

// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase';

export const load = async ({ params }) => {
	const verifier = await pb.collection('verifiers').getOne(params.verifier_id);
	return {
		verifier
	};
};

// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { getNestStandards } from '$lib/conformance';

//

export const load = async ({ fetch }) => {
	const result = await getNestStandards({ fetch, surface: 'manual' });

	if (result instanceof Error) {
		error(500, { message: result.message });
	} else {
		return {
			standardsAndTestSuites: result
		};
	}
};

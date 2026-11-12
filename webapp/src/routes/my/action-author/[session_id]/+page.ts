// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { SESSION_ID_PATTERN } from '$lib/action-author/api';

//

export const load = ({ params, url }) => {
	if (!SESSION_ID_PATTERN.test(params.session_id)) error(404);
	return {
		sessionId: params.session_id,
		deviceId: url.searchParams.get('device') ?? ''
	};
};

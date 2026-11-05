// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';

//

const SESSION_ID =
	/^ActionAuthorSession-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/;

export const load = ({ params, url }) => {
	if (!SESSION_ID.test(params.session_id)) error(404);
	return {
		sessionId: params.session_id,
		deviceId: url.searchParams.get('device') ?? ''
	};
};

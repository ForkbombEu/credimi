// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { error } from '@sveltejs/kit';
import { getWalletTestParams } from './_partials';

const ALLOWED = new Set(['wallet', 'ewc', 'eudiw']);

export const load = ({ url }) => {
	const params = getWalletTestParams(url);
	if (!params.workflowId || !isAllowedTestUrl(url)) {
		error(404);
	}

	return params;
};

function isAllowedTestUrl(url: URL): boolean {
  const segments = url.pathname.match(/[^/]+/g) || [];
  const last = segments[segments.length - 1];
  return ALLOWED.has(last);
}

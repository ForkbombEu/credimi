// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// THROWAWAY: delete this folder + the DEV mock in live-view.ts when done previewing.

import { error } from '@sveltejs/kit';
import { dev } from '$app/environment';

export const load = () => {
	if (!dev) error(404);
};

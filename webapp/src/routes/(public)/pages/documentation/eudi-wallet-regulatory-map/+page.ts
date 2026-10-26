// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { redirect } from '@sveltejs/kit';

// The regulatory map moved to the EUDI Trust & Conformance Atlas, which maintains it; old links keep working.
export function load() {
	redirect(308, 'https://atlas.credimi.io/');
}

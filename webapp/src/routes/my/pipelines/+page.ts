// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { Pipeline } from '$lib';

export const load = async ({ fetch }) => {
	const workflows = await Pipeline.Workflows.listAll({ fetch });
	return { workflows };
};

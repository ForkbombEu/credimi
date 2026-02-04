// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getAllPipelinesWorkflows } from './_partials/workflows';

export const load = async ({ fetch }) => {
	const workflows = await getAllPipelinesWorkflows({ fetch });
	return { workflows };
};

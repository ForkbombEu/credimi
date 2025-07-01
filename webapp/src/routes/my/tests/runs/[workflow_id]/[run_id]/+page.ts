// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

export const load = async ({ params }) => {
	const { workflow_id, run_id } = params;

	return {
		workflowId: workflow_id,
		runId: run_id
	};
};

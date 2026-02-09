// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase';

//

export async function cancel(workflowId: string, runId: string) {
	return await pb.send(`/api/my/checks/${workflowId}/runs/${runId}/cancel`, {
		method: 'POST'
	});
}

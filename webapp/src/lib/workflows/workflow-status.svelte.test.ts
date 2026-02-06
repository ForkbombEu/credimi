// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { expect, test } from 'vitest';
import { render } from 'vitest-browser-svelte';

import WorkflowStatus from './workflow-status.svelte';

test('renders queued badge with position', async () => {
	const screen = render(WorkflowStatus, {
		status: 'Queued',
		queue: {
			ticket_id: 'ticket-1',
			position: 0,
			line_len: 2,
			runner_ids: ['runner-1']
		}
	});

	await expect.element(screen.getByText('Queued')).toBeVisible();
	await expect.element(screen.getByText('1 of 2')).toBeVisible();
});

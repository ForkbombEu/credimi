// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { expect, test, vi } from 'vitest';
import { render } from 'vitest-browser-svelte';

import WorkflowActions from './workflow-actions.svelte';

vi.mock('@/pocketbase', () => ({
	pb: {
		send: vi.fn()
	}
}));

vi.mock('$lib/layout/global-loading.svelte', () => ({
	runWithLoading: ({ fn }: { fn: () => Promise<unknown> }) => fn()
}));

vi.mock('@/i18n', () => ({
	m: {
		Cancel: () => 'Cancel',
		Swagger: () => 'Swagger'
	},
	localizeHref: (href: string) => href
}));

test('queued cancel triggers queue endpoint', async () => {
	const { pb } = await import('@/pocketbase');

	const screen = render(WorkflowActions, {
		workflow: {
			workflowId: 'wf-1',
			runId: 'run-1',
			status: null,
			name: 'Queued run',
			queue: {
				ticket_id: 'ticket-queued',
				runner_ids: ['runner-1', 'runner-2']
			}
		},
		mode: 'buttons'
	});

	await screen.getByRole('button', { name: 'Cancel' }).click();

	expect(vi.mocked(pb.send)).toHaveBeenCalledWith(
		'/api/pipeline/queue/ticket-queued?runner_ids=runner-1%2Crunner-2',
		{
			method: 'DELETE'
		}
	);
});

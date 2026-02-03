// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { PipelinesResponse } from '@/pocketbase/types';

import { runPipeline } from './utils';

vi.mock('@/i18n', () => ({
	goto: vi.fn(),
	m: {
		Pipeline_started_successfully: () => 'Pipeline started successfully',
		View_workflow_details: () => 'View workflow details',
		View: () => 'View',
		Cancel: () => 'Cancel'
	}
}));

vi.mock('svelte-sonner', () => ({
	toast: {
		success: vi.fn(),
		error: vi.fn(),
		info: vi.fn(() => 'toast-id'),
		message: vi.fn(),
		dismiss: vi.fn()
	}
}));

vi.mock('@/pocketbase', () => ({
	pb: {
		send: vi.fn()
	}
}));

vi.mock('$lib/utils', async () => {
	const actual = await vi.importActual<typeof import('$lib/utils')>('$lib/utils');
	return {
		...actual,
		runWithLoading: async ({ fn }: { fn: () => Promise<unknown> }) => fn()
	};
});

describe('runPipeline', () => {
	beforeEach(async () => {
		vi.useFakeTimers();
		const { pb } = await import('@/pocketbase');
		const { toast } = await import('svelte-sonner');
		const { goto } = await import('@/i18n');
		vi.mocked(pb.send).mockReset();
		vi.mocked(toast.success).mockReset();
		vi.mocked(toast.error).mockReset();
		vi.mocked(toast.info).mockReset();
		vi.mocked(toast.message).mockReset();
		vi.mocked(toast.dismiss).mockReset();
		vi.mocked(goto).mockReset();
	});

	afterEach(() => {
		vi.useRealTimers();
	});

	it('polls until running and exposes workflow navigation', async () => {
		const { pb } = await import('@/pocketbase');
		const { toast } = await import('svelte-sonner');
		const { goto } = await import('@/i18n');

		vi.mocked(pb.send)
			.mockResolvedValueOnce({
				ticket_id: 'ticket-1',
				runner_ids: ['runner-1'],
				status: 'queued',
				position: 0,
				line_len: 1
			})
			.mockResolvedValueOnce({
				ticket_id: 'ticket-1',
				runner_ids: ['runner-1'],
				status: 'running',
				position: 0,
				line_len: 1,
				workflow_id: 'wf-1',
				run_id: 'run-1'
			});

		const pipeline = {
			id: 'pipeline-1',
			yaml: `
name: test
steps:
  - id: step-1
    use: mobile-automation
    with:
      runner_id: runner-1
      action_id: action-1
`,
			__canonified_path__: 'pipelines/test'
		} as PipelinesResponse;

		await runPipeline(pipeline);
		await Promise.resolve();
		await Promise.resolve();

		expect(vi.mocked(pb.send)).toHaveBeenCalledTimes(2);
		expect(vi.mocked(toast.success)).toHaveBeenCalledTimes(1);

		const [, options] = vi.mocked(toast.success).mock.calls[0] ?? [];
		expect(options?.action?.onClick).toBeTypeOf('function');
		options?.action?.onClick();
		expect(vi.mocked(goto)).toHaveBeenCalledWith('/my/tests/runs/wf-1/run-1');

		vi.advanceTimersByTime(3000);
		expect(vi.mocked(pb.send)).toHaveBeenCalledTimes(2);
	});
});

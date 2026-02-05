// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { PipelinesResponse } from '@/pocketbase/types';

import { runPipeline } from './utils';

const pipelineFixture = (id: string, runnerId = 'runner-1'): PipelinesResponse => ({
	id,
	yaml: `
name: test
steps:
  - id: step-1
    use: mobile-automation
    with:
      runner_id: ${runnerId}
      action_id: action-1
`,
	__canonified_path__: 'pipelines/test'
});

const flushPromises = async () => {
	await Promise.resolve();
	await Promise.resolve();
};

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

		await runPipeline(pipelineFixture('pipeline-1'));
		await flushPromises();

		expect(vi.mocked(pb.send)).toHaveBeenCalledTimes(2);
		expect(vi.mocked(toast.success)).toHaveBeenCalledTimes(1);

		const [, options] = vi.mocked(toast.success).mock.calls[0] ?? [];
		expect(options?.action?.onClick).toBeTypeOf('function');
		options?.action?.onClick();
		expect(vi.mocked(goto)).toHaveBeenCalledWith('/my/tests/runs/wf-1/run-1');

		vi.advanceTimersByTime(3000);
		expect(vi.mocked(pb.send)).toHaveBeenCalledTimes(2);
	});

	it('shows enqueue errors and skips polling', async () => {
		const { pb } = await import('@/pocketbase');
		const { toast } = await import('svelte-sonner');

		vi.mocked(pb.send).mockRejectedValueOnce(new Error('queue limit exceeded'));

		await runPipeline(pipelineFixture('pipeline-1'));
		await flushPromises();

		expect(vi.mocked(toast.error)).toHaveBeenCalledWith('queue limit exceeded');
		expect(vi.mocked(toast.info)).not.toHaveBeenCalled();
		expect(vi.mocked(pb.send)).toHaveBeenCalledTimes(1);

		vi.advanceTimersByTime(2000);
		expect(vi.mocked(pb.send)).toHaveBeenCalledTimes(1);
	});

	it('keeps polling when cancel fails', async () => {
		const { pb } = await import('@/pocketbase');
		const { toast } = await import('svelte-sonner');

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
				status: 'queued',
				position: 0,
				line_len: 1
			})
			.mockRejectedValueOnce(new Error('cancel failed'))
			.mockResolvedValueOnce({
				ticket_id: 'ticket-1',
				runner_ids: ['runner-1'],
				status: 'queued',
				position: 0,
				line_len: 1
			});

		await runPipeline(pipelineFixture('pipeline-1'));
		await flushPromises();

		const [, options] = vi.mocked(toast.info).mock.calls[0] ?? [];
		await options?.action?.onClick?.();

		expect(vi.mocked(toast.dismiss)).not.toHaveBeenCalled();
		expect(vi.mocked(pb.send)).toHaveBeenCalledTimes(3);

		vi.advanceTimersByTime(1000);
		await Promise.resolve();
		await Promise.resolve();

		expect(vi.mocked(pb.send)).toHaveBeenCalledTimes(4);
	});

	it('does not mark as canceled when cancel returns running', async () => {
		const { pb } = await import('@/pocketbase');
		const { toast } = await import('svelte-sonner');

		vi.mocked(pb.send)
			.mockResolvedValueOnce({
				ticket_id: 'ticket-2',
				runner_ids: ['runner-1'],
				status: 'queued',
				position: 0,
				line_len: 1
			})
			.mockResolvedValueOnce({
				ticket_id: 'ticket-2',
				runner_ids: ['runner-1'],
				status: 'queued',
				position: 0,
				line_len: 1
			})
			.mockResolvedValueOnce({
				ticket_id: 'ticket-2',
				runner_ids: ['runner-1'],
				status: 'running',
				workflow_id: 'wf-2',
				run_id: 'run-2'
			});

		await runPipeline(pipelineFixture('pipeline-2'));
		await flushPromises();

		const [, options] = vi.mocked(toast.info).mock.calls[0] ?? [];
		await options?.action?.onClick?.();

		expect(vi.mocked(toast.success)).toHaveBeenCalledTimes(1);
		expect(vi.mocked(toast.message)).not.toHaveBeenCalled();
		expect(vi.mocked(pb.send)).toHaveBeenCalledTimes(3);

		vi.advanceTimersByTime(1000);
		expect(vi.mocked(pb.send)).toHaveBeenCalledTimes(3);
	});

	it('keeps polling when cancel returns queued', async () => {
		const { pb } = await import('@/pocketbase');
		const { toast } = await import('svelte-sonner');

		vi.mocked(pb.send)
			.mockResolvedValueOnce({
				ticket_id: 'ticket-3',
				runner_ids: ['runner-1'],
				status: 'queued',
				position: 0,
				line_len: 1
			})
			.mockResolvedValueOnce({
				ticket_id: 'ticket-3',
				runner_ids: ['runner-1'],
				status: 'queued',
				position: 0,
				line_len: 1
			})
			.mockResolvedValueOnce({
				ticket_id: 'ticket-3',
				runner_ids: ['runner-1'],
				status: 'queued',
				position: 0,
				line_len: 1
			})
			.mockResolvedValueOnce({
				ticket_id: 'ticket-3',
				runner_ids: ['runner-1'],
				status: 'queued',
				position: 0,
				line_len: 1
			});

		await runPipeline(pipelineFixture('pipeline-3'));
		await flushPromises();

		const [, options] = vi.mocked(toast.info).mock.calls[0] ?? [];
		await options?.action?.onClick?.();

		expect(vi.mocked(toast.dismiss)).not.toHaveBeenCalled();
		expect(vi.mocked(pb.send)).toHaveBeenCalledTimes(3);

		vi.advanceTimersByTime(1000);
		await Promise.resolve();

		expect(vi.mocked(pb.send)).toHaveBeenCalledTimes(4);
	});
});

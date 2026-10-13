// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import {
	buildPipelineResultViewerHref,
	maestroFlowsAt,
	parseMaestroTimeline
} from './maestro-timeline';

const START = 1_000_000;

function jsonl(...entries: Record<string, unknown>[]): string {
	return entries.map((entry) => JSON.stringify(entry)).join('\n');
}

function command(
	t: number,
	flowId: string,
	index: number,
	status: string,
	extra: Record<string, unknown> = {}
) {
	return {
		t,
		kind: 'command',
		flow_id: flowId,
		index,
		parent: -1,
		depth: 0,
		description: `Command ${index}`,
		status,
		...extra
	};
}

describe('parseMaestroTimeline', () => {
	it('skips invalid lines and picks the recording start', () => {
		const text = [
			'',
			'not json',
			JSON.stringify({ kind: 'command' }),
			JSON.stringify({ t: START, kind: 'recording_started', index: 0, parent: 0, depth: 0 }),
			'   ',
			JSON.stringify(command(START + 2000, 'f1', 1, 'COMPLETED')),
			JSON.stringify(command(START + 1000, 'f1', 1, 'RUNNING')),
			JSON.stringify({ t: START + 5000, kind: 'recording_started', index: 0 }),
			'{"t": 1, "kind": "unknown", "index": 0}'
		].join('\n');

		const timeline = parseMaestroTimeline(text);

		expect(timeline.recordingStartedAt).toBe(START);
		expect(timeline.screenshotRow).toBeUndefined();
		expect(timeline.entries.map((entry) => [entry.t, entry.status])).toEqual([
			[START + 1000, 'RUNNING'],
			[START + 2000, 'COMPLETED']
		]);
	});

	it('sets the screenshot row from the last screenshot entry', () => {
		const timeline = parseMaestroTimeline(
			jsonl(
				command(START + 1000, 'f1', 2, 'FAILED'),
				{ t: START + 2000, kind: 'screenshot', flow_id: 'f0', index: 1 },
				{ t: START + 3000, kind: 'screenshot', flow_id: 'f1', index: 2 }
			)
		);

		expect(timeline.screenshotRow).toEqual({ flowId: 'f1', index: 2 });
		expect(timeline.entries).toHaveLength(1);
	});
});

describe('maestroFlowsAt', () => {
	it('replays a row from RUNNING to COMPLETED', () => {
		const timeline = parseMaestroTimeline(
			jsonl(
				command(START + 1000, 'f1', 1, 'RUNNING', { step_id: 'get-credential' }),
				command(START + 2000, 'f1', 1, 'COMPLETED', { step_id: 'get-credential' })
			)
		);

		expect(maestroFlowsAt(timeline, START + 500)).toEqual([]);

		const running = maestroFlowsAt(timeline, START + 1500);
		expect(running).toHaveLength(1);
		expect(running[0].title).toBe('get-credential:');
		expect(running[0].rows).toEqual([
			{
				index: 1,
				parent: -1,
				depth: 0,
				kind: 'command',
				description: 'Command 1',
				status: 'RUNNING',
				message: undefined,
				startedAt: START + 1000
			}
		]);

		const completed = maestroFlowsAt(timeline, START + 2000);
		expect(completed[0].rows[0].status).toBe('COMPLETED');
		expect(completed[0].rows[0].startedAt).toBe(START + 1000);
	});

	it('interrupts running rows of an older flow once a newer flow starts', () => {
		const timeline = parseMaestroTimeline(
			jsonl(
				command(START + 1000, 'f1', 2, 'RUNNING'),
				command(START + 1100, 'f1', 1, 'COMPLETED'),
				command(START + 3000, 'f2', 1, 'RUNNING')
			)
		);

		const before = maestroFlowsAt(timeline, START + 2000);
		expect(before[0].rows.map((row) => [row.index, row.status])).toEqual([
			[1, 'COMPLETED'],
			[2, 'RUNNING']
		]);

		const after = maestroFlowsAt(timeline, START + 3000);
		expect(after.map((flow) => flow.flowId)).toEqual(['f1', 'f2']);
		expect(after[0].rows[1].status).toBe('INTERRUPTED');
		expect(after[1].rows[0].status).toBe('RUNNING');

		const final = maestroFlowsAt(timeline, Infinity);
		expect(final[1].rows[0].status).toBe('INTERRUPTED');
	});

	it('numbers retries of a step only after a flow that did not succeed', () => {
		const step = { step_id: 'get-credential' };
		const timeline = parseMaestroTimeline(
			jsonl(
				command(START + 1000, 'a', 1, 'FAILED', step),
				{ ...command(START + 1100, 'a', 2, 'FAILED', step), kind: 'flow' },
				command(START + 2000, 'b', 1, 'COMPLETED', step),
				command(START + 3000, 'c', 1, 'COMPLETED', step),
				command(START + 4000, 'd', 1, 'COMPLETED')
			)
		);

		expect(maestroFlowsAt(timeline, Infinity).map((flow) => flow.title)).toEqual([
			'get-credential:',
			'get-credential (retry 1):',
			'get-credential:',
			'Flow 4:'
		]);
	});
});

describe('buildPipelineResultViewerHref', () => {
	const result = {
		video: '/api/files/pipeline_results/rec/video.mp4',
		screenshot: '/api/files/pipeline_results/rec/shot.png'
	};

	it('returns the raw file without a timeline', () => {
		expect(buildPipelineResultViewerHref('video', result)).toBe(result.video);
		expect(buildPipelineResultViewerHref('screenshot', { ...result, timeline: '' })).toBe(
			result.screenshot
		);
	});

	it('links the viewer page with a timeline', () => {
		const timeline = '/api/files/pipeline_results/rec/timeline.jsonl';
		expect(buildPipelineResultViewerHref('screenshot', { ...result, timeline })).toBe(
			'/pipeline-result?mode=screenshot' +
				`&video=${encodeURIComponent(result.video)}` +
				`&screenshot=${encodeURIComponent(result.screenshot)}` +
				`&timeline=${encodeURIComponent(timeline)}`
		);
	});
});

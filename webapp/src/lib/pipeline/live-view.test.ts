// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import {
	groupMaestroCommands,
	liveViewCommandsUrl,
	mergeMaestroCommands,
	type MaestroCommand
} from './live-view';
import { getExecutionDevices, type ExecutionSummary } from './workflows';

function command(overrides: Partial<MaestroCommand>): MaestroCommand {
	return {
		seq: 1,
		flow_id: 'action.yaml1',
		index: 0,
		parent: -1,
		depth: 0,
		description: 'Press Home key',
		status: 'RUNNING',
		...overrides
	};
}

describe('getExecutionDevices', () => {
	it('returns structured devices from the API', () => {
		const summary = {
			devices: [
				{ device_id: 'org/runner/a', name: 'a', live_view: true },
				{ device_id: 'org/runner/b', name: 'b', live_view: false }
			]
		} as ExecutionSummary;
		expect(getExecutionDevices(summary)).toEqual(summary.devices);
	});

	it('returns empty when devices are absent', () => {
		expect(getExecutionDevices({} as ExecutionSummary)).toEqual([]);
	});
});

describe('liveViewCommandsUrl', () => {
	it('appends the commands path to the stream url', () => {
		expect(liveViewCommandsUrl('https://runner.example/live/tok', 0)).toBe(
			'https://runner.example/live/tok/commands?after=0'
		);
	});

	it('drops trailing slashes from the stream url', () => {
		expect(liveViewCommandsUrl('https://runner.example/live/tok//', 42)).toBe(
			'https://runner.example/live/tok/commands?after=42'
		);
	});
});

describe('mergeMaestroCommands', () => {
	it('replaces updated rows in place and appends new rows in order', () => {
		const current = [
			command({ seq: 1, index: 0, status: 'COMPLETED' }),
			command({ seq: 2, index: 1, description: 'Run flow' })
		];
		const merged = mergeMaestroCommands(current, [
			command({ seq: 3, index: 2, parent: 1, depth: 1 }),
			command({ seq: 4, index: 1, description: 'Run flow', status: 'FAILED' }),
			command({ seq: 5, flow_id: 'action.yaml2', index: 0 })
		]);

		expect(merged.map((c) => [c.flow_id, c.index, c.status])).toEqual([
			['action.yaml1', 0, 'COMPLETED'],
			['action.yaml1', 1, 'FAILED'],
			['action.yaml1', 2, 'RUNNING'],
			['action.yaml2', 0, 'RUNNING']
		]);
		expect(current[1].status).toBe('RUNNING');
	});
});

describe('groupMaestroCommands', () => {
	it('groups consecutive rows by flow', () => {
		const first = command({ index: 0 });
		const nested = command({ index: 1, parent: 0, depth: 1 });
		const second = command({ flow_id: 'action.yaml2', index: 0 });

		expect(groupMaestroCommands([first, nested, second])).toEqual([
			{ flowId: 'action.yaml1', commands: [first, nested] },
			{ flowId: 'action.yaml2', commands: [second] }
		]);
	});
});

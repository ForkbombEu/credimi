// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import {
	type ExecutionStep,
	STEPS_INITIAL_LIMIT,
	stepDetailsHref,
	visibleSteps
} from './execution-steps';

function step(index: number, overrides: Partial<ExecutionStep> = {}): ExecutionStep {
	return {
		step_id: `step-${index}`,
		kind: 'activity',
		type: 'Make an HTTP request',
		display_name: `step-${index}`,
		status: 'Completed',
		history_event: { workflow_id: 'wf', run_id: 'run', event_id: index },
		...overrides
	};
}

function steps(count: number): ExecutionStep[] {
	return Array.from({ length: count }, (_, index) => step(index));
}

describe('visibleSteps', () => {
	it('shows the first steps of a long list', () => {
		expect(visibleSteps(steps(150), false)).toHaveLength(STEPS_INITIAL_LIMIT);
	});

	it('keeps a later failed step, in order', () => {
		const list = steps(150);
		list[140] = step(140, { status: 'Failed' });
		const shown = visibleSteps(list, false);
		expect(shown).toHaveLength(101);
		expect(shown[100]).toBe(list[140]);
		expect(shown[99]).toBe(list[99]);
	});

	it('shows every step on request', () => {
		expect(visibleSteps(steps(150), true)).toHaveLength(150);
	});
});

describe('stepDetailsHref', () => {
	it('opens an activity step at its history event', () => {
		expect(stepDetailsHref(step(7))).toBe('/my/tests/runs/wf/run?event=7');
	});

	it('opens a workflow step on its own run', () => {
		const workflowStep = step(8, {
			kind: 'workflow',
			execution: { workflowId: 'wf-check', runId: 'check-run' }
		});
		expect(stepDetailsHref(workflowStep)).toBe('/my/tests/runs/wf-check/check-run');
	});
});

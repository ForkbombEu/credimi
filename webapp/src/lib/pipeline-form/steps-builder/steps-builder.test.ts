// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { afterEach, describe, expect, it, vi } from 'vitest';

vi.mock('$lib', async () => {
	const { validateYaml } = await import('$lib/pipeline/validate-yaml');
	return { Pipeline: { validateYaml } };
});

vi.mock('./steps-builder.svelte', () => ({ default: class {} }));
vi.mock('$lib/layout/global-confirm.svelte', () => ({
	confirm: vi.fn()
}));

import { confirm } from '$lib/layout/global-confirm.svelte';

import * as ExecutionTarget from '../execution-target/state.svelte.js';
import { GLOBAL_RUNNER } from '../steps/wallet-action/wallet-action-step-form.svelte.js';
import type { EnrichedStep } from './types.js';
import { StepsBuilder } from './steps-builder.svelte.js';

const VALID_YAML = `name: test

steps:
  - use: debug
`;

const wallet = { id: 'w1', name: 'W' } as never;
const version = { id: 'v1', tag: '1' } as never;
const action = { id: 'a1', name: 'A' } as never;

function mobileTuple(): EnrichedStep {
	const data = { wallet, version, runner: GLOBAL_RUNNER, action };
	return [{ use: 'mobile-automation', id: 's', continue_on_error: false, with: {} }, data] as unknown as EnrichedStep;
}

function createBuilder(steps: EnrichedStep[] = []) {
	return new StepsBuilder({
		steps,
		yamlPreview: () => VALID_YAML
	});
}

type BuilderInternal = {
	state: { mode: StepsBuilder['mode']; steps: unknown[] };
};

describe('StepsBuilder undo/redo', () => {
	afterEach(() => {
		ExecutionTarget.clear();
		vi.clearAllMocks();
	});

	it('resyncs ExecutionTarget after undo and redo', () => {
		const builder = createBuilder([mobileTuple()]);
		ExecutionTarget.syncFromSteps(builder.steps);
		expect(ExecutionTarget.state.current?.wallet.id).toBe('w1');

		builder.deleteStep(0);
		expect(builder.steps).toHaveLength(0);
		expect(ExecutionTarget.state.current).toBeUndefined();

		builder.undo();
		expect(builder.steps).toHaveLength(1);
		expect(ExecutionTarget.state.current?.wallet.id).toBe('w1');

		builder.redo();
		expect(builder.steps).toHaveLength(0);
		expect(ExecutionTarget.state.current).toBeUndefined();
	});
});

describe('StepsBuilder manual mode', () => {
	afterEach(() => {
		vi.clearAllMocks();
	});

	it('exposes isSavedManualPipeline from constructor props', () => {
		const builder = new StepsBuilder({
			steps: [],
			yamlPreview: () => VALID_YAML,
			isSavedManualPipeline: true
		});

		expect(builder.isSavedManualPipeline).toBe(true);
	});

	it('defaults isSavedManualPipeline to false', () => {
		const builder = createBuilder();

		expect(builder.isSavedManualPipeline).toBe(false);
	});

	it('enterManualMode closes form mode and sets manual', () => {
		const builder = createBuilder();
		const exitFormState = vi.spyOn(builder, 'exitFormState');
		(builder as unknown as BuilderInternal).state.mode = {
			id: 'form',
			intent: 'add',
			config: {} as never,
			form: { onSubmit: vi.fn() } as never
		};

		builder.enterManualMode(VALID_YAML);

		expect(exitFormState).toHaveBeenCalledOnce();
		expect(builder.isManualMode).toBe(true);
		expect(builder.mode.id).toBe('manual');
		if (builder.mode.id === 'manual') {
			expect(builder.mode.editor.yaml).toBe(VALID_YAML);
			builder.mode.editor.dispose();
		}
	});

	it('exitManualMode returns to idle when not dirty', async () => {
		const builder = createBuilder();
		builder.enterManualMode(VALID_YAML);

		const ok = await builder.exitManualMode();

		expect(ok).toBe(true);
		expect(builder.mode.id).toBe('idle');
	});

	it('enterManualMode with locked sets isManualLocked', () => {
		const builder = createBuilder();
		builder.enterManualMode(VALID_YAML, { locked: true });

		expect(builder.isManualLocked).toBe(true);
		expect(builder.isManualMode).toBe(true);
		if (builder.mode.id === 'manual') builder.mode.editor.dispose();
	});

	it('exitManualMode is no-op when locked', async () => {
		const builder = createBuilder();
		builder.enterManualMode(VALID_YAML, { locked: true });
		if (builder.mode.id !== 'manual') throw new Error('expected manual mode');
		builder.mode.editor.yaml = `${VALID_YAML}\n`;

		const ok = await builder.exitManualMode();

		expect(confirm).not.toHaveBeenCalled();
		expect(ok).toBe(true);
		expect(builder.mode.id).toBe('manual');
		expect(builder.isManualLocked).toBe(true);
		if (builder.mode.id === 'manual') builder.mode.editor.dispose();
	});

	it('exitManualMode clears manualLocked when unlocked', async () => {
		const builder = createBuilder();
		builder.enterManualMode(VALID_YAML);

		const ok = await builder.exitManualMode();

		expect(ok).toBe(true);
		expect(builder.isManualLocked).toBe(false);
		expect(builder.mode.id).toBe('idle');
	});

	it('exitManualMode prompts when dirty and respects confirm', async () => {
		const builder = createBuilder();
		builder.enterManualMode(VALID_YAML);
		if (builder.mode.id !== 'manual') throw new Error('expected manual mode');
		builder.mode.editor.yaml = `${VALID_YAML}\n`;

		vi.mocked(confirm).mockResolvedValue(false);
		const cancelled = await builder.exitManualMode();
		expect(cancelled).toBe(false);
		expect(builder.mode.id).toBe('manual');

		vi.mocked(confirm).mockResolvedValue(true);
		const ok = await builder.exitManualMode();
		expect(ok).toBe(true);
		expect(builder.mode.id).toBe('idle');
	});
});

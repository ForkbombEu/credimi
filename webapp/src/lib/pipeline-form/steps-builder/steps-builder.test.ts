// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { afterEach, describe, expect, it, vi } from 'vitest';

vi.mock('$lib', async () => {
	const { validateYaml } = await import('$lib/pipeline/validate-yaml');
	return { Pipeline: { validateYaml } };
});

vi.mock('./steps-builder.svelte', () => ({ default: class {} }));

import { StepsBuilder } from './steps-builder.svelte.js';

const VALID_YAML = `name: test

steps:
  - use: debug
`;

function createBuilder() {
	return new StepsBuilder({
		steps: [],
		yamlPreview: () => VALID_YAML
	});
}

type BuilderWithState = StepsBuilder & {
	state: { mode: StepsBuilder['mode']; steps: unknown[] };
};

describe('StepsBuilder manual mode', () => {
	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('enterManualMode closes form mode and sets manual', () => {
		const builder = createBuilder();
		const exitFormState = vi.spyOn(builder, 'exitFormState');
		(builder as BuilderWithState).state.mode = {
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

	it('exitManualMode prompts when dirty and respects confirm', async () => {
		const builder = createBuilder();
		builder.enterManualMode(VALID_YAML);
		if (builder.mode.id !== 'manual') throw new Error('expected manual mode');
		builder.mode.editor.yaml = `${VALID_YAML}\n`;

		vi.stubGlobal('confirm', vi.fn(() => false));
		const cancelled = await builder.exitManualMode();
		expect(cancelled).toBe(false);
		expect(builder.mode.id).toBe('manual');

		vi.stubGlobal('confirm', vi.fn(() => true));
		const ok = await builder.exitManualMode();
		expect(ok).toBe(true);
		expect(builder.mode.id).toBe('idle');
	});
});

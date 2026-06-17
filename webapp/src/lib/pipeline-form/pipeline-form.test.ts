// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { afterEach, describe, expect, it, vi } from 'vitest';

vi.mock('$app/navigation', () => ({ beforeNavigate: vi.fn() }));
vi.mock('$lib', async () => {
	const { validateYaml } = await import('$lib/pipeline/validate-yaml');
	return { Pipeline: { validateYaml } };
});
vi.mock('./pipeline-form.svelte', () => ({ default: class {} }));
vi.mock('./steps-builder/steps-builder.svelte', () => ({ default: class {} }));
vi.mock('./metadata-form/metadata-form.svelte', () => ({ default: class {} }));
vi.mock('./runtime-options-form/runtime-options-form.svelte', () => ({ default: class {} }));
vi.mock('./execution-target/index.js', () => ({
	ExecutionTarget: { loadFromPipeline: vi.fn(), clear: vi.fn() }
}));

import { PipelineForm } from './pipeline-form.svelte.js';

const STORED_YAML = `name: manual-pipeline

steps:
  - use: debug
`;

const record = {
	id: 'rec1',
	name: 'manual-pipeline',
	description: '',
	yaml: STORED_YAML,
	manual: true
} as never;

describe('PipelineForm locked manual init', () => {
	afterEach(() => {
		vi.clearAllMocks();
	});

	it('auto-enters locked manual mode when pipeline.manual is true', () => {
		const form = new PipelineForm({
			mode: 'edit',
			pipeline: { record, steps: [], runtime: undefined }
		});

		expect(form.stepsBuilder.isManualMode).toBe(true);
		expect(form.stepsBuilder.isManualLocked).toBe(true);
		if (form.stepsBuilder.mode.id === 'manual') {
			expect(form.stepsBuilder.mode.editor.yaml).toBe(STORED_YAML);
			form.stepsBuilder.mode.editor.dispose();
		}
	});

	it('auto-enters locked manual mode when startLockedManual is true', () => {
		const form = new PipelineForm({
			mode: 'edit',
			pipeline: {
				record: {
					id: 'rec1',
					name: 'manual-pipeline',
					description: '',
					yaml: STORED_YAML,
					manual: false
				} as never,
				steps: [],
				runtime: undefined
			},
			startLockedManual: true
		});

		expect(form.stepsBuilder.isManualMode).toBe(true);
		expect(form.stepsBuilder.isManualLocked).toBe(true);
		if (form.stepsBuilder.mode.id === 'manual') {
			expect(form.stepsBuilder.mode.editor.yaml).toBe(STORED_YAML);
			form.stepsBuilder.mode.editor.dispose();
		}
	});
});

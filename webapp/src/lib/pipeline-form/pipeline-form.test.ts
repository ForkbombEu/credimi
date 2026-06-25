// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

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
	ExecutionTarget: { loadFromPipeline: vi.fn(), clear: vi.fn(), syncFromSteps: vi.fn() }
}));
vi.mock('@/pocketbase/index.js', () => ({
	pb: {
		collection: vi.fn(() => ({
			create: vi.fn().mockResolvedValue({}),
			update: vi.fn().mockResolvedValue({})
		}))
	}
}));
vi.mock('@/i18n', async (importOriginal) => {
	const actual = await importOriginal<typeof import('@/i18n')>();
	return {
		...actual,
		goto: vi.fn().mockResolvedValue(undefined)
	};
});
vi.mock('$lib/utils/index.js', () => ({
	runWithLoading: vi.fn(({ fn }: { fn: () => Promise<void> }) => fn())
}));
vi.mock('svelte-sonner', () => ({
	toast: { error: vi.fn() }
}));
vi.mock('$lib/layout/global-confirm.svelte', () => ({
	confirm: vi.fn()
}));

import { confirm } from '$lib/layout/global-confirm.svelte';
import { goto } from '@/i18n';
import { pb } from '@/pocketbase/index.js';

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
		expect(form.stepsBuilder.isSavedManualPipeline).toBe(true);
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

const METADATA = {
	name: 'test-pipeline',
	description: 'A test pipeline'
};

function createFormWithMetadata(
	props: ConstructorParameters<typeof PipelineForm>[0]
): PipelineForm {
	return new PipelineForm({
		pipeline: { record: METADATA as never, steps: [], runtime: undefined },
		...props
	});
}

function enterDirtyManualMode(form: PipelineForm, yaml = STORED_YAML) {
	form.stepsBuilder.enterManualMode(yaml);
	if (form.stepsBuilder.mode.id !== 'manual') throw new Error('expected manual mode');
	form.stepsBuilder.mode.editor.yaml = `${yaml}\n`;
}

describe('PipelineForm manual save warning', () => {
	const create = vi.fn().mockResolvedValue({});
	const update = vi.fn().mockResolvedValue({});

	beforeEach(() => {
		vi.mocked(pb.collection).mockReturnValue({
			create,
			update
		} as never);
	});

	afterEach(() => {
		vi.clearAllMocks();
	});

	it('create + manual mode + confirm cancel does not persist', async () => {
		vi.mocked(confirm).mockResolvedValue(false);

		const form = createFormWithMetadata({ mode: 'create' });
		enterDirtyManualMode(form);

		await form.save();

		expect(confirm).toHaveBeenCalledOnce();
		expect(create).not.toHaveBeenCalled();
		expect(update).not.toHaveBeenCalled();
		if (form.stepsBuilder.mode.id === 'manual') form.stepsBuilder.mode.editor.dispose();
	});

	it('create + manual mode + confirm OK persists', async () => {
		vi.mocked(confirm).mockResolvedValue(true);

		const form = createFormWithMetadata({ mode: 'create' });
		enterDirtyManualMode(form);

		await form.save();

		expect(confirm).toHaveBeenCalledOnce();
		expect(create).toHaveBeenCalledOnce();
		expect(goto).toHaveBeenCalledWith('/my/pipelines');
		if (form.stepsBuilder.mode.id === 'manual') form.stepsBuilder.mode.editor.dispose();
	});

	it('edit blocks pipeline + manual mode + confirm OK updates with warning', async () => {
		vi.mocked(confirm).mockResolvedValue(true);

		const form = new PipelineForm({
			mode: 'edit',
			pipeline: {
				record: {
					id: 'rec1',
					...METADATA,
					yaml: STORED_YAML,
					manual: false
				} as never,
				steps: [],
				runtime: undefined
			}
		});
		enterDirtyManualMode(form);

		await form.save();

		expect(confirm).toHaveBeenCalledOnce();
		expect(update).toHaveBeenCalledOnce();
		expect(create).not.toHaveBeenCalled();
		if (form.stepsBuilder.mode.id === 'manual') form.stepsBuilder.mode.editor.dispose();
	});

	it('edit manual:true pipeline re-save does not confirm', async () => {
		vi.mocked(confirm).mockResolvedValue(true);

		const form = new PipelineForm({
			mode: 'edit',
			pipeline: { record, steps: [], runtime: undefined }
		});
		if (form.stepsBuilder.mode.id !== 'manual') throw new Error('expected manual mode');
		form.stepsBuilder.mode.editor.yaml = `${STORED_YAML}\n`;

		await form.save();

		expect(confirm).not.toHaveBeenCalled();
		expect(update).toHaveBeenCalledOnce();
		form.stepsBuilder.mode.editor.dispose();
	});
});

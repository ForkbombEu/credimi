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

vi.mock('../steps/wallet-action/index.js', () => ({
	walletActionStepConfig: {
		serialize: (data: { version: string | { __canonified_path__: string } }) => ({
			action_id: 'org/w-a/action',
			version_id:
				data.version === 'installed_from_external_source'
					? 'installed_from_external_source'
					: (data.version as { __canonified_path__: string }).__canonified_path__
		})
	}
}));

import type { PipelineStepByType } from '$lib/pipeline/types.js';
import type { EnrichedStep } from '$pipeline-form/shared/enriched-step.js';
import type { WalletActionStepData } from '$pipeline-form/steps/wallet-action/types.js';

import { confirm } from '$lib/layout/global-confirm.svelte';
import { EXTERNAL_VERSION, GLOBAL_RUNNER } from '$pipeline-form/execution-target/types.js';

import { getBulkWalletVersionContext } from './_partials/index.js';
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

type BuilderInternal = {
	state: { mode: StepsBuilder['mode']; steps: unknown[] };
};

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

	it('yamlPreview tracks upstream yaml changes', () => {
		let name = 'first-pipeline';
		const builder = new StepsBuilder({
			steps: [],
			yamlPreview: () => `name: ${name}\n\nsteps:\n  - use: debug\n`
		});

		expect(builder.yamlPreview).toContain('first-pipeline');

		name = 'second-pipeline';

		expect(builder.yamlPreview).toContain('second-pipeline');
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

describe('StepsBuilder form mode', () => {
	function setFormMode(builder: StepsBuilder) {
		(builder as unknown as BuilderInternal).state.mode = {
			id: 'form',
			intent: 'add',
			config: {} as never,
			form: { onSubmit: vi.fn() } as never
		};
	}

	it('exposes isFormMode when form panel is open', () => {
		const builder = createBuilder();

		expect(builder.isFormMode).toBe(false);

		setFormMode(builder);

		expect(builder.isFormMode).toBe(true);
	});

	it('blocks clone, delete, and reorder while in form mode', () => {
		const builder = createBuilder();
		builder.addDebugStep();
		builder.addDebugStep();
		const initialLength = builder.steps.length;

		setFormMode(builder);

		builder.deleteStep(0);
		builder.cloneStep(0);
		builder.shiftStep(0, 1);

		expect(builder.steps).toHaveLength(initialLength);
	});
});

describe('StepsBuilder bulk wallet version sync', () => {
	type MobileAutomationStep = PipelineStepByType<'mobile-automation'>;

	const walletA = { id: 'w-a', name: 'Wallet A' } as never;
	const walletB = { id: 'w-b', name: 'Wallet B' } as never;
	const action = {
		id: 'a1',
		name: 'Action',
		canonified_name: 'action',
		wallet: 'w-a',
		code: ''
	} as never;

	function mobileStep(
		data: WalletActionStepData,
		withPayload: PipelineStepByType<'mobile-automation'>['with']
	): EnrichedStep {
		return [
			{
				use: 'mobile-automation',
				with: withPayload
			} as PipelineStepByType<'mobile-automation'>,
			data as never
		];
	}

	function stepData(
		wallet: typeof walletA,
		version: WalletActionStepData['version']
	): WalletActionStepData {
		return { wallet, version, runner: GLOBAL_RUNNER, action };
	}

	it('updates all mobile-automation steps with the same wallet and re-serializes with', () => {
		const oldVersion = EXTERNAL_VERSION;
		const newVersion = {
			id: 'v2',
			tag: '2.0',
			__canonified_path__: 'org/w-a/v2'
		} as unknown as WalletActionStepData['version'];

		const step1 = mobileStep(stepData(walletA, oldVersion), {
			action_id: 'org/w-a/action',
			version_id: EXTERNAL_VERSION
		});
		const step2 = mobileStep(stepData(walletA, oldVersion), {
			action_id: 'org/w-a/action',
			version_id: EXTERNAL_VERSION
		});
		const debugStep: EnrichedStep = [{ use: 'debug' }, {} as never];

		const original = [step1, step2, debugStep];
		expect(getBulkWalletVersionContext(original)).not.toBeNull();

		const builder = new StepsBuilder({
			steps: original,
			yamlPreview: () => VALID_YAML
		});

		builder.applyBulkWalletVersion(newVersion);

		const result = builder.steps;
		expect(result[0]![1]).toMatchObject({ version: newVersion });
		expect(result[1]![1]).toMatchObject({ version: newVersion });
		expect((result[0]![0] as MobileAutomationStep).with).toEqual({
			action_id: 'org/w-a/action',
			version_id: 'org/w-a/v2'
		});
		expect((result[1]![0] as MobileAutomationStep).with).toEqual({
			action_id: 'org/w-a/action',
			version_id: 'org/w-a/v2'
		});
		expect(result[2]).toBe(debugStep);
	});

	it('does not apply bulk version when mobile steps use different wallets', () => {
		const walletAStep = mobileStep(stepData(walletA, EXTERNAL_VERSION), {
			action_id: 'org/w-a/action',
			version_id: EXTERNAL_VERSION
		});
		const otherWallet = mobileStep(stepData(walletB, EXTERNAL_VERSION), {
			action_id: 'org/w-b/action',
			version_id: EXTERNAL_VERSION
		});
		const steps = [walletAStep, otherWallet];

		expect(getBulkWalletVersionContext(steps)).toBeNull();

		const builder = new StepsBuilder({
			steps,
			yamlPreview: () => VALID_YAML
		});

		builder.applyBulkWalletVersion({
			id: 'v2',
			tag: '2.0',
			__canonified_path__: 'org/w-a/v2'
		} as never);

		expect(builder.steps).toBe(steps);
	});
});

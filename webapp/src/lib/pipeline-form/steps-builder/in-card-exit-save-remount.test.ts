// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * In-card SAVE used to jump card height; dismiss could shrink-exit.
 *
 * Causal chain (was confirmed):
 * 1. `{#each ... (step)}` keyed by object identity.
 * 2. Save runs mutative `applyEdit*` which replaces the step tuple → new each key
 *    → StepCard remounts → `useHeldFormMode` last is wiped → `showFormBody=false`.
 * 3. Exit `$effect` in step-card-display requires `!editing && showFormBody`;
 *    remount skips it → structural collapse (no `playInCardExit`).
 * 4. Dismiss only sets mode idle → same tuple identity → held survives → exit runs.
 *
 * Fix: parallel `stepKeys` / `followUpKeys` as `{#each}` keys — they survive applyEdit
 * and still move with reorder/delete. Exit layout (`playInCardExitLayout`) is orthogonal.
 *
 * Mount coverage of display enter/exit lives in `step-card-display.svelte.test.ts`.
 */
import { create } from 'mutative';
import { describe, expect, it } from 'vitest';

type StepTuple = [
	{ use: string; id: string; with: Record<string, string> },
	Record<string, string>
];

type BuilderSlice = {
	steps: StepTuple[];
	/** Production `{#each}` key is `builder.stepKeys[index]`, not the tuple. */
	stepKeys: string[];
	mode: { id: 'form' | 'idle'; intent?: string; stepIndex?: number };
};

function seed(): BuilderSlice {
	return {
		steps: [
			[
				{ use: 'http-request', id: '', with: { url: 'https://a.example' } },
				{ url: 'https://a.example' }
			]
		],
		stepKeys: ['card-key-0'],
		mode: { id: 'form', intent: 'edit', stepIndex: 0 }
	};
}

/** Mirrors `applyEditStep` + `mode = idle` inside `form.onSubmit` (keys untouched). */
function savePath(state: BuilderSlice): BuilderSlice {
	return create(state, (draft) => {
		const tuple = draft.steps[0];
		if (!tuple) return;
		tuple[0].with = { url: 'https://b.example' };
		tuple[1] = { url: 'https://b.example' };
		draft.mode = { id: 'idle' };
	});
}

/** Mirrors `exitFormState` (dismiss) — mode only. */
function dismissPath(state: BuilderSlice): BuilderSlice {
	return create(state, (draft) => {
		draft.mode = { id: 'idle' };
	});
}

describe('in-card exit vs save remount', () => {
	it('save replaces the step tuple; dismiss does not', () => {
		const before = seed();
		const keyed = before.steps[0];

		const afterDismiss = dismissPath(before);
		expect(afterDismiss.steps[0]).toBe(keyed);

		const afterSave = savePath(before);
		// Tuple identity still changes under mutative applyEdit — that is why
		// `{#each}` must not key by `(step)`.
		expect(afterSave.steps[0]).not.toBe(keyed);
	});

	it('save keeps parallel stepKeys identity used as {#each} key', () => {
		const before = seed();
		const eachKey = before.stepKeys[0];
		const afterSave = savePath(before);

		// `{#each builder.steps as step, index (builder.stepKeys[index])}`
		expect(afterSave.stepKeys[0]).toBe(eachKey);
		expect(afterSave.steps[0]).not.toBe(before.steps[0]);
	});
});

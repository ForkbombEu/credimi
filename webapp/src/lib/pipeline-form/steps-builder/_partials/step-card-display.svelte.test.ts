// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Enter-ready and save/remount through Step-card display (no duplicated `$effect` predicate).
 */
import type { Snippet } from 'svelte';

import type { EnrichedStep } from '$pipeline-form/shared/enriched-step.js';

import { createRawSnippet } from 'svelte';
import { render } from 'vitest-browser-svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const playInCardEnterLayout = vi.hoisted(() => vi.fn());
const playInCardExitLayout = vi.hoisted(() => vi.fn());

vi.mock('../in-card-layout.js', async (importOriginal) => {
	const orig = await importOriginal<typeof import('../in-card-layout.js')>();
	return {
		...orig,
		playInCardEnterLayout,
		playInCardExitLayout
	};
});

import StepCardDisplay from './step-card-display.svelte';

const debugStep: EnrichedStep = [{ use: 'debug' }, {} as never];

const formBody = createRawSnippet(() => ({
	render: () => `<div data-testid="stub-form">form</div>`,
	setup: () => {}
})) as Snippet;

type Handle = { cancel: ReturnType<typeof vi.fn>; finished: Promise<void> };

function motionHandle(): Handle {
	return { cancel: vi.fn(), finished: Promise.resolve() };
}

beforeEach(() => {
	playInCardEnterLayout.mockReset();
	playInCardExitLayout.mockReset();
	playInCardEnterLayout.mockImplementation((opts: { onSettled?: () => void | Promise<void> }) => {
		void opts.onSettled?.();
		return motionHandle();
	});
	playInCardExitLayout.mockImplementation((opts: { onComplete?: () => void | Promise<void> }) => {
		void opts.onComplete?.();
		return motionHandle();
	});
});

afterEach(() => {
	document.body.replaceChildren();
});

async function renderDisplay(props: {
	editing: boolean;
	showFormBody: boolean;
	expandReady: boolean;
	onExitComplete: () => void;
}) {
	return render(StepCardDisplay, {
		step: debugStep,
		formBody,
		editing: props.editing,
		showFormBody: props.showFormBody,
		expandReady: props.expandReady,
		onExitComplete: props.onExitComplete,
		maxHeightPx: 480
	});
}

describe('step-card-display enter-ready', () => {
	it('does not expand while aligning (enter-ready false)', async () => {
		await renderDisplay({
			editing: true,
			showFormBody: true,
			expandReady: false,
			onExitComplete: vi.fn()
		});
		expect(playInCardEnterLayout).not.toHaveBeenCalled();
		expect(playInCardExitLayout).not.toHaveBeenCalled();
	});

	it('expands in still while editing (enter-ready true)', async () => {
		await renderDisplay({
			editing: true,
			showFormBody: true,
			expandReady: true,
			onExitComplete: vi.fn()
		});
		await vi.waitFor(() => expect(playInCardEnterLayout).toHaveBeenCalledTimes(1));
		expect(playInCardExitLayout).not.toHaveBeenCalled();
	});
});

describe('step-card-display save vs remount', () => {
	it('save with stable identity still reaches playInCardExitLayout / onExitComplete', async () => {
		const onExitComplete = vi.fn();
		const screen = await renderDisplay({
			editing: true,
			showFormBody: true,
			expandReady: true,
			onExitComplete
		});
		await vi.waitFor(() => expect(playInCardEnterLayout).toHaveBeenCalledTimes(1));

		await screen.rerender({ editing: false, showFormBody: true, expandReady: false });
		await vi.waitFor(() => expect(playInCardExitLayout).toHaveBeenCalledTimes(1));
		await vi.waitFor(() => expect(onExitComplete).toHaveBeenCalledTimes(1));
	});

	it('remount with wiped held form does not unlock while editing', async () => {
		const onExitComplete = vi.fn();
		const screen = await renderDisplay({
			editing: true,
			showFormBody: true,
			expandReady: true,
			onExitComplete
		});
		await vi.waitFor(() => expect(playInCardEnterLayout).toHaveBeenCalledTimes(1));
		await screen.unmount();

		expect(onExitComplete).not.toHaveBeenCalled();
		expect(playInCardExitLayout).not.toHaveBeenCalled();

		playInCardEnterLayout.mockClear();
		await renderDisplay({
			editing: true,
			showFormBody: false,
			expandReady: false,
			onExitComplete
		});
		expect(playInCardEnterLayout).not.toHaveBeenCalled();
		expect(playInCardExitLayout).not.toHaveBeenCalled();
		expect(onExitComplete).not.toHaveBeenCalled();
	});
});

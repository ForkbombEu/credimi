// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Enter-ready and save/remount through In-card host (no duplicated `$effect` predicate).
 */
import type { Snippet } from 'svelte';

import type { EnrichedStep } from '$pipeline-form/shared/enriched-step.js';
import type { StepsBuilder } from '$pipeline-form/steps-builder/steps-builder.svelte.js';

import { createRawSnippet } from 'svelte';
import { render } from 'vitest-browser-svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const playInCardEnterLayout = vi.hoisted(() => vi.fn());
const playInCardExitLayout = vi.hoisted(() => vi.fn());

vi.mock('./in-card-layout.js', async (importOriginal) => {
	const orig = await importOriginal<typeof import('./in-card-layout.js')>();
	return {
		...orig,
		playInCardEnterLayout,
		playInCardExitLayout
	};
});

import StubForm from './in-card-host-form-stub.svelte';
import InCardHost from './in-card-host.svelte';

const debugStep: EnrichedStep = [{ use: 'debug' }, {} as never];

const topRight = createRawSnippet(() => ({
	render: () => `<span data-testid="idle-actions">actions</span>`,
	setup: () => {}
})) as Snippet;

type Handle = { cancel: ReturnType<typeof vi.fn>; finished: Promise<void> };

function motionHandle(): Handle {
	return { cancel: vi.fn(), finished: Promise.resolve() };
}

function stubBuilder(kind: 'form' | 'idle'): StepsBuilder {
	const form = {
		canSave: () => true,
		commit: vi.fn(),
		Component: StubForm
	};
	return {
		mode: kind === 'form' ? { id: 'form', intent: 'edit', config: {}, form } : { id: 'idle' },
		exitFormState: vi.fn()
	} as unknown as StepsBuilder;
}

beforeEach(() => {
	playInCardEnterLayout.mockReset();
	playInCardExitLayout.mockReset();
	playInCardEnterLayout.mockImplementation(
		(opts: { onSettled?: (result: { bodyMaxPx: number }) => void | Promise<void> }) => {
			void opts.onSettled?.({ bodyMaxPx: 480 });
			return motionHandle();
		}
	);
	playInCardExitLayout.mockImplementation((opts: { onComplete?: () => void | Promise<void> }) => {
		void opts.onComplete?.();
		return motionHandle();
	});
});

afterEach(() => {
	document.body.replaceChildren();
});

async function renderHost(props: {
	editing: boolean;
	expandReady: boolean;
	builder?: StepsBuilder;
	onExitUnlock?: () => void;
	faded?: boolean;
	topRight?: Snippet;
}) {
	return render(InCardHost, {
		step: debugStep,
		builder: props.builder ?? stubBuilder('form'),
		editing: props.editing,
		expandReady: props.expandReady,
		onExitUnlock: props.onExitUnlock,
		faded: props.faded,
		cardFillMaxPx: 480,
		topRight: props.topRight ?? topRight
	});
}

describe('in-card-host enter-ready', () => {
	it('does not expand while aligning (enter-ready false)', async () => {
		await renderHost({
			editing: true,
			expandReady: false,
			onExitUnlock: vi.fn()
		});
		expect(playInCardEnterLayout).not.toHaveBeenCalled();
		expect(playInCardExitLayout).not.toHaveBeenCalled();
	});

	it('expands in still while editing (enter-ready true)', async () => {
		await renderHost({
			editing: true,
			expandReady: true,
			onExitUnlock: vi.fn()
		});
		await vi.waitFor(() => expect(playInCardEnterLayout).toHaveBeenCalledTimes(1));
		expect(playInCardExitLayout).not.toHaveBeenCalled();
	});
});

describe('in-card-host faded chrome', () => {
	it('dims the card with pointer-events-none but keeps topRight pointer-events-auto', async () => {
		const pencilTopRight = createRawSnippet(() => ({
			render: () =>
				`<button type="button" data-testid="pencil" class="pointer-events-auto">edit</button>`,
			setup: () => {}
		})) as Snippet;

		const screen = await renderHost({
			editing: false,
			expandReady: false,
			faded: true,
			topRight: pencilTopRight
		});

		const root = screen.container.firstElementChild;
		expect(root?.className).toMatch(/pointer-events-none/);
		expect(root?.className).toMatch(/opacity-40/);
		const pencil = screen.container.querySelector('[data-testid="pencil"]');
		expect(pencil?.className).toMatch(/pointer-events-auto/);
	});
});

describe('in-card-host save vs remount', () => {
	it('save with stable identity still reaches playInCardExitLayout / onExitUnlock', async () => {
		const onExitUnlock = vi.fn();
		const builder = stubBuilder('form');
		const screen = await renderHost({
			editing: true,
			expandReady: true,
			builder,
			onExitUnlock
		});
		await vi.waitFor(() => expect(playInCardEnterLayout).toHaveBeenCalledTimes(1));

		await screen.rerender({ editing: false, expandReady: false });
		await vi.waitFor(() => expect(playInCardExitLayout).toHaveBeenCalledTimes(1));
		await vi.waitFor(() => expect(onExitUnlock).toHaveBeenCalledTimes(1));
	});

	it('remount with wiped held form does not unlock while editing', async () => {
		const onExitUnlock = vi.fn();
		const screen = await renderHost({
			editing: true,
			expandReady: true,
			onExitUnlock
		});
		await vi.waitFor(() => expect(playInCardEnterLayout).toHaveBeenCalledTimes(1));
		await screen.unmount();

		expect(onExitUnlock).not.toHaveBeenCalled();
		expect(playInCardExitLayout).not.toHaveBeenCalled();

		playInCardEnterLayout.mockClear();
		await renderHost({
			editing: true,
			expandReady: false,
			builder: stubBuilder('idle'),
			onExitUnlock
		});
		expect(playInCardEnterLayout).not.toHaveBeenCalled();
		expect(playInCardExitLayout).not.toHaveBeenCalled();
		expect(onExitUnlock).not.toHaveBeenCalled();
	});
});

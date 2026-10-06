// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { tick } from 'svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const playInCardEnterLayout = vi.hoisted(() => vi.fn());
const playInCardExitLayout = vi.hoisted(() => vi.fn());

vi.mock('./in-card-layout.js', () => ({
	playInCardEnterLayout,
	playInCardExitLayout
}));

import { createInCardLayoutMachine, type InCardLayoutEls } from './in-card-host-machine.svelte.js';

type FakeEl = {
	className: string;
	style: Record<string, string>;
	getBoundingClientRect: () => { height: number };
};

function fakeEl(height = 0, naturalHeight = 120, className = ''): HTMLElement {
	const el: FakeEl = {
		className,
		style: { height: `${height}px` },
		getBoundingClientRect: () => {
			if (el.style.height === 'auto') return { height: naturalHeight };
			const parsed = parseFloat(el.style.height || '0');
			if (Number.isFinite(parsed) && el.style.height !== '') return { height: parsed };
			return { height };
		}
	};
	return el as unknown as HTMLElement;
}

function fakeEls(): InCardLayoutEls {
	return {
		card: fakeEl(120, 120),
		lock: fakeEl(80, 80),
		display: fakeEl(80, 80),
		form: fakeEl(0, 200)
	};
}

type Handle = { cancel: ReturnType<typeof vi.fn>; finished: Promise<void> };

function motionHandle(): Handle {
	return { cancel: vi.fn(), finished: Promise.resolve() };
}

type EnterOpts = {
	onSettled?: (info: { bodyMaxPx: number | undefined }) => void | Promise<void>;
	cardFillMaxPx?: number;
};

type ExitOpts = {
	enterSettled: boolean;
	onComplete?: () => void | Promise<void>;
};

describe('createInCardLayoutMachine', () => {
	let els: InCardLayoutEls | null;
	let onDone: ReturnType<typeof vi.fn>;
	let machine: ReturnType<typeof createInCardLayoutMachine>;

	beforeEach(() => {
		els = fakeEls();
		onDone = vi.fn();
		playInCardEnterLayout.mockReset();
		playInCardExitLayout.mockReset();
		playInCardEnterLayout.mockImplementation(() => motionHandle());
		playInCardExitLayout.mockImplementation(() => motionHandle());
		machine = createInCardLayoutMachine({
			getEls: () => els,
			getCardFillMaxPx: () => 480,
			onDone
		});
	});

	afterEach(() => {
		machine.destroy();
	});

	it('hold only → enter not called, current waiting', () => {
		machine.sync({ showFormBody: true, expandReady: false, editing: true });
		expect(machine.current).toBe('waiting');
		expect(playInCardEnterLayout).not.toHaveBeenCalled();
	});

	it('hold + ready with els → enter once, current entering', () => {
		machine.sync({ showFormBody: true, expandReady: true, editing: true });
		expect(machine.current).toBe('entering');
		expect(playInCardEnterLayout).toHaveBeenCalledOnce();
		expect(playInCardEnterLayout.mock.calls[0]?.[0]).toEqual(
			expect.objectContaining({
				card: els?.card,
				lock: els?.lock,
				display: els?.display,
				form: els?.form,
				cardFillMaxPx: 480
			})
		);
	});

	it('ready twice → enter still once', () => {
		machine.sync({ showFormBody: true, expandReady: true, editing: true });
		machine.sync({ showFormBody: true, expandReady: true, editing: true });
		expect(machine.current).toBe('entering');
		expect(playInCardEnterLayout).toHaveBeenCalledOnce();
	});

	it('fire onSettled → settled, bodyMaxPx set', () => {
		machine.sync({ showFormBody: true, expandReady: true, editing: true });
		const enterOpts = playInCardEnterLayout.mock.calls[0]?.[0] as EnterOpts;
		enterOpts.onSettled?.({ bodyMaxPx: 440 });
		expect(machine.current).toBe('settled');
		expect(machine.bodyMaxPx).toBe(440);
	});

	it('from settled, stop editing → exit once enterSettled true', async () => {
		machine.sync({ showFormBody: true, expandReady: true, editing: true });
		const enterOpts = playInCardEnterLayout.mock.calls[0]?.[0] as EnterOpts;
		enterOpts.onSettled?.({ bodyMaxPx: 440 });
		machine.sync({ showFormBody: true, expandReady: false, editing: false });
		expect(machine.current).toBe('exiting');
		expect(playInCardExitLayout).not.toHaveBeenCalled();
		await tick();
		expect(playInCardExitLayout).toHaveBeenCalledOnce();
		expect(playInCardExitLayout.mock.calls[0]?.[0]).toEqual(
			expect.objectContaining({ enterSettled: true })
		);
	});

	it('ready then release before settled → enter cancelled, exit enterSettled false', () => {
		machine.sync({ showFormBody: true, expandReady: true, editing: true });
		const enterHandle = playInCardEnterLayout.mock.results[0]?.value as Handle;
		machine.sync({ showFormBody: true, expandReady: false, editing: false });
		expect(enterHandle.cancel).toHaveBeenCalled();
		expect(playInCardExitLayout).toHaveBeenCalledOnce();
		expect(playInCardExitLayout.mock.calls[0]?.[0]).toEqual(
			expect.objectContaining({ enterSettled: false })
		);
		expect(machine.current).toBe('exiting');
	});

	it('drop while waiting → idle, no exit', () => {
		machine.sync({ showFormBody: true, expandReady: false, editing: true });
		machine.sync({ showFormBody: false, expandReady: false, editing: false });
		expect(machine.current).toBe('idle');
		expect(playInCardExitLayout).not.toHaveBeenCalled();
		expect(onDone).not.toHaveBeenCalled();
	});

	it('destroy while entering → cancel, idle, onDone not called', () => {
		machine.sync({ showFormBody: true, expandReady: true, editing: true });
		const enterHandle = playInCardEnterLayout.mock.results[0]?.value as Handle;
		machine.destroy();
		expect(enterHandle.cancel).toHaveBeenCalled();
		expect(machine.current).toBe('idle');
		expect(onDone).not.toHaveBeenCalled();
	});

	it('exit onComplete → onDone once, idle', () => {
		machine.sync({ showFormBody: true, expandReady: false, editing: false });
		expect(machine.current).toBe('exiting');
		const exitOpts = playInCardExitLayout.mock.calls[0]?.[0] as ExitOpts;
		exitOpts.onComplete?.();
		expect(onDone).toHaveBeenCalledOnce();
		expect(machine.current).toBe('idle');
		exitOpts.onComplete?.();
		expect(onDone).toHaveBeenCalledOnce();
	});
});

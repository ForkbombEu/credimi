// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Layout outcome for Save-pin during In-card enter vs settled.
 * Seam: `inCardFormHostClass` on the form host + shell flex structure from
 * `in-card-form-shell.svelte` (not token-string trivia).
 */
import { afterEach, describe, expect, it } from 'vitest';

import '../../../routes/layout.css';
import { inCardFormHostClass } from './in-card-layout.js';

/** Tall lock, short form — the Save-bar jump case during absolute-fill enter. */
const LOCK_HEIGHT_PX = 280;

type Fixture = {
	root: HTMLElement;
	host: HTMLElement;
	save: HTMLElement;
};

function mountSavePinFixture(settled: boolean): Fixture {
	const root = document.createElement('div');
	// Lock mirrors step-card-display: relative; settled lock is a flex column so grow works.
	const lock = document.createElement('div');
	lock.dataset.testid = 'in-card-body-lock';
	lock.className = settled ? 'relative flex min-h-0 flex-col' : 'relative';
	lock.style.height = `${LOCK_HEIGHT_PX}px`;
	lock.style.width = '320px';

	const host = document.createElement('div');
	host.dataset.testid = 'in-card-form-host';
	host.className = inCardFormHostClass(settled);

	// Shell classes match in-card-form-shell.svelte (region / body / save row).
	const shell = document.createElement('div');
	shell.dataset.testid = 'in-card-form-shell';
	shell.className = 'flex min-h-0 grow flex-col';

	const body = document.createElement('div');
	body.dataset.testid = 'in-card-form-body';
	body.className = 'min-h-0 grow overflow-y-auto';
	const short = document.createElement('p');
	short.textContent = 'short form fields';
	body.appendChild(short);

	const save = document.createElement('div');
	save.dataset.testid = 'in-card-form-save';
	save.className = 'shrink-0 border-t p-3';
	save.textContent = 'Save';

	shell.append(body, save);
	host.appendChild(shell);
	lock.appendChild(host);
	root.appendChild(lock);
	document.body.appendChild(root);
	return { root, host, save };
}

afterEach(() => {
	document.body.replaceChildren();
});

describe('in-card Save pin (layout)', () => {
	it.each([
		{ settled: false, label: 'enter absolute fill' },
		{ settled: true, label: 'settled grow column' }
	] as const)(
		'keeps Save at host bottom when lock is taller than form ($label)',
		({ settled }) => {
			const { host, save } = mountSavePinFixture(settled);

			const hostBottom = host.getBoundingClientRect().bottom;
			const saveBottom = save.getBoundingClientRect().bottom;
			const hostHeight = host.getBoundingClientRect().height;

			// Host fills the tall lock (enter: absolute inset-0; settled: grow in flex lock).
			expect(hostHeight).toBeGreaterThan(120);
			expect(Math.abs(hostHeight - LOCK_HEIGHT_PX)).toBeLessThan(1);

			// Column pin: Save sits on the form-host bottom, not under short content mid-lock.
			expect(Math.abs(hostBottom - saveBottom)).toBeLessThan(1);
		}
	);
});

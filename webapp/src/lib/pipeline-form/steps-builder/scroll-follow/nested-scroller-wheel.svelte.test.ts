// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { afterEach, describe, expect, it } from 'vitest';

import { nestedScrollerConsumesWheel } from './nested-scroller-wheel.js';

afterEach(() => {
	document.body.replaceChildren();
});

function wheel(target: EventTarget, deltaY: number, path: EventTarget[]): WheelEvent {
	const event = new WheelEvent('wheel', { deltaY, deltaX: 0, bubbles: true, cancelable: true });
	Object.defineProperty(event, 'target', { value: target });
	Object.defineProperty(event, 'composedPath', { value: () => path });
	return event;
}

function overflowingFormBody(): { root: HTMLElement; formBody: HTMLElement; field: HTMLElement } {
	const root = document.createElement('div');
	root.style.cssText = 'height:400px;overflow:hidden';
	const formBody = document.createElement('div');
	formBody.style.cssText = 'height:120px;overflow-y:auto';
	formBody.dataset.testid = 'in-card-form-body';
	const inner = document.createElement('div');
	inner.style.height = '800px';
	const field = document.createElement('input');
	inner.appendChild(field);
	formBody.appendChild(inner);
	root.appendChild(formBody);
	document.body.appendChild(root);
	return { root, formBody, field };
}

describe('nestedScrollerConsumesWheel', () => {
	it('is false when only the parked root would scroll', () => {
		const { root } = overflowingFormBody();
		const dead = document.createElement('div');
		root.appendChild(dead);
		const event = wheel(dead, 40, [dead, root]);
		expect(nestedScrollerConsumesWheel(event, root)).toBe(false);
	});

	it('is true when the in-card form body can scroll with the delta', () => {
		const { root, formBody, field } = overflowingFormBody();
		expect(formBody.scrollHeight).toBeGreaterThan(formBody.clientHeight);
		const event = wheel(field, 40, [field, formBody, root]);
		expect(nestedScrollerConsumesWheel(event, root)).toBe(true);
	});

	it('is false when the nested scroller is already at the end', () => {
		const { root, formBody } = overflowingFormBody();
		formBody.scrollTop = formBody.scrollHeight;
		const event = wheel(formBody, 40, [formBody, root]);
		expect(nestedScrollerConsumesWheel(event, root)).toBe(false);
	});

	it('is true when scrolling up and nested scroller is not at top', () => {
		const { root, formBody } = overflowingFormBody();
		formBody.scrollTop = 40;
		const event = wheel(formBody, -40, [formBody, root]);
		expect(nestedScrollerConsumesWheel(event, root)).toBe(true);
	});
});

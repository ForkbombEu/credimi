// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { HubItem } from '$lib/hub';

import { describe, expect, it, vi } from 'vitest';

vi.mock('./hub-item-step-form.svelte', () => ({ default: class {} }));

import { HubItemStepForm } from './hub-item-step-form.svelte.js';

const hubItem = {
	id: 'h1',
	name: 'Test Credential',
	organization_name: 'Org',
	type: 'credentials'
} as HubItem;

describe('HubItemStepForm', () => {
	it('selectItem commits on add intent', () => {
		const onSubmit = vi.fn();
		const form = new HubItemStepForm(
			{
				collection: 'credentials',
				entityData: { labels: { singular: 'Credential' } } as never
			},
			{ intent: 'add' }
		);
		form.onSubmit(onSubmit);
		form.selectItem(hubItem);
		expect(onSubmit).toHaveBeenCalledOnce();
		expect(onSubmit.mock.calls[0][0]).toBe(hubItem);
	});

	it('selectItem does not commit on edit intent', () => {
		const onSubmit = vi.fn();
		const form = new HubItemStepForm(
			{
				collection: 'credentials',
				entityData: { labels: { singular: 'Credential' } } as never
			},
			{ intent: 'edit', initial: hubItem }
		);
		form.onSubmit(onSubmit);
		form.selectItem({ ...hubItem, id: 'h2', name: 'Other' });
		expect(onSubmit).not.toHaveBeenCalled();
		expect(form.selectedItem?.id).toBe('h2');
	});

	it('discardSelection clears selectedItem', () => {
		const form = new HubItemStepForm(
			{
				collection: 'credentials',
				entityData: { labels: { singular: 'Credential' } } as never
			},
			{ intent: 'edit', initial: hubItem }
		);
		form.discardSelection();
		expect(form.selectedItem).toBeUndefined();
	});
});

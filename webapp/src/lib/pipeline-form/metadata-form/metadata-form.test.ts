// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { afterEach, describe, expect, it, vi } from 'vitest';

vi.mock('@/forms', () => ({
	createForm: vi.fn(({ initialData }: { initialData?: unknown }) => ({
		initialData,
		id: Math.random()
	}))
}));
vi.mock('./metadata-form.svelte', () => ({ default: class {} }));
vi.mock('svelte', () => ({ tick: vi.fn().mockResolvedValue(undefined) }));

import { createForm } from '@/forms';

import { MetadataForm } from './metadata-form.svelte.js';

const INITIAL = {
	name: 'initial-name',
	description: 'initial description',
	published: false
};

describe('MetadataForm mountForm freshness', () => {
	afterEach(() => {
		vi.clearAllMocks();
	});

	it('returns a new superform after resetForm', () => {
		const form = new MetadataForm({ initialData: INITIAL });

		const first = form.mountForm();
		form.resetForm();
		const second = form.mountForm();

		expect(first).not.toBe(second);
	});

	it('remounts with updated value after submit clears cached superform', async () => {
		const form = new MetadataForm({ initialData: INITIAL });
		form.mountForm();

		const saved = {
			name: 'saved-name',
			description: 'saved description',
			published: true
		};

		const onSubmit = createForm.mock.calls.at(-1)?.[0]?.onSubmit;
		expect(onSubmit).toBeTypeOf('function');

		await onSubmit?.({ form: { data: saved, valid: true } });

		form.isOpen = true;
		form.mountForm();

		const lastCall = createForm.mock.calls.at(-1)?.[0];
		expect(lastCall?.initialData).toEqual(saved);
		expect(form.value).toEqual(saved);
	});
});

// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { render } from '@testing-library/svelte';
import { zod4 } from 'sveltekit-superforms/adapters';
import { describe, expect, test } from 'vitest';
import { z } from 'zod/v4';

import FormTest from './form.test.svelte';

//

/* svelte:definitions
  import { Form } from '@/v2/form.svelte.js';
*/

const testSchema = z.object({
	name: z.string().min(1, 'Name is required'),
	email: z.email('Invalid email'),
	age: z.number().min(18, 'Must be 18 or older')
});

type Schema = z.infer<typeof testSchema>;

describe('Form', () => {
	// test('creates form with default values', () => {
	// 	expect(form.superform).toBeDefined();
	// 	expect(form.values).toBeDefined();
	// 	expect(form.validationErrors).toBeDefined();
	// });

	test('initializes with provided initial data', () => {
		render(FormTest<Schema>, {
			adapter: zod4(testSchema),
			initialData: {
				name: 'John',
				email: 'john@example.com',
				age: 25
			},
			onReady: (form) => {
				expect(form.values.current).toEqual({
					name: 'John',
					email: 'john@example.com',
					age: 25
				});
			}
		});

		// const initialData = { name: 'John', email: 'john@example.com', age: 25 };

		// const cleanup = $effect.root(() => {
		// 	const formWithInitialData = new Form({
		// 		adapter: zod4(testSchema),
		// 		initialData
		// 	});

		// 	expect(formWithInitialData.values.current).toEqual(
		// 		expect.objectContaining(initialData)
		// 	);
		// });

		// cleanup();
	});

	// test('submit calls superform submit', () => {
	// 	form.submit();
	// 	expect(onSubmit).toHaveBeenCalled();
	// });

	// test('onSubmit is called when form is valid', async () => {
	// 	// Set valid form data
	// 	form.values.current = {
	// 		name: 'John Doe',
	// 		email: 'john@example.com',
	// 		age: 25
	// 	};

	// 	// Trigger form submission
	// 	await form.superform.submit();

	// 	expect(onSubmit).toHaveBeenCalled();
	// });

	// test('onSubmit is not called when form is invalid', async () => {
	// 	// Set invalid form data
	// 	form.values.current = {
	// 		name: '',
	// 		email: 'invalid-email',
	// 		age: 15
	// 	};

	// 	// Trigger form submission
	// 	await form.superform.submit();

	// 	expect(onSubmit).not.toHaveBeenCalled();
	// });

	// test('handles onSubmit errors and calls onError', async () => {
	// 	const error = new Error('Submission failed');
	// 	onSubmit.mockRejectedValue(error);

	// 	// Set valid form data
	// 	form.values.current = {
	// 		name: 'John Doe',
	// 		email: 'john@example.com',
	// 		age: 25
	// 	};

	// 	// Trigger form submission
	// 	await form.superform.submit();

	// 	expect(onSubmit).toHaveBeenCalled();
	// 	expect(onError).toHaveBeenCalledWith(expect.any(t.BaseError));
	// });

	// test('sets form error when onSubmit throws', async () => {
	// 	const errorMessage = 'Network error';
	// 	onSubmit.mockRejectedValue(new Error(errorMessage));

	// 	// Set valid form data
	// 	form.values.current = {
	// 		name: 'John Doe',
	// 		email: 'john@example.com',
	// 		age: 25
	// 	};

	// 	// Trigger form submission
	// 	await form.superform.submit();

	// 	// Check that form has error set
	// 	expect(form.validationErrors.current._errors).toContain(errorMessage);
	// });

	// test('reactive error property reflects validation errors', () => {
	// 	// Initially no errors
	// 	expect(form.error).toEqual([]);

	// 	// Set some validation errors manually for testing
	// 	form.validationErrors.current = {
	// 		_errors: ['Form error'],
	// 		name: ['Name error'],
	// 		email: ['Email error'],
	// 		age: ['Age error']
	// 	};

	// 	expect(form.error).toEqual(['Form error']);
	// });

	// test('form options are passed to superForm', () => {
	// 	const customOptions = {
	// 		resetForm: false,
	// 		clearOnSubmit: 'errors' as const
	// 	};

	// 	const formWithOptions = new Form({
	// 		adapter: zod4(testSchema),
	// 		options: customOptions
	// 	});

	// 	expect(formWithOptions.superform).toBeDefined();
	// 	// Note: We can't easily test that options were applied without diving into internals
	// 	// This test mainly ensures the form can be created with custom options
	// });

	// test('async onSubmit is handled correctly', async () => {
	// 	const asyncOnSubmit = vi.fn().mockResolvedValue(undefined);

	// 	const asyncForm = new Form({
	// 		adapter: zod4(testSchema),
	// 		onSubmit: asyncOnSubmit
	// 	});

	// 	// Set valid form data
	// 	asyncForm.values.current = {
	// 		name: 'Jane Doe',
	// 		email: 'jane@example.com',
	// 		age: 30
	// 	};

	// 	await asyncForm.superform.submit();

	// 	expect(asyncOnSubmit).toHaveBeenCalled();
	// });

	// test('async onError is handled correctly', async () => {
	// 	const asyncOnError = vi.fn().mockResolvedValue(undefined);
	// 	const failingOnSubmit = vi.fn().mockRejectedValue(new Error('Async error'));

	// 	const asyncForm = new Form({
	// 		adapter: zod4(testSchema),
	// 		onSubmit: failingOnSubmit,
	// 		onError: asyncOnError
	// 	});

	// 	// Set valid form data
	// 	asyncForm.values.current = {
	// 		name: 'Jane Doe',
	// 		email: 'jane@example.com',
	// 		age: 30
	// 	};

	// 	await asyncForm.superform.submit();

	// 	expect(failingOnSubmit).toHaveBeenCalled();
	// 	expect(asyncOnError).toHaveBeenCalledWith(expect.any(t.BaseError));
	// });
});

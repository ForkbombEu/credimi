// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { form, types as t } from '#';
import { mount } from 'svelte';
import { zod4 } from 'sveltekit-superforms/adapters';
import { beforeEach, describe, expect, test, vi } from 'vitest';
import { z } from 'zod/v4';

import FormTestComponent from './form-test.svelte';

//

const schema = z.object({
	name: z.string().min(1, 'Name is required'),
	email: z.email('Invalid email'),
	age: z.number().min(18, 'Must be 18 or older')
});

type Schema = z.infer<typeof schema>;

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function mountComponent<Form extends form.Instance<any>>(form: Form) {
	return mount(FormTestComponent, {
		target: document.body,
		props: { form }
	});
}

describe('Form', () => {
	let theForm: form.Instance<Schema>;
	let fullData: Schema;

	beforeEach(() => {
		theForm = new form.Instance({
			adapter: zod4(schema)
		});
		fullData = {
			name: 'John',
			email: 'john@example.com',
			age: 25
		};
	});

	test('becomes valid when values are set', async () => {
		mountComponent(theForm);
		await theForm.update(fullData);
		expect(theForm.valid).toBe(true);
	});

	test('stays invalid when partial data is set', async () => {
		mountComponent(theForm);
		await theForm.update({ name: 'John' });
		expect(theForm.valid).toBe(false);
	});

	test('initializes with provided initial data', () => {
		theForm = new form.Instance({
			adapter: zod4(schema),
			initialData: fullData
		});
		mountComponent(theForm);
		expect(theForm.values).toEqual(expect.objectContaining(fullData));
	});

	test('reactive dependent is updated when form values change', async () => {
		mountComponent(theForm);
		const doubleAge = $derived.by(() => (theForm.values.age ?? 0) * 2);
		expect(doubleAge).toBe(0);
		await theForm.update({ age: 25 });
		expect(doubleAge).toBe(50);
	});

	test('onSubmit gets called when form is valid', async () => {
		const onSubmit = vi.fn();
		theForm = new form.Instance({
			adapter: zod4(schema),
			onSubmit
		});
		mountComponent(theForm);
		await theForm.update({
			name: 'John',
			email: 'john@example.com',
			age: 25
		});
		expect(theForm.valid).toBe(true);
		await theForm.submit();
		expect(onSubmit).toHaveBeenCalledWith(fullData);
	});

	test('onSubmit is not called when form is invalid', async () => {
		const onSubmit = vi.fn();
		theForm = new form.Instance({
			adapter: zod4(schema),
			onSubmit
		});
		mountComponent(theForm);
		await theForm.update({
			name: 'John',
			age: 25
		});
		expect(theForm.valid).toBe(false);
		await theForm.submit();
		expect(onSubmit).not.toHaveBeenCalled();
	});

	test('handles onSubmit errors and calls onError', async () => {
		const onSubmit = vi.fn().mockRejectedValue(new Error('Submission failed'));
		const onError = vi.fn();

		theForm = new form.Instance({
			adapter: zod4(schema),
			onSubmit,
			onError
		});

		mountComponent(theForm);
		await theForm.update(fullData);
		expect(theForm.valid).toBe(true);

		await theForm.submit();
		expect(onSubmit).toHaveBeenCalled();
		expect(onError).toHaveBeenCalled();
	});

	test('sets form error when onSubmit throws', async () => {
		const errorMessage = 'Network error';
		const onSubmit = () => {
			throw new Error(errorMessage);
		};

		theForm = new form.Instance({
			adapter: zod4(schema),
			onSubmit
		});

		mountComponent(theForm);
		await theForm.update(fullData);
		expect(theForm.valid).toBe(true);

		await theForm.submit();
		expect(theForm.submitError).toBeInstanceOf(t.BaseError);
		expect(theForm.submitError?.message).toBe(errorMessage);
	});

	test('catches schema refinement errors', async () => {
		const errorMessage = 'Must be 100 or older';

		const schema = z
			.object({
				name: z.string().min(1, 'Name is required'),
				email: z.email('Invalid email'),
				age: z.number().min(18, 'Must be 18 or older')
			})
			.refine((data) => data.age > 100, errorMessage);

		theForm = new form.Instance({
			adapter: zod4(schema)
		});

		mountComponent(theForm);
		await theForm.update({ ...fullData, age: 99 });
		expect(theForm.valid).toBe(false);

		const error = theForm.errors.find((e) => e.messages.includes(errorMessage));
		expect(error).toBeDefined();
	});
});

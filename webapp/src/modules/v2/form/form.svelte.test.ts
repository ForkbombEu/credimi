// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { nanoid } from 'nanoid';
import { mount } from 'svelte';
import { zod4 } from 'sveltekit-superforms/adapters';
import { describe, expect, test, vi } from 'vitest';
import { z } from 'zod/v4';

import { form } from '@/v2';

import FormTest from './form-test.svelte';

//

const schema = z.object({
	name: z.string().min(1, 'Name is required'),
	email: z.email('Invalid email'),
	age: z.number().min(18, 'Must be 18 or older')
});

type Schema = z.infer<typeof schema>;

//

type MountFormProps = form.Options<Schema> & {
	onReady?: (form: form.Form<Schema>) => void;
};

function mountForm(props: MountFormProps) {
	return mount(FormTest<Schema>, {
		target: document.body,
		props: {
			adapter: zod4(schema),
			options: { id: nanoid(6) },
			...props
		}
	});
}

//

class ReactiveDependent {
	constructor(readonly form: form.Form<Schema>) {}
	doubleAge = $derived.by(() => this.form.values.current.age * 2);
}

describe('Form', () => {
	test('creates form with default values', () => {
		mountForm({
			onReady: (form) => {
				expect(form.superform).toBeDefined();
				expect(form.values).toBeDefined();
				expect(form.validationErrors).toBeDefined();
			}
		});
	});

	test('initializes with provided initial data', () => {
		const initialData = {
			name: 'John',
			email: 'john@example.com',
			age: 25
		};
		mountForm({
			initialData,
			onReady: (form) => {
				expect(form.values.current).toEqual(expect.objectContaining(initialData));
			}
		});
	});

	test('reactive dependent is updated when form values change', () => {
		mountForm({
			initialData: {
				age: 25
			},
			onReady: (form) => {
				const context = new ReactiveDependent(form);
				expect(context.doubleAge).toBe(50);
			}
		});
	});

	test('submit calls superform submit', () => {
		const onSubmit = vi.fn(() => {});
		mountForm({
			onSubmit,

			initialData: {
				name: 'John',
				email: 'john@example.com',
				age: 25
			},
			onReady: async (form) => {
				form.submit();
				expect(onSubmit).toHaveBeenCalled();
			}
		});
	});

	// test('onSubmit is called when form is valid', async () => {
	// 	const onSubmit = vi.fn();

	// 	render(FormTest<Schema>, {
	// 		adapter: zod4(testSchema),
	// 		onSubmit,
	// 		onReady: async (form) => {
	// 			// Set valid form data
	// 			form.values.current = {
	// 				name: 'John Doe',
	// 				email: 'john@example.com',
	// 				age: 25
	// 			};

	// 			// Trigger form submission
	// 			await form.superform.submit();
	// 			expect(onSubmit).toHaveBeenCalled();
	// 		}
	// 	});
	// });

	// test('onSubmit is not called when form is invalid', async () => {
	// 	const onSubmit = vi.fn();

	// 	render(FormTest<Schema>, {
	// 		adapter: zod4(testSchema),
	// 		onSubmit,
	// 		onReady: async (form) => {
	// 			// Set invalid form data
	// 			form.values.current = {
	// 				name: '',
	// 				email: 'invalid-email',
	// 				age: 15
	// 			};

	// 			// Trigger form submission
	// 			await form.superform.submit();
	// 			expect(onSubmit).not.toHaveBeenCalled();
	// 		}
	// 	});
	// });

	// test('handles onSubmit errors and calls onError', async () => {
	// 	const onSubmit = vi.fn().mockRejectedValue(new Error('Submission failed'));
	// 	const onError = vi.fn();

	// 	render(FormTest<Schema>, {
	// 		adapter: zod4(testSchema),
	// 		onSubmit,
	// 		onError,
	// 		onReady: async (form) => {
	// 			// Set valid form data
	// 			form.values.current = {
	// 				name: 'John Doe',
	// 				email: 'john@example.com',
	// 				age: 25
	// 			};

	// 			// Trigger form submission
	// 			await form.superform.submit();
	// 			expect(onSubmit).toHaveBeenCalled();
	// 			expect(onError).toHaveBeenCalled();
	// 		}
	// 	});
	// });

	// test('sets form error when onSubmit throws', async () => {
	// 	const errorMessage = 'Network error';
	// 	const onSubmit = vi.fn().mockRejectedValue(new Error(errorMessage));

	// 	render(FormTest<Schema>, {
	// 		adapter: zod4(testSchema),
	// 		onSubmit,
	// 		onReady: async (form) => {
	// 			// Set valid form data
	// 			form.values.current = {
	// 				name: 'John Doe',
	// 				email: 'john@example.com',
	// 				age: 25
	// 			};

	// 			// Trigger form submission
	// 			await form.superform.submit();

	// 			// Check that form has error set
	// 			expect(form.validationErrors.current._errors).toContain(errorMessage);
	// 		}
	// 	});
	// });

	// test('reactive error property reflects validation errors', () => {
	// 	render(FormTest<Schema>, {
	// 		adapter: zod4(testSchema),
	// 		onReady: (form) => {
	// 			// Initially no errors
	// 			expect(form.error).toEqual([]);

	// 			// Set some validation errors manually for testing
	// 			form.validationErrors.current = {
	// 				_errors: ['Form error'],
	// 				name: ['Name error'],
	// 				email: ['Email error'],
	// 				age: ['Age error']
	// 			};

	// 			expect(form.error).toEqual(['Form error']);
	// 		}
	// 	});
	// });

	// test('form options are passed to superForm', () => {
	// 	const customOptions = {
	// 		resetForm: false,
	// 		clearOnSubmit: 'errors' as const
	// 	};

	// 	render(FormTest<Schema>, {
	// 		adapter: zod4(testSchema),
	// 		options: customOptions,
	// 		onReady: (form) => {
	// 			expect(form.superform).toBeDefined();
	// 			// Note: We can't easily test that options were applied without diving into internals
	// 			// This test mainly ensures the form can be created with custom options
	// 		}
	// 	});
	// });

	// test('async onSubmit is handled correctly', async () => {
	// 	const asyncOnSubmit = vi.fn().mockResolvedValue(undefined);

	// 	render(FormTest<Schema>, {
	// 		adapter: zod4(testSchema),
	// 		onSubmit: asyncOnSubmit,
	// 		onReady: async (form) => {
	// 			// Set valid form data
	// 			form.values.current = {
	// 				name: 'Jane Doe',
	// 				email: 'jane@example.com',
	// 				age: 30
	// 			};

	// 			await form.superform.submit();
	// 			expect(asyncOnSubmit).toHaveBeenCalled();
	// 		}
	// 	});
	// });

	// test('async onError is handled correctly', async () => {
	// 	const asyncOnError = vi.fn().mockResolvedValue(undefined);
	// 	const failingOnSubmit = vi.fn().mockRejectedValue(new Error('Async error'));

	// 	render(FormTest<Schema>, {
	// 		adapter: zod4(testSchema),
	// 		onSubmit: failingOnSubmit,
	// 		onError: asyncOnError,
	// 		onReady: async (form) => {
	// 			// Set valid form data
	// 			form.values.current = {
	// 				name: 'Jane Doe',
	// 				email: 'jane@example.com',
	// 				age: 30
	// 			};

	// 			await form.superform.submit();
	// 			expect(failingOnSubmit).toHaveBeenCalled();
	// 			expect(asyncOnError).toHaveBeenCalled();
	// 		}
	// 	});
	// });
});

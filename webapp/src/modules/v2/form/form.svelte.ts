// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type * as sf from 'sveltekit-superforms';

import { fromStore } from 'svelte/store';
import { type ValidationAdapter } from 'sveltekit-superforms/adapters';
import { defaults, setError, superForm } from 'sveltekit-superforms/client';

import { types as t } from '@/v2';

//

type SuperformOptions<Data extends t.GenericRecord> = Omit<sf.FormOptions<Data>, 'onUpdate'>;

export type Config<Data extends t.GenericRecord> = {
	adapter: ValidationAdapter<Data>;
	options?: SuperformOptions<Data>;
	onSubmit?: (data: Data) => void | Promise<void>;
	initialData?: Partial<Data>;
	onError?: (error: t.BaseError) => void | Promise<void>;
};

export class Form<Data extends t.GenericRecord> {
	public readonly superform: sf.SuperForm<Data>;

	readonly values: t.State<Data>;
	readonly validationErrors: t.State<sf.ValidationErrors<Data>>;
	readonly error = $derived.by(() => this.validationErrors.current._errors);

	constructor(config: Config<Data>) {
		const { adapter, options, onSubmit = () => {}, initialData, onError = () => {} } = config;
		this.superform = superForm(defaults(initialData, adapter), {
			SPA: true,
			applyAction: false,
			scrollToError: 'smooth',
			validators: adapter,
			dataType: 'json',
			taintedMessage: null,
			...options,
			onUpdate: async (event) => {
				try {
					if (event.form.valid) await onSubmit(event.form.data);
				} catch (e) {
					const error = new t.BaseError(e);
					setError(event.form, error.message);
					await onError(error);
				}
			}
		});
		this.values = fromStore(this.superform.form);
		this.validationErrors = fromStore(this.superform.errors);
	}

	submit() {
		this.superform.submit();
	}
}

// type SubmitFunction<Data extends t.GenericRecord> = NonNullable<sf.FormOptions<Data>['onUpdate']>;

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

export type Options<Data extends t.GenericRecord> = {
	options?: SuperformOptions<Data>;
	onSubmit?: (data: Data) => void | Promise<void>;
	initialData?: Partial<Data>;
	onError?: (error: t.BaseError) => void | Promise<void>;
};

export type Config<Data extends t.GenericRecord> = Options<Data> & {
	adapter: ValidationAdapter<Data>;
};

export class Form<Data extends t.GenericRecord> {
	private readonly supervalidated: sf.SuperValidated<Data>;
	public readonly superform: sf.SuperForm<Data>;

	readonly values: t.State<Data>;
	readonly validationErrors: t.State<sf.ValidationErrors<Data>>;
	readonly error = $derived.by(() => this.validationErrors.current._errors);
	valid = $state(false);

	constructor(readonly config: Config<Data>) {
		const { adapter, options, initialData } = config;
		this.supervalidated = defaults(initialData, adapter);
		this.superform = superForm(this.supervalidated, {
			SPA: true,
			applyAction: false,
			scrollToError: 'smooth',
			validators: adapter,
			dataType: 'json',
			taintedMessage: null,
			...options,
			onUpdate: () => {
				this.submit();
			}
		});
		this.values = fromStore(this.superform.form);
		this.validationErrors = fromStore(this.superform.errors);

		$effect(() => {
			if (this.values.current) {
				this.superform.validateForm({ update: false }).then((result) => {
					this.valid = result.valid;
				});
			}
		});
	}

	async submit() {
		console.log('submit');
		const { onSubmit, onError } = this.config;
		try {
			if (this.valid) await onSubmit?.(this.values.current);
		} catch (e) {
			const error = new t.BaseError(e);
			setError(this.supervalidated, error.message);
			await onError?.(error);
		}
	}
}

// type SubmitFunction<Data extends t.GenericRecord> = NonNullable<sf.FormOptions<Data>['onUpdate']>;

// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type * as sf from 'sveltekit-superforms';

import { nanoid } from 'nanoid';
import { type ValidationAdapter } from 'sveltekit-superforms/adapters';
import { defaults, superForm } from 'sveltekit-superforms/client';

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
	constructor(readonly config: Config<Data>) {}

	private supervalidated?: sf.SuperValidated<Data>;
	private superform?: sf.SuperForm<Data>;

	//

	private _values = $state<Partial<Data>>({});
	get values() {
		return this._values;
	}

	private _submitError = $state<t.BaseError>();
	get submitError() {
		return this._submitError;
	}

	private _errors = $state<FormError[]>([]);
	get errors() {
		return this._errors;
	}

	private _tainted = $state(false);
	get valid() {
		return this._tainted && this.errors.length === 0;
	}

	//

	attachSuperform() {
		const { adapter, options, initialData } = this.config;
		this.supervalidated = defaults(initialData, adapter);
		this.superform = superForm(this.supervalidated, {
			id: nanoid(5),
			SPA: true,
			applyAction: false,
			scrollToError: 'smooth',
			validators: adapter,
			dataType: 'json',
			taintedMessage: null,
			...options,
			onUpdate: async ({ form }) => {
				if (form.valid) await this.submit();
			}
		});
		this.superform.form.subscribe((data) => {
			this._values = data;
		});
		this.superform.allErrors.subscribe((errors) => {
			this._errors = errors;
		});
		this.superform.tainted.subscribe((tainted) => {
			this._tainted = tainted !== undefined;
		});
	}

	async update(value: Partial<Data>) {
		this.superform?.form.update((v) => ({ ...v, ...value }), { taint: true });
		await this.superform?.validateForm({ update: true });
	}

	async submit() {
		const { onSubmit, onError } = this.config;
		this._submitError = undefined;
		try {
			if (this.valid) await onSubmit?.(this._values as Data);
		} catch (e) {
			this._submitError = new t.BaseError(e);
			await onError?.(this._submitError);
		}
	}
}

type FormError = {
	path: string;
	messages: string[];
};

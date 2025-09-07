// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type * as sf from 'sveltekit-superforms';

import { types as t } from '#';
import { nanoid } from 'nanoid';
import { type ValidationAdapter } from 'sveltekit-superforms/adapters';
import { defaults, superForm } from 'sveltekit-superforms/client';

//

type SuperformOptions<Data extends t.GenericRecord> = Omit<sf.FormOptions<Data>, 'onUpdate'>;

export type Options<Data extends t.GenericRecord> = {
	superform: SuperformOptions<Data>;
	onSubmit: (data: Data) => void | Promise<void>;
	initialData: Partial<Data>;
	onError: (error: t.BaseError) => void | Promise<void>;
};

export type Config<Data extends t.GenericRecord> = Partial<Options<Data>> & {
	adapter: ValidationAdapter<Data>;
};

export class Instance<Data extends t.GenericRecord> {
	constructor(readonly config: Config<Data>) {}

	private _supervalidated?: sf.SuperValidated<Data>;
	private _superform?: sf.SuperForm<Data>;
	get superform() {
		return this._superform;
	}

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
		const { adapter, superform, initialData } = this.config;
		this._supervalidated = defaults(initialData, adapter);
		this._superform = superForm(this._supervalidated, {
			id: nanoid(5),
			SPA: true,
			applyAction: false,
			scrollToError: 'smooth',
			validators: adapter,
			dataType: 'json',
			taintedMessage: null,
			...superform,
			onUpdate: async ({ form }) => {
				if (form.valid) await this.submit();
			}
		});
		this._superform.form.subscribe((data) => {
			this._values = data;
		});
		this._superform.allErrors.subscribe((errors) => {
			this._errors = errors;
		});
		this._superform.tainted.subscribe((tainted) => {
			this._tainted = tainted !== undefined;
		});
	}

	async update(value: Partial<Data>, options: { taint?: boolean; validate?: boolean } = {}) {
		const { taint = true, validate = true } = options;
		this._superform?.form.update((v) => ({ ...v, ...value }), { taint });
		await this._superform?.validateForm({ update: validate });
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

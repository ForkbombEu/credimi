// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getExceptionMessage } from '@/utils/errors';
import type { GenericRecord } from '@/utils/types';
import type { FormOptions as SuperformOptions } from 'sveltekit-superforms';
import { type ValidationAdapter } from 'sveltekit-superforms/adapters';
import { defaults, setError, superForm } from 'sveltekit-superforms/client';

//

export type SubmitFunction<Data extends GenericRecord = GenericRecord> = NonNullable<
	SuperformOptions<Data>['onUpdate']
>;

export type FormOptions<Data extends GenericRecord = GenericRecord> = Omit<
	SuperformOptions<Data>,
	'onUpdate'
>;

export type CreateFormProps<Data extends GenericRecord> = {
	adapter: ValidationAdapter<Data>;
	options?: FormOptions<Data>;
	onSubmit?: SubmitFunction<Data>;
	initialData?: Partial<Data>;
	onError?: (payload: { error: unknown; errorMessage: string; setFormError: (er:string) => void }) => void;
};

export function createForm<Data extends GenericRecord>(props: CreateFormProps<Data>) {
	const {
		adapter,
		initialData = {} as Partial<Data>,
		options = {},
		onSubmit = () => {},
		onError
	} = props;

	const form = defaults(initialData, adapter);
	return superForm(form, {
		SPA: true,
		applyAction: false,
		scrollToError: 'smooth',
		validators: adapter,
		dataType: 'json',
		taintedMessage: null,
		onUpdate: async (event) => {
			try {
				if (event.form.valid) await onSubmit(event);
			} catch (e) {
				const errorMessage = getExceptionMessage(e);
				const setFormError = (er:string = errorMessage) => {
					setError(event.form, er);
				};
				if (onError) onError({ error: e, errorMessage, setFormError });
				else setFormError();
			}
		},
		...options
	});
}

export const FORM_ERROR_PATH = '_errors';

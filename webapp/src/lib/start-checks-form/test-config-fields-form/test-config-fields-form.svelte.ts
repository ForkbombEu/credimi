// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { createTestConfigFormInitialData, createTestConfigFormSchema } from './utils';

import type { SuperForm, SuperValidated } from 'sveltekit-superforms';
import type { ConfigField } from '$start-checks-form/types';
import { createForm } from '@/forms';
import { zod } from 'sveltekit-superforms/adapters';
import { nanoid } from 'nanoid';
import { fromStore } from 'svelte/store';
import { Record } from 'effect';
import { watch } from 'runed';
import type { State, StringRecord } from '@/utils/types';
import type { BaseForm } from '../_utils';

//

export type TestConfigFieldsFormState = {
	isValid: boolean;
	validData: StringRecord;
	invalidData: StringRecord;
};

export type TestConfigFieldsFormProps = {
	fields: ConfigField[];
};

export class TestConfigFieldsForm implements BaseForm {
	public readonly superform: SuperForm<StringRecord>;
	public readonly values: State<StringRecord>;

	private currentValidationResult = $state<SuperValidated<StringRecord>>();
	isValid = $derived.by(() => this.currentValidationResult?.valid ?? false);

	constructor(public readonly props: TestConfigFieldsFormProps) {
		this.superform = createForm({
			adapter: zod(createTestConfigFormSchema(this.props.fields)),
			initialData: createTestConfigFormInitialData(this.props.fields),
			options: {
				id: nanoid(6)
			}
		});

		this.values = fromStore(this.superform.form);
		this.effectUpdateValidationResult();
	}

	getCompletionReport() {
		const errors = this.currentValidationResult?.errors ?? {};
		const validData = Record.filter(this.values.current, (_, id) => !(id in errors));
		const invalidData = Record.filter(this.values.current, (_, id) => id in errors);
		return {
			isValid: this.isValid,
			validData,
			invalidData,
			validFieldsCount: Record.size(validData),
			invalidFieldsCount: Record.size(invalidData)
		};
	}

	effectUpdateValidationResult() {
		watch(
			() => this.values.current,
			() => {
				this.superform.validateForm({ update: false }).then((result) => {
					this.currentValidationResult = result;
				});
			}
		);
	}

	getFormData() {
		return {
			fields: this.values.current
		};
	}
}

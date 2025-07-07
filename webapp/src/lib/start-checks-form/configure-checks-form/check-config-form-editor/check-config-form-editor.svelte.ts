// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { createCheckConfigFormInitialData, createCheckConfigFormSchema } from './utils';

import type { SuperForm, SuperValidated } from 'sveltekit-superforms';
import type { ConfigField } from '$start-checks-form/types';
import { createForm } from '@/forms';
import { zod } from 'sveltekit-superforms/adapters';
import { nanoid } from 'nanoid';
import { fromStore } from 'svelte/store';
import { Record } from 'effect';
import { watch } from 'runed';
import type { State, StringRecord } from '@/utils/types';
import type { BaseEditor } from '../../_utils';

//

export interface CheckConfigFormEditorProps {
	fields: ConfigField[];
}

export class CheckConfigFormEditor implements BaseEditor {
	public readonly superform: SuperForm<StringRecord>;
	private values: State<StringRecord>;
	private currentValidationResult = $state<SuperValidated<StringRecord>>();

	isValid = $derived.by(() => this.currentValidationResult?.valid ?? false);

	constructor(public readonly props: CheckConfigFormEditorProps) {
		this.superform = createForm({
			adapter: zod(createCheckConfigFormSchema(this.props.fields)),
			initialData: createCheckConfigFormInitialData(this.props.fields),
			options: {
				id: nanoid(6)
			}
		});
		this.values = fromStore(this.superform.form);
		this.registerEffect_UpdateValidationResult();
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

	private registerEffect_UpdateValidationResult() {
		watch(
			() => this.values.current,
			() => {
				this.superform.validateForm({ update: false }).then((result) => {
					this.currentValidationResult = result;
				});
			}
		);
	}

	getData() {
		return this.values.current;
	}
}

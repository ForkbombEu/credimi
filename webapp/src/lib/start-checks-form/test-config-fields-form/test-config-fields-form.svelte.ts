// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { createTestConfigFormInitialData, createTestConfigFormSchema } from './utils';

import type { SuperForm } from 'sveltekit-superforms';
import type { TestConfigField } from '$lib/start-checks-form/test-config-field';
import { createForm } from '@/forms';
import { zod } from 'sveltekit-superforms/adapters';
import { nanoid } from 'nanoid';
import { fromStore } from 'svelte/store';
import { Record } from 'effect';
import { watch } from 'runed';
import type { State, StringRecord } from '@/utils/types';

//

export type TestConfigFieldsFormState = {
	isValid: boolean;
	validData: StringRecord;
	invalidData: StringRecord;
};

export type TestConfigFieldsFormProps = {
	fields: TestConfigField[];
	onStateChange?: (formState: TestConfigFieldsFormState) => void;
};

export class TestConfigFieldsForm {
	public readonly superform: SuperForm<StringRecord>;
	public readonly values: State<StringRecord>;

	state = $state<TestConfigFieldsFormState>({
		isValid: false,
		validData: {},
		invalidData: {}
	});

	constructor(public readonly props: TestConfigFieldsFormProps) {
		this.superform = createForm({
			adapter: zod(createTestConfigFormSchema(this.props.fields)),
			initialData: createTestConfigFormInitialData(this.props.fields),
			options: {
				id: nanoid(6)
			}
		});

		this.values = fromStore(this.superform.form);
		this.effectDispatchUpdate();
	}

	async getFormState(): Promise<TestConfigFieldsFormState> {
		const { validateForm } = this.superform;
		const { errors, valid } = await validateForm({ update: false });

		return {
			isValid: valid,
			validData: Record.filter(this.values.current, (_, id) => !(id in errors)),
			invalidData: Record.filter(this.values.current, (_, id) => id in errors)
		};
	}

	effectDispatchUpdate() {
		watch(
			() => this.values.current,
			() => {
				this.getFormState().then((newState) => {
					this.state = newState;
					this.props.onStateChange?.(newState);
				});
			}
		);
	}
}

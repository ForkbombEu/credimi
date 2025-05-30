// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { StringRecord } from '@/utils/types';
import { watch } from 'runed';
import {
	TestConfigFieldsForm,
	type TestConfigFieldsFormProps
} from './test-config-fields-form.svelte.js';
import { Array, pipe, Record } from 'effect';
import type { Getter } from '@/utils/types';

//

export type DependentTestConfigFieldsProps = TestConfigFieldsFormProps & {
	dependentFieldsIds: string[];
	valuesDependency: Getter<StringRecord>;
};

export class DependentTestConfigFieldsForm extends TestConfigFieldsForm {
	constructor(public readonly props: DependentTestConfigFieldsProps) {
		super(props);
		this.effectUpdateDependentFields();
	}

	private overriddenFieldsIds = $state<string[]>([]);

	overriddenFields = $derived.by(() =>
		this.props.fields.filter((field) => this.overriddenFieldsIds.includes(field.CredimiID))
	);

	independentFields = $derived.by(() =>
		this.props.fields.filter(
			(field) => !this.props.dependentFieldsIds.includes(field.CredimiID)
		)
	);

	dependentFields = $derived.by(() =>
		pipe(
			this.props.fields,
			Array.difference(this.overriddenFields),
			Array.difference(this.independentFields)
		)
	);

	overrideField(fieldId: string) {
		this.overriddenFieldsIds.push(fieldId);
	}

	resetOverride(fieldId: string) {
		this.overriddenFieldsIds = this.overriddenFieldsIds.filter((id) => id !== fieldId);
		this.updateDependentFieldsValues();
	}

	private updateDependentFieldsValues() {
		const notOverriddenValues = pipe(
			this.props.valuesDependency(),
			Record.filter((_, id) => this.props.dependentFieldsIds.includes(id)),
			Record.filter((_, id) => !this.overriddenFieldsIds.includes(id))
		);

		this.superform.form.update((oldValues) => ({
			...oldValues,
			...notOverriddenValues
		}));
	}

	effectUpdateDependentFields() {
		watch(this.props.valuesDependency, () => this.updateDependentFieldsValues());
	}
}

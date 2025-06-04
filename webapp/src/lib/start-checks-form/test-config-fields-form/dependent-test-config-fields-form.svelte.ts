// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { watch } from 'runed';
import {
	TestConfigFieldsForm,
	type TestConfigFieldsFormProps
} from './config-form-fields.svelte.js';
import { Array, pipe, Record } from 'effect';

//

export type DependentTestConfigFieldsProps = TestConfigFieldsFormProps & {
	formDependency: TestConfigFieldsForm;
};

export class DependentTestConfigFieldsForm extends TestConfigFieldsForm {
	private dependentFieldsIds: string[];

	constructor(public readonly props: DependentTestConfigFieldsProps) {
		super(props);
		this.dependentFieldsIds = this.props.formDependency.props.fields.map((f) => f.CredimiID);
		this.effectUpdateDependentFields();
	}

	private overriddenFieldsIds = $state<string[]>([]);

	overriddenFields = $derived.by(() =>
		this.props.fields.filter((field) => this.overriddenFieldsIds.includes(field.CredimiID))
	);

	independentFields = $derived.by(() =>
		this.props.fields.filter((field) => !this.dependentFieldsIds.includes(field.CredimiID))
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
			this.props.formDependency.values.current,
			Record.filter((_, id) => this.dependentFieldsIds.includes(id)),
			Record.filter((_, id) => !this.overriddenFieldsIds.includes(id))
		);

		this.superform.form.update((oldValues) => ({
			...oldValues,
			...notOverriddenValues
		}));
	}

	effectUpdateDependentFields() {
		watch(
			() => this.props.formDependency.values.current,
			() => this.updateDependentFieldsValues()
		);
	}
}

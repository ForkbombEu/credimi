// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { NamedConfigField } from '$lib/start-checks-form/types.js';
import {
	CheckConfigFormEditor,
	type CheckConfigFormEditorProps
} from './check-config-form-editor.svelte.js';
import { Array, pipe, Record } from 'effect';

//

export interface DependentCheckConfigFormEditorProps extends CheckConfigFormEditorProps {
	fields: NamedConfigField[];
	formDependency: CheckConfigFormEditor;
}

export class DependentCheckConfigFormEditor extends CheckConfigFormEditor {
	constructor(public readonly props: DependentCheckConfigFormEditorProps) {
		super(props);
		this.dependentFieldsIds = this.props.formDependency.props.fields.map((f) => f.credimi_id);
		this.registerEffect_UpdateDependentFields();
	}

	private dependentFieldsIds: string[];
	private overriddenFieldsIds = $state<string[]>([]);

	overriddenFields = $derived.by(() =>
		this.props.fields.filter((field) => this.overriddenFieldsIds.includes(field.credimi_id))
	);

	independentFields = $derived.by(() =>
		this.props.fields.filter((field) => !this.dependentFieldsIds.includes(field.credimi_id))
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
		this.updateDependentFields();
	}

	private updateDependentFields() {
		const notOverriddenValues = pipe(
			this.props.formDependency.getData(),
			Record.filter((_, id) => this.dependentFieldsIds.includes(id)),
			Record.filter((_, id) => !this.overriddenFieldsIds.includes(id))
		);

		this.superform.form.update((oldValues) => ({
			...oldValues,
			...notOverriddenValues
		}));
	}

	private registerEffect_UpdateDependentFields() {
		$effect(() => {
			this.updateDependentFields();
		});
	}
}

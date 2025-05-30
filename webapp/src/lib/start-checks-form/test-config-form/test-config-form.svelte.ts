// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { TestConfigJsonForm } from '$lib/start-checks-form/test-config-json-form';
import {
	DependentTestConfigFieldsForm,
	type DependentTestConfigFieldsProps
} from '$lib/start-checks-form/test-config-fields-form';

//

type TestConfigFormProps = DependentTestConfigFieldsProps & {
	json: string;
};

export type TestConfigFormMode = 'json' | 'fields';

export class TestConfigForm {
	public readonly jsonForm: TestConfigJsonForm;
	public readonly fieldsForm: DependentTestConfigFieldsForm;

	mode: TestConfigFormMode = $derived.by(() => (this.jsonForm.isTainted ? 'json' : 'fields'));

	isValid = $derived.by(() =>
		this.mode === 'json' ? this.jsonForm.isValid : this.fieldsForm.state.isValid
	);

	constructor(public readonly props: TestConfigFormProps) {
		this.fieldsForm = new DependentTestConfigFieldsForm({
			fields: this.props.fields,
			formDependency: this.props.formDependency
		});

		this.jsonForm = new TestConfigJsonForm({
			json: this.props.json,
			formDependency: this.fieldsForm
		});
	}
}

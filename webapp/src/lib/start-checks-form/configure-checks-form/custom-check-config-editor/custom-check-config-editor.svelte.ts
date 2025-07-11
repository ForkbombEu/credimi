// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { createJsonSchemaForm, type JsonSchemaForm } from '@/components/json-schema-form';
import type { CustomChecksResponse } from '@/pocketbase/types';
import type { BaseEditor } from '$start-checks-form/_utils';
import type { SuperForm, SuperValidated } from 'sveltekit-superforms';
import { z } from 'zod';
import { zod } from 'sveltekit-superforms/adapters';
import { createForm } from '@/forms';
import { yamlStringSchema } from '$lib/utils';
import type { State } from '@/utils/types';
import { fromStore } from 'svelte/store';
import { watch } from 'runed';

//

export type CustomCheckConfigEditorProps = {
	customCheck: CustomChecksResponse;
};

export class CustomCheckConfigEditor implements BaseEditor {
	public readonly jsonSchemaForm: JsonSchemaForm;
	public readonly yamlForm: SuperForm<YamlFormData>;
	private yamlFormState: State<YamlFormData>;
	private yamlFormValidationResult = $state<SuperValidated<YamlFormData>>();

	isValid = $derived.by(() => {
		const jsonSchemaFormIsValid = this.jsonSchemaForm.validate().size === 0;
		const yamlFormIsValid = this.yamlFormValidationResult?.valid ?? false;
		return jsonSchemaFormIsValid && yamlFormIsValid;
	});

	constructor(public readonly props: CustomCheckConfigEditorProps) {
		this.jsonSchemaForm = createJsonSchemaForm(props.customCheck.input_json_schema as object, {
			hideTitle: true
		});

		this.yamlForm = createForm({
			adapter: zod(z.object({ yaml: yamlStringSchema })),
			initialData: { yaml: props.customCheck.yaml }
		});
		this.yamlFormState = fromStore(this.yamlForm.form);

		this.registerEffect_UpdateYamlFormValidationResult();
	}

	getData() {
		return {
			form: this.jsonSchemaForm.value,
			yaml: this.yamlFormState.current.yaml
		};
	}

	private registerEffect_UpdateYamlFormValidationResult() {
		watch(
			() => this.yamlFormState.current,
			() => {
				this.yamlForm.validateForm({ update: false }).then((result) => {
					this.yamlFormValidationResult = result;
				});
			}
		);
	}
}

type YamlFormData = {
	yaml: string;
};

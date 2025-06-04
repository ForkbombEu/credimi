// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { createJsonSchemaForm, type JsonSchemaForm } from '@/components/json-schema-form';
import type { CustomChecksResponse } from '@/pocketbase/types';
import type { BaseEditor } from '$start-checks-form/_utils';

export type CustomCheckConfigEditorProps = {
	customCheck: CustomChecksResponse;
};

//

export class CustomCheckConfigEditor implements BaseEditor {
	public readonly jsonSchemaForm: JsonSchemaForm;

	isValid = $derived.by(() => this.jsonSchemaForm.validate().size === 0);

	constructor(public readonly props: CustomCheckConfigEditorProps) {
		this.jsonSchemaForm = createJsonSchemaForm(props.customCheck.input_json_schema as object, {
			hideTitle: true
		});
	}

	getData() {
		return {
			form: this.jsonSchemaForm.value,
			yaml: this.props.customCheck.yaml
		};
	}
}

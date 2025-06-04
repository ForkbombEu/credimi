// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { createJsonSchemaForm, type JsonSchemaForm } from '@/components/json-schema-form';
import type { CustomChecksResponse } from '@/pocketbase/types';
import type { BaseForm } from '../_utils';

export type CustomCheckFormProps = {
	customCheck: CustomChecksResponse;
};

//

export class CustomCheckForm implements BaseForm {
	public readonly jsonSchemaForm: JsonSchemaForm;

	isValid = $derived.by(() => this.jsonSchemaForm.validate().size === 0);

	constructor(public readonly props: CustomCheckFormProps) {
		this.jsonSchemaForm = createJsonSchemaForm(props.customCheck.input_json_schema as object, {
			hideTitle: true
		});
	}

	getFormData() {
		return {
			form: this.jsonSchemaForm.value,
			yaml: this.props.customCheck.yaml
		};
	}
}

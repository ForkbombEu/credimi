// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { createJsonSchemaForm, type JsonSchemaForm } from '@/components/json-schema-form';
import type { CustomChecksResponse } from '@/pocketbase/types';

export type CustomCheckFormProps = {
	customCheck: CustomChecksResponse;
};

//

export class CustomCheckForm {
	public readonly jsonSchemaForm: JsonSchemaForm;

	isValid = $derived.by(() => this.jsonSchemaForm.validate().size === 0);

	constructor(public readonly props: CustomCheckFormProps) {
		this.jsonSchemaForm = createJsonSchemaForm(props.customCheck.input_json_schema as object, {
			hideTitle: true
		});
	}
}

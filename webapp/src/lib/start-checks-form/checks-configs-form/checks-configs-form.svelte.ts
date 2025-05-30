// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { z } from 'zod';
import { pb } from '@/pocketbase';
import {
	baseTestConfigFieldSchema,
	namedTestConfigFieldSchema,
	testConfigFieldComparator
} from '$lib/start-checks-form/test-config-field';
import { stringifiedObjectSchema } from '$lib/start-checks-form/_utils';
import { TestConfigFieldsForm } from '$lib/start-checks-form/test-config-fields-form';
import { TestConfigForm } from '$lib/start-checks-form/test-config-form';
import { Record } from 'effect';

//

export const checksConfigFormPropsSchema = z.object({
	normalized_fields: z.array(baseTestConfigFieldSchema),
	specific_fields: z.record(
		z.string(),
		z.object({
			content: stringifiedObjectSchema,
			fields: z.array(namedTestConfigFieldSchema)
		})
	)
});

export type ChecksConfigFormProps = z.infer<typeof checksConfigFormPropsSchema>;

//

export class ChecksConfigForm {
	public readonly sharedFieldsForm: TestConfigFieldsForm;
	public readonly checksForms: Record<string, TestConfigForm>;

	hasSharedFields = $derived.by(() => this.props.normalized_fields.length > 0);

	constructor(public readonly props: ChecksConfigFormProps) {
		this.sharedFieldsForm = new TestConfigFieldsForm({
			fields: this.props.normalized_fields.sort(testConfigFieldComparator)
		});

		this.checksForms = Record.map(
			this.props.specific_fields,
			(data, id) =>
				new TestConfigForm({
					id,
					json: data.content,
					fields: data.fields.sort(testConfigFieldComparator),
					formDependency: this.sharedFieldsForm
				})
		);
	}

	getCompletionStatus() {
		const forms = Record.map(this.checksForms, (form) => form.isValid);
		const missingSharedFieldsCount = this.sharedFieldsForm.state.invalidData.length;

		const validFormsCount = Object.values(forms).filter(Boolean).length;
		const invalidFormsCount = Object.values(forms).filter((v) => !v).length;
		const totalForms = Object.keys(forms).length;
		const invalidFormIds = Object.entries(forms)
			.filter(([, isValid]) => !isValid)
			.map(([id]) => id);

		return {
			sharedFields: this.sharedFieldsForm.state.isValid,
			validFormsCount,
			invalidFormsCount,
			totalForms,
			invalidFormIds,
			missingSharedFieldsCount
		};
	}
}

//

export async function getChecksConfigFormProps(suiteAndVersionPath: string, filenames: string[]) {
	const data = await pb.send('/api/template/placeholders', {
		method: 'POST',
		body: {
			test_id: suiteAndVersionPath,
			filenames
		}
	});
	return checksConfigFormPropsSchema.parse(data);
}

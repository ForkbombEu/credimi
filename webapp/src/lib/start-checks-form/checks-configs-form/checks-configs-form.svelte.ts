// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { z } from 'zod';
import { pb } from '@/pocketbase';
import {
	baseTestConfigFieldSchema,
	namedTestConfigFieldSchema
} from '$lib/start-checks-form/test-config-field';
import { stringifiedObjectSchema } from '$lib/start-checks-form/utils';
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

	constructor(public readonly props: ChecksConfigFormProps) {
		this.sharedFieldsForm = new TestConfigFieldsForm({
			fields: this.props.normalized_fields
		});

		this.checksForms = Record.map(
			this.props.specific_fields,
			(data) =>
				new TestConfigForm({
					json: data.content,
					fields: data.fields,
					formDependency: this.sharedFieldsForm
				})
		);
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

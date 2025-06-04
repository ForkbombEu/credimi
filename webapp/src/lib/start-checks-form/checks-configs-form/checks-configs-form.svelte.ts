// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { z } from 'zod';
import { pb } from '@/pocketbase';
import { baseConfigFieldSchema, namedConfigFieldSchema } from '$start-checks-form/types';
import { configFieldComparator, stringifiedObjectSchema } from '$start-checks-form/_utils';
import { TestConfigFieldsForm } from '$start-checks-form/test-config-fields-form';
import { TestConfigForm } from '$start-checks-form/test-config-form';
import { pipe, Record } from 'effect';
import type { CustomChecksResponse } from '@/pocketbase/types';
import { CustomCheckForm } from '$start-checks-form/custom-check-form';
import { goto } from '@/i18n';

//

export const checksConfigFormPropsSchema = z.object({
	normalized_fields: z.array(baseConfigFieldSchema),
	specific_fields: z.record(
		z.string(),
		z.object({
			content: stringifiedObjectSchema,
			fields: z.array(namedConfigFieldSchema)
		})
	)
});

export type ChecksConfigFormProps = {
	standardAndVersionPath: string;
	standardChecks: z.infer<typeof checksConfigFormPropsSchema>;
	customChecks: CustomChecksResponse[];
};

//

export class ChecksConfigForm {
	public readonly sharedFieldsForm: TestConfigFieldsForm;
	public readonly checksForms: Record<string, TestConfigForm>;
	public readonly customChecksForms: CustomCheckForm[];

	hasSharedFields = $derived.by(() => this.props.standardChecks.normalized_fields.length > 0);
	isValid = $derived.by(() => this.getCompletionStatus().isValid);

	isLoading = $state(false);
	loadingError = $state<Error>();

	constructor(public readonly props: ChecksConfigFormProps) {
		this.sharedFieldsForm = new TestConfigFieldsForm({
			fields: this.props.standardChecks.normalized_fields.sort(configFieldComparator)
		});

		this.checksForms = Record.map(
			this.props.standardChecks.specific_fields,
			(data, id) =>
				new TestConfigForm({
					id,
					json: data.content,
					fields: data.fields.sort(configFieldComparator),
					formDependency: this.sharedFieldsForm
				})
		);

		this.customChecksForms = this.props.customChecks.map(
			(c) => new CustomCheckForm({ customCheck: c })
		);
	}

	async submit() {
		this.isLoading = true;
		try {
			console.log(this.getFormData());
			await pb.send(
				`/api/compliance/${this.props.standardAndVersionPath}/save-variables-and-start`,
				{
					method: 'POST',
					body: this.getFormData()
				}
			);
			await goto(`/my/tests/runs`);
		} catch (error) {
			this.loadingError = error as Error;
		} finally {
			this.isLoading = false;
		}
	}

	getFormData() {
		const configs_with_fields = pipe(
			this.checksForms,
			Record.map((form) => form.getFormData()),
			Record.filter((v) => v.mode == 'fields'),
			Record.map((v) => v.value.fields)
		);
		const configs_with_json = pipe(
			this.checksForms,
			Record.map((form) => form.getFormData()),
			Record.filter((v) => v.mode == 'json'),
			Record.map((v) => v.value.json)
		);
		const custom_checks = Record.fromIterableWith(this.customChecksForms, (form) => [
			form.props.customCheck.id,
			form.getFormData()
		]);

		return $state.snapshot({
			configs_with_fields,
			configs_with_json,
			custom_checks
		});
	}

	getCompletionStatus() {
		const missingSharedFieldsCount =
			this.sharedFieldsForm.getCompletionReport().invalidFieldsCount;

		//

		const baseForms = Record.map(this.checksForms, (form) => form.isValid);
		const validBaseFormsCount = Object.values(baseForms).filter(Boolean).length;
		const invalidBaseFormsCount = Object.keys(baseForms).length - validBaseFormsCount;

		const validCustomChecksFormsCount = this.customChecksForms.filter((c) => c.isValid).length;
		const invalidCustomChecksFormsCount =
			this.customChecksForms.length - validCustomChecksFormsCount;

		//

		const invalidBaseFormsEntries: InvalidFormEntry[] = Object.entries(baseForms)
			.filter(([, isValid]) => !isValid)
			.map(([id]) => ({ text: id, id }));

		const invalidCustomChecksFormsEntries: InvalidFormEntry[] = this.customChecksForms
			.filter((c) => !c.isValid)
			.map((c) => ({ text: c.props.customCheck.name, id: c.props.customCheck.id }));

		return {
			isValid:
				missingSharedFieldsCount === 0 &&
				invalidBaseFormsCount === 0 &&
				invalidCustomChecksFormsCount === 0,
			sharedFields: this.sharedFieldsForm.getCompletionReport().isValid,
			validFormsCount: validBaseFormsCount + validCustomChecksFormsCount,
			invalidFormsCount: invalidBaseFormsCount + invalidCustomChecksFormsCount,
			invalidFormsEntries: [...invalidBaseFormsEntries, ...invalidCustomChecksFormsEntries],
			missingSharedFieldsCount
		};
	}
}

type InvalidFormEntry = {
	text: string;
	id: string;
};

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

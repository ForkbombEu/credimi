// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { configFieldComparator, LatestCheckRunsStorage } from '$start-checks-form/_utils';
import { pipe, Record } from 'effect';

import { goto } from '@/i18n';
import { pb } from '@/pocketbase';

import type { StartChecksResponse } from '$start-checks-form/types';
import type { SelectChecksSubmitData } from '../select-checks-form';

import { CheckConfigEditor } from './check-config-editor';
import { CheckConfigFormEditor } from './check-config-form-editor';
import { CustomCheckConfigEditor } from './custom-check-config-editor';

//

export type ConfigureChecksFormProps = SelectChecksSubmitData;

export class ConfigureChecksForm {
	public readonly sharedFieldsEditor: CheckConfigFormEditor;
	public readonly checkConfigEditors: Record<string, CheckConfigEditor>;
	public readonly customCheckConfigEditors: CustomCheckConfigEditor[];

	constructor(public readonly props: ConfigureChecksFormProps) {
		this.sharedFieldsEditor = new CheckConfigFormEditor({
			fields: this.fields.normalized_fields.sort(configFieldComparator)
		});

		this.checkConfigEditors = Record.map(
			this.fields.specific_fields,
			(data, id) =>
				new CheckConfigEditor({
					id,
					code: data.content,
					fields: data.fields.sort(configFieldComparator),
					formDependency: this.sharedFieldsEditor
				})
		);

		this.customCheckConfigEditors = this.props.customChecks.map(
			(c) => new CustomCheckConfigEditor({ customCheck: c })
		);
	}

	// Utility

	private get fields() {
		return (
			this.props.checksConfigsFields ?? {
				normalized_fields: [],
				specific_fields: {}
			}
		);
	}

	hasSharedFields = $derived.by(() => this.fields.normalized_fields.length > 0);

	// Form submission

	isLoading = $state(false);
	loadingError = $state<Error>();

	async submit() {
		this.isLoading = true;
		try {
			const response: StartChecksResponse = await pb.send(
				`/api/compliance/${this.props.standardAndVersionPath}/save-variables-and-start`,
				{
					method: 'POST',
					body: this.getFormData()
				}
			);
			LatestCheckRunsStorage.set(
				response.results.map((res) => ({
					...res,
					standardAndVersion: response['protocol/version']
				}))
			);
			await goto(`/my/tests/runs`);
		} catch (error) {
			this.loadingError = error as Error;
		} finally {
			this.isLoading = false;
		}
	}

	getFormData() {
		type Entries = {
			credimi_id: string;
			value: unknown;
			field_name: string;
		};
		const configs_with_fields = pipe(
			this.checkConfigEditors,
			Record.map((form) => {
				const { mode, value } = form.getData();
				if (mode != 'form') return undefined;

				const entries: Entries[] = [];
				// TODO: dentro value ci sono campi che non dovrebbero esserci, da investigare
				for (const [credimiId, datum] of Record.toEntries(value)) {
					entries.push({
						credimi_id: credimiId,
						value: datum,
						field_name:
							form.props.fields.find((f) => f.credimi_id == credimiId)?.field_id ?? ''
					});
				}
				return entries.filter((e) => e.field_name != '');
			}),
			Record.filter((v) => v != undefined)
		);

		const configs_with_json = pipe(
			this.checkConfigEditors,
			Record.map((form) => form.getData()),
			Record.filter((v) => v.mode == 'code'),
			Record.map((v) => v.value)
		);

		const custom_checks = Record.fromIterableWith(this.customCheckConfigEditors, (form) => [
			form.props.customCheck.id,
			form.getData()
		]);

		return $state.snapshot({
			configs_with_fields,
			configs_with_json,
			custom_checks
		});
	}

	// Form completion status

	isValid = $derived.by(() => this.getCompletionStatus().isValid);

	getCompletionStatus() {
		const missingSharedFieldsCount =
			this.sharedFieldsEditor.getCompletionReport().invalidFieldsCount;

		//

		const baseForms = Record.map(this.checkConfigEditors, (form) => form.isValid);
		const validBaseFormsCount = Object.values(baseForms).filter(Boolean).length;
		const invalidBaseFormsCount = Object.keys(baseForms).length - validBaseFormsCount;

		const validCustomChecksFormsCount = this.customCheckConfigEditors.filter(
			(c) => c.isValid
		).length;
		const invalidCustomChecksFormsCount =
			this.customCheckConfigEditors.length - validCustomChecksFormsCount;

		//

		const invalidBaseFormsEntries: InvalidFormEntry[] = Object.entries(baseForms)
			.filter(([, isValid]) => !isValid)
			.map(([id]) => ({ text: id, id }));

		const invalidCustomChecksFormsEntries: InvalidFormEntry[] = this.customCheckConfigEditors
			.filter((c) => !c.isValid)
			.map((c) => ({ text: c.props.customCheck.name, id: c.props.customCheck.id }));

		return {
			isValid:
				missingSharedFieldsCount === 0 &&
				invalidBaseFormsCount === 0 &&
				invalidCustomChecksFormsCount === 0,
			sharedFields: this.sharedFieldsEditor.getCompletionReport().isValid,
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

// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { SuperForm, SuperValidated } from 'sveltekit-superforms';

import { getValueSnapshot, validate } from '@sjsf/form';
import { resolve } from '$app/paths';
import { runWithLoading, yamlStringSchema } from '$lib/utils';
import { watch } from 'runed';
import { toast } from 'svelte-sonner';
import { fromStore } from 'svelte/store';
import { zod } from 'sveltekit-superforms/adapters';
import { z } from 'zod/v3';

import type { CustomChecksResponse } from '@/pocketbase/types';
import type { State } from '@/utils/types';

import { createJsonSchemaForm, type JsonSchemaForm } from '@/components/json-schema-form';
import { createForm } from '@/forms';
import { goto, m } from '@/i18n';
import { pb } from '@/pocketbase';

//

export type CustomCheckConfigEditorProps = {
	customCheck: CustomChecksResponse;
};

export class CustomCheckConfigEditor {
	public readonly jsonSchemaForm?: JsonSchemaForm;
	public readonly yamlForm: SuperForm<YamlFormData>;
	private yamlFormState: State<YamlFormData>;
	private yamlFormValidationResult = $state<SuperValidated<YamlFormData>>();

	isValid = $derived.by(() => {
		let jsonSchemaFormIsValid = true;
		if (this.jsonSchemaForm) {
			const errors = validate(this.jsonSchemaForm).errors ?? [];
			jsonSchemaFormIsValid = errors.length === 0;
		}

		const yamlFormIsValid = this.yamlFormValidationResult?.valid ?? false;
		return jsonSchemaFormIsValid && yamlFormIsValid;
	});

	constructor(public readonly props: CustomCheckConfigEditorProps) {
		const jsonSchema = props.customCheck.input_json_schema;
		if (jsonSchema) {
			this.jsonSchemaForm = createJsonSchemaForm(jsonSchema as object, {
				hideTitle: true,
				initialValue: props.customCheck.input_json_sample
			});
		}

		this.yamlForm = createForm({
			adapter: zod(z.object({ yaml: yamlStringSchema })),
			initialData: { yaml: props.customCheck.yaml }
		});
		this.yamlFormState = fromStore(this.yamlForm.form);

		this.registerEffect_UpdateYamlFormValidationResult();
	}

	getData() {
		return {
			data: this.jsonSchemaForm ? getValueSnapshot(this.jsonSchemaForm) : undefined,
			yaml: this.yamlFormState.current.yaml
		};
	}

	async submit() {
		const formData = this.getData();
		await runWithLoading({
			fn: async () => {
				try {
					const result = await pb.send<RunResult>('/api/custom-integrations/run', {
						method: 'POST',
						body: formData
					});
					toast.success(m.Custom_integration_started_successfully(), {
						action: {
							label: m.View(),
							onClick: () => {
								goto(
									resolve('/my/tests/runs/[workflow_id]/[run_id]', {
										workflow_id: result.workflowId,
										run_id: result.workflowRunId
									})
								);
							}
						}
					});
					await goto(resolve('/my/custom-integrations'));
				} catch (error) {
					toast.error(m.Failed_to_run_custom_integration());
					throw error;
				}
			},
			showSuccessToast: false
		});
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

type RunResult = {
	workflowId: string;
	workflowRunId: string;
};

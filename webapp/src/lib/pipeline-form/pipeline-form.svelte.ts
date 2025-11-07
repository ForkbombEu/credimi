// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { slug } from '$lib/utils/index.js';
import { goto } from '@/i18n';
import { pb } from '@/pocketbase/index.js';
import type { PipelinesFormData } from '@/pocketbase/types/extra.generated.js';
import { stringify } from 'yaml';
import { ActivityOptionsForm } from './activity-options-form/activity-options-form.svelte.js';
import { MetadataForm } from './metadata-form/metadata-form.svelte.js';
import Component from './pipeline-form.svelte';
import { convertBuilderSteps, formatYaml } from './pipeline.functions.js';
import type { HttpsGithubComForkbombeuCredimiPkgWorkflowenginePipelineWorkflowDefinition as Pipeline } from './pipeline.types.generated';
import { StepsBuilder } from './steps-builder/steps-builder.svelte.js';

//

export class PipelineForm {
	readonly Component = Component;

	readonly stepsBuilder = new StepsBuilder({
		steps: [],
		yamlPreview: () => this.yamlString
	});
	readonly activityOptionsForm = new ActivityOptionsForm();

	readonly metadataForm = new MetadataForm();

	readonly yaml: Pipeline = $derived({
		name: this.metadataForm.value?.name ?? '',
		runtime: {
			temporal: {
				activity_options: this.activityOptionsForm.value
			}
		},
		steps: convertBuilderSteps(this.stepsBuilder.steps)
	});
	readonly yamlString: string = $derived(formatYaml(stringify(this.yaml)));

	//

	async save() {
		if (!this.metadataForm.value) {
			this.metadataForm.isOpen = true;
		} else {
			const data: Omit<PipelinesFormData, 'owner'> = {
				...this.metadataForm.getValueOrThrow(),
				canonified_name: slug(this.metadataForm.value.name),
				steps: JSON.stringify(this.stepsBuilder.steps),
				yaml: this.yamlString
			};
			await pb.collection('pipelines').create(data);
			goto('/my/pipelines');
		}
	}
}

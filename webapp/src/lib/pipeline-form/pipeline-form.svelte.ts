// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

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
		name: this.metadataForm.value.name,
		runtime: {
			temporal: {
				activity_options: this.activityOptionsForm.value
			}
		},
		steps: convertBuilderSteps(this.stepsBuilder.steps)
	});
	readonly yamlString: string = $derived(formatYaml(stringify(this.yaml)));
}

// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { beforeNavigate } from '$app/navigation';
import type { Renderable } from '$lib/renderable';
import { runWithLoading } from '$lib/utils/index.js';
import { goto, m } from '@/i18n';
import { pb } from '@/pocketbase/index.js';
import type { PipelinesFormData } from '@/pocketbase/types/extra.generated.js';
import { ActivityOptionsForm } from './activity-options-form/activity-options-form.svelte.js';
import { createPipelineYaml } from './functions.js';
import { MetadataForm } from './metadata-form/metadata-form.svelte.js';
import Component from './pipeline-form.svelte';
import { StepsBuilder } from './steps-builder/steps-builder.svelte.js';
import type { EnrichedPipeline } from './types';

//

type Props = {
	mode: 'create' | 'edit' | 'view';
	pipeline?: EnrichedPipeline;
};

export class PipelineForm implements Renderable<PipelineForm> {
	readonly Component = Component;

	readonly stepsBuilder: StepsBuilder;
	readonly activityOptionsForm: ActivityOptionsForm;
	readonly metadataForm: MetadataForm;

	constructor(private props: Props) {
		this.stepsBuilder = new StepsBuilder({
			steps: props.pipeline?.steps ?? [],
			yamlPreview: () => this.yamlString
		});

		this.activityOptionsForm = new ActivityOptionsForm({
			initialData: props.pipeline?.activity_options
		});

		this.metadataForm = new MetadataForm({
			initialData: props.mode === 'view' ? undefined : props.pipeline?.metadata,
			onSubmit: async () => {
				if (!this.saveAfterMetadataFormSubmit) return;
				await this.save();
				this.saveAfterMetadataFormSubmit = false;
			}
		});

		beforeNavigate(({ cancel }) => {
			if (this.isSaving) return;
			if (!this.validateExit()) cancel();
		});
	}

	get mode() {
		return this.props.mode;
	}

	readonly yamlString: string = $derived.by(() =>
		createPipelineYaml(
			$state.snapshot(this.metadataForm.value?.name ?? ''),
			$state.snapshot(this.stepsBuilder.steps.map(([step]) => step)),
			$state.snapshot(this.activityOptionsForm.value)
		)
	);

	//

	private saveAfterMetadataFormSubmit = $state(false);

	private isSaving = false;

	async save() {
		if (!this.metadataForm.value) {
			this.metadataForm.isOpen = true;
			if (this.props.mode === 'create' || this.props.mode === 'view') {
				this.saveAfterMetadataFormSubmit = true;
			}
		} else {
			const data: Omit<PipelinesFormData, 'owner' | 'canonified_name'> = {
				...this.metadataForm.value,
				yaml: this.yamlString
			};
			runWithLoading({
				fn: async () => {
					this.isSaving = true;
					if (this.props.mode === 'edit' && this.props.pipeline) {
						await pb
							.collection('pipelines')
							.update(this.props.pipeline.metadata.id, data);
					} else {
						await pb.collection('pipelines').create(data);
					}
					goto('/my/pipelines');
				}
			});
		}
	}

	//

	hasChanges = $derived.by(() => {
		if (this.props.mode === 'create') {
			return this.stepsBuilder.steps.length > 0;
		} else if (this.props.mode === 'edit') {
			return this.props.pipeline?.metadata.yaml !== this.yamlString;
		} else {
			return false;
		}
	});

	validateExit() {
		if (this.hasChanges) {
			return confirm(
				m.You_have_unsaved_changes() + '\n' + m.Are_you_sure_you_want_to_exit_the_form()
			);
		} else {
			return true;
		}
	}
}

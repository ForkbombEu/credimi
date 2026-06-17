// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Renderable } from '$lib/renderable';

import { beforeNavigate } from '$app/navigation';
import { Pipeline } from '$lib';
import { runWithLoading } from '$lib/utils/index.js';
import _ from 'lodash';

import type { PipelinesFormData } from '@/pocketbase/types/extra.generated.js';

import { goto, m } from '@/i18n';
import { pb } from '@/pocketbase/index.js';

import { showPipelineFormError } from './errors.js';
import { ExecutionTarget } from './execution-target/index.js';
import { createPipelineYaml, type EnrichedPipeline } from './functions.js';
import { MetadataForm } from './metadata-form/metadata-form.svelte.js';
import Component from './pipeline-form.svelte';
import { RuntimeOptionsForm } from './runtime-options-form/runtime-options-form.svelte.js';
import { StepsBuilder } from './steps-builder/steps-builder.svelte.js';

//

type Props = {
	mode: 'create' | 'edit';
	pipeline?: EnrichedPipeline;
};

export class PipelineForm implements Renderable<PipelineForm> {
	readonly Component = Component;

	readonly stepsBuilder: StepsBuilder;
	readonly runtimeOptionsForm: RuntimeOptionsForm;
	readonly metadataForm: MetadataForm;

	constructor(private props: Props) {
		if (props.pipeline) {
			ExecutionTarget.loadFromPipeline(props.pipeline);
		} else {
			ExecutionTarget.clear();
		}

		this.stepsBuilder = new StepsBuilder({
			steps: props.pipeline?.steps ?? [],
			yamlPreview: () => this.yamlString
		});

		this.runtimeOptionsForm = new RuntimeOptionsForm({
			initialData: props.pipeline?.runtime
		});

		this.metadataForm = new MetadataForm({
			initialData: props.pipeline?.record,
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

	readonly manualEditHref = $derived.by(() => {
		const record = this.props.pipeline?.record;
		if (!record) return '/my/pipelines/new/manual';
		return Pipeline.getManualEditHref(record);
	});

	readonly yamlString: string = $derived.by(() =>
		createPipelineYaml(
			this.metadataForm.value?.name ?? '',
			this.stepsBuilder.steps.map(([step]) => step),
			this.runtimeOptionsForm.value
		)
	);

	//

	private saveAfterMetadataFormSubmit = $state(false);

	private isSaving = false;

	async save() {
		if (!this.metadataForm.value) {
			this.metadataForm.isOpen = true;
			if (this.props.mode === 'create') {
				this.saveAfterMetadataFormSubmit = true;
			}
		} else {
			let yaml: string;
			let manual = false;

			if (this.stepsBuilder.isManualMode && this.stepsBuilder.mode.id === 'manual') {
				const result = await this.stepsBuilder.mode.editor.validateNow();
				if (!result.ok) {
					showPipelineFormError(result.message);
					return;
				}
				yaml = result.value;
				manual = true;
			} else {
				try {
					yaml = this.yamlString;
				} catch (e) {
					showPipelineFormError(e);
					return;
				}
			}

			const data: Omit<PipelinesFormData, 'owner' | 'canonified_name'> = {
				...this.metadataForm.value,
				yaml,
				...(manual ? { manual: true } : {})
			};
			await runWithLoading({
				fn: async () => {
					try {
						this.isSaving = true;
						if (this.props.mode === 'edit' && this.props.pipeline) {
							await pb
								.collection('pipelines')
								.update(this.props.pipeline.record.id, data);
						} else {
							await pb.collection('pipelines').create(data);
						}
						await goto('/my/pipelines');
					} catch (e) {
						showPipelineFormError(e);
						throw e;
					} finally {
						this.isSaving = false;
					}
				}
			});
		}
	}

	//

	hasChanges = $derived.by(() => {
		const { pipeline } = this.props;

		const runtimeOptionsChanged = !_.isEqual(this.runtimeOptionsForm.value, pipeline?.runtime);
		const nameChanged = this.metadataForm.value?.name !== pipeline?.record.name;
		const descChanged = this.metadataForm.value?.description !== pipeline?.record.description;
		const metadataChanged = nameChanged || descChanged;

		if (this.stepsBuilder.isManualMode && this.stepsBuilder.mode.id === 'manual') {
			return (
				this.stepsBuilder.mode.editor.isDirty || runtimeOptionsChanged || metadataChanged
			);
		}

		const stepsChanged = !_.isEqual(
			this.stepsBuilder.steps.map(([step]) => step),
			pipeline?.steps.map(([step]) => step)
		);

		return stepsChanged || runtimeOptionsChanged || metadataChanged;
	});

	canSave = $derived.by(() => {
		if (this.stepsBuilder.isManualMode && this.stepsBuilder.mode.id === 'manual') {
			return this.hasChanges && this.stepsBuilder.mode.editor.isValid;
		}
		return this.hasChanges && this.stepsBuilder.steps.length > 0;
	});

	validateExit() {
		const manualEditorDirty =
			this.stepsBuilder.isManualMode &&
			this.stepsBuilder.mode.id === 'manual' &&
			this.stepsBuilder.mode.editor.isDirty;

		if (this.hasChanges || manualEditorDirty) {
			return confirm(
				m.You_have_unsaved_changes() + '\n' + m.Are_you_sure_you_want_to_exit_the_form()
			);
		} else {
			return true;
		}
	}
}

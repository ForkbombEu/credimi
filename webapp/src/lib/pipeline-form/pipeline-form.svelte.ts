// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { BeforeNavigate } from '@sveltejs/kit';
import type { Renderable } from '$lib/renderable';

import { beforeNavigate } from '$app/navigation';
import { confirm } from '$lib/layout/global-confirm.svelte';
import { runWithLoading } from '$lib/utils/index.js';
import _ from 'lodash';
import { toast } from 'svelte-sonner';

import type { PipelinesFormData } from '@/pocketbase/types/extra.generated.js';

import { goto, m } from '@/i18n';
import { pb } from '@/pocketbase/index.js';
import { getExceptionMessage } from '@/utils/errors.js';

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
	startLockedManual?: boolean;
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
			yamlPreview: () => this.yamlString,
			isSavedManualPipeline: props.pipeline?.record.manual === true
		});

		const shouldStartLockedManual =
			props.pipeline?.record.manual === true || props.startLockedManual === true;

		if (shouldStartLockedManual && props.pipeline?.record.yaml) {
			this.stepsBuilder.enterManualMode(props.pipeline.record.yaml, { locked: true });
		}

		this.runtimeOptionsForm = new RuntimeOptionsForm({
			initialData: props.pipeline?.runtime,
			isDisabled: () => this.stepsBuilder.isManualMode
		});

		this.metadataForm = new MetadataForm({
			initialData: props.pipeline?.record,
			onSubmit: async () => {
				if (!this.saveAfterMetadataFormSubmit) return;
				await this.save();
				this.saveAfterMetadataFormSubmit = false;
			}
		});

		beforeNavigate((navigation) => this.guardUnsavedExit(navigation));
	}

	private guardUnsavedExit(navigation: BeforeNavigate) {
		if (this.isSaving) return;

		// A navigation the user already confirmed must pass through, otherwise the
		// confirm()-then-navigate flow below would re-trigger this guard and loop forever.
		if (this.exitConfirmed) {
			this.exitConfirmed = false;
			return;
		}

		if (!this.shouldWarnOnExit() || !navigation.to) return;

		navigation.cancel();
		void this.confirmExit(navigation.to.url);
	}

	private async confirmExit(url: URL) {
		const confirmed = await confirm({
			message:
				m.You_have_unsaved_changes() + '\n' + m.Are_you_sure_you_want_to_exit_the_form(),
			destructive: true
		});
		if (!confirmed) return;

		this.exitConfirmed = true;
		await goto(`${url.pathname}${url.search}${url.hash}`);
	}

	get mode() {
		return this.props.mode;
	}

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

	private exitConfirmed = false;

	async save() {
		if (!this.ensureMetadataBeforeSave()) return;

		const payload = await this.buildSavePayload();
		if (!payload) return;

		if (!(await this.confirmManualSaveIfNeeded())) return;

		await this.persistPipeline(payload);
	}

	private async confirmManualSaveIfNeeded(): Promise<boolean> {
		const isAlreadyManualPipeline =
			this.props.mode === 'edit' && this.props.pipeline?.record.manual === true;

		if (this.stepsBuilder.mode.id !== 'manual' || isAlreadyManualPipeline) {
			return true;
		}

		return confirm({
			message: m.manual_save_warning(),
			confirmLabel: m.Save(),
			destructive: true
		});
	}

	private ensureMetadataBeforeSave(): boolean {
		if (this.metadataForm.value) return true;

		this.metadataForm.isOpen = true;
		if (this.props.mode === 'create') {
			this.saveAfterMetadataFormSubmit = true;
		}
		return false;
	}

	private async buildSavePayload(): Promise<
		Omit<PipelinesFormData, 'owner' | 'canonified_name'> | undefined
	> {
		const { mode } = this.stepsBuilder;
		let yaml: string;

		if (mode.id === 'manual') {
			const result = await mode.editor.validateNow();
			if (!result.ok) {
				toast.error(result.message);
				return;
			}
			yaml = result.value;
		} else {
			try {
				yaml = this.yamlString;
			} catch (e) {
				toast.error(getExceptionMessage(e));
				return;
			}
		}

		return {
			...this.metadataForm.value!,
			yaml,
			...(mode.id === 'manual' ? { manual: true } : {})
		};
	}

	private async persistPipeline(data: Omit<PipelinesFormData, 'owner' | 'canonified_name'>) {
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

	//

	hasChanges = $derived.by(() => {
		const { pipeline } = this.props;

		const runtimeOptionsChanged = !_.isEqual(this.runtimeOptionsForm.value, pipeline?.runtime);
		const nameChanged = this.metadataForm.value?.name !== pipeline?.record.name;
		const descChanged = this.metadataForm.value?.description !== pipeline?.record.description;
		const metadataChanged = nameChanged || descChanged;

		const { mode } = this.stepsBuilder;
		if (mode.id === 'manual') {
			return mode.editor.isDirty || runtimeOptionsChanged || metadataChanged;
		}

		const stepsChanged = !_.isEqual(
			this.stepsBuilder.steps.map(([step]) => step),
			pipeline?.steps.map(([step]) => step)
		);

		return stepsChanged || runtimeOptionsChanged || metadataChanged;
	});

	canSave = $derived.by(() => {
		const { mode } = this.stepsBuilder;
		if (mode.id === 'manual') {
			return this.hasChanges && mode.editor.isValid;
		}
		return this.hasChanges && this.stepsBuilder.steps.length > 0;
	});

	private shouldWarnOnExit(): boolean {
		const { mode } = this.stepsBuilder;
		const manualEditorDirty = mode.id === 'manual' && mode.editor.isDirty;

		return this.hasChanges || manualEditorDirty;
	}
}

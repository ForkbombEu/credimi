<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import BackButton from '$lib/layout/back-button.svelte';
	import { RedoIcon, SaveIcon, UndoIcon } from 'lucide-svelte';

	import T from '@/components/ui-custom/t.svelte';
	import { Button } from '@/components/ui/button';
	import { Separator } from '@/components/ui/separator';
	import { m } from '@/i18n';

	import type { PipelineForm } from './pipeline-form.svelte.js';

	//

	type Props = {
		form: PipelineForm;
	};

	const { form }: Props = $props();
	const metadata = form.metadataForm;
	const activityOptions = form.activityOptionsForm;
	const builder = form.stepsBuilder;

	const isViewMode = $derived(form['props'].mode === 'view');
	const saveButtonText = $derived(isViewMode ? m.Create_record() : m.Save());
</script>

<div class="bg-secondary flex h-screen flex-col gap-4 overflow-hidden px-4 pb-4 pt-2">
	<div class="flex items-center justify-between">
		<div class="flex items-center gap-3">
			<BackButton href="/my/pipelines" class="h-6">
				{m.Back()}
			</BackButton>
			<Separator orientation="vertical" class="self-stretch bg-slate-400" />
			<T tag="h3" class="text-xl">{m.New_pipeline()}</T>
		</div>

		<div>
			<Button variant="ghost" onclick={() => builder.undo()}>
				<UndoIcon />
				{m.Undo()}
			</Button>
			<Button variant="ghost" onclick={() => builder.redo()}>
				<RedoIcon />
				{m.Redo()}
			</Button>
		</div>

		<div class="flex items-center gap-2">
			<metadata.Component form={metadata} />
			<activityOptions.Component form={activityOptions} />
			<Button disabled={!builder.isReady()} onclick={() => form.save()}>
				<SaveIcon />
				{saveButtonText}
			</Button>
		</div>
	</div>

	<builder.Component {builder} />
</div>

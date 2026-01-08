<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Render, type SelfProp } from '$lib/renderable';
	import { PencilIcon, RedoIcon, SaveIcon, UndoIcon } from 'lucide-svelte';

	import { Button } from '@/components/ui/button';
	import { m } from '@/i18n';

	import type { PipelineForm } from './pipeline-form.svelte.js';

	import PipelineFormLayout from './pipeline-form-layout.svelte';

	//

	const { self: form }: SelfProp<PipelineForm> = $props();

	const metadata = form.metadataForm;
	const activityOptions = form.activityOptionsForm;
	const builder = form.stepsBuilder;

	const isViewMode = $derived(form['props'].mode === 'view');
	const saveButtonText = $derived(isViewMode ? m.Create_record() : m.Save());

	const title = $derived.by(() => {
		if (form.mode === 'create') {
			return m.New_pipeline();
		} else if (form.mode === 'edit') {
			return m.Edit_pipeline();
		} else {
			return m.View();
		}
	});
</script>

<PipelineFormLayout {title} class="h-screen overflow-hidden">
	{#snippet topbarMiddle()}
		<Button variant="ghost" onclick={() => builder.undo()}>
			<UndoIcon />
			{m.Undo()}
		</Button>
		<Button variant="ghost" onclick={() => builder.redo()}>
			<RedoIcon />
			{m.Redo()}
		</Button>
	{/snippet}

	{#snippet topbarRight()}
		{#if form.mode === 'create'}
			<Button href="/my/pipelines/new/manual" variant="ghost">
				<PencilIcon />
				{m.manual_mode()}
			</Button>
		{/if}
		<Render item={metadata} />
		<Render item={activityOptions} />
		<Button disabled={!builder.hasSteps()} onclick={() => form.save()}>
			<SaveIcon />
			{saveButtonText}
		</Button>
	{/snippet}

	<Render item={builder} />
</PipelineFormLayout>

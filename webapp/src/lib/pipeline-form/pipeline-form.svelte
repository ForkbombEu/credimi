<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { RedoIcon, SaveIcon, UndoIcon } from '@lucide/svelte';
	import { Render, type SelfProp } from '$lib/renderable';

	import Tooltip from '@/components/ui-custom/tooltip.svelte';
	import { Button } from '@/components/ui/button';
	import { m } from '@/i18n';

	import type { PipelineForm } from './pipeline-form.svelte.js';

	import PipelineFormLayout from './pipeline-form-layout.svelte';

	//

	const { self: form }: SelfProp<PipelineForm> = $props();

	const metadata = form.metadataForm;
	const activityOptions = form.runtimeOptionsForm;
	const builder = form.stepsBuilder;

	const manualMode = $derived(builder.isManualMode);
	const manualTooltip = m.unavailable_in_manual_edit();

	const saveButtonText = $derived(m.Save());

	const title = $derived.by(() => {
		if (form.mode === 'create') {
			return m.New_pipeline();
		} else {
			return m.Edit_pipeline();
		}
	});
</script>

<PipelineFormLayout {title} class="h-screen overflow-hidden">
	{#snippet topbarMiddle()}
		<Tooltip disabled={!manualMode}>
			{#snippet child({ props })}
				<span {...props} class="inline-flex">
					<Button variant="ghost" disabled={manualMode} onclick={() => builder.undo()}>
						<UndoIcon />
						{m.Undo()}
					</Button>
				</span>
			{/snippet}
			{#snippet content()}{manualTooltip}{/snippet}
		</Tooltip>
		<Tooltip disabled={!manualMode}>
			{#snippet child({ props })}
				<span {...props} class="inline-flex">
					<Button variant="ghost" disabled={manualMode} onclick={() => builder.redo()}>
						<RedoIcon />
						{m.Redo()}
					</Button>
				</span>
			{/snippet}
			{#snippet content()}{manualTooltip}{/snippet}
		</Tooltip>
	{/snippet}

	{#snippet topbarRight()}
		<Render item={activityOptions} />
		<Render item={metadata} />
		<Button disabled={!form.canSave} onclick={() => form.save()}>
			<SaveIcon />
			{saveButtonText}
		</Button>
	{/snippet}

	<Render item={builder} />
</PipelineFormLayout>

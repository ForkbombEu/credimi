<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ArrowLeftIcon } from 'lucide-svelte';
	import { fly } from 'svelte/transition';

	import Button from '@/components/ui-custom/button.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import ScrollArea from '@/components/ui/scroll-area/scroll-area.svelte';
	import { m } from '@/i18n';

	import AddStepForm from './add-step-form.svelte';
	import {
		AddStepState,
		IdleState,
		type PipelineBuilder,
		StepType
	} from './pipeline-builder.svelte.js';
	import Column from './utils/column.svelte';
	import { getStepDisplayData } from './utils/display-data';

	//

	let { builder }: { builder: PipelineBuilder } = $props();
</script>

<div class="grid grow grid-cols-3 gap-4">
	<Column title="Add step">
		{#if builder.state instanceof IdleState}
			{@render stepButtons()}
		{:else if builder.state instanceof AddStepState}
			<div in:fly>
				<AddStepForm state={builder.state} />
			</div>
		{/if}

		{#snippet titleRight()}
			{#if builder.state instanceof AddStepState}
				<Button variant="link" class="h-6 !p-0" onclick={() => builder.discardAddStep()}>
					<ArrowLeftIcon />
					{m.Back()}
				</Button>
			{/if}
		{/snippet}
	</Column>

	<Column title={m.Steps_sequence()}>
		<ScrollArea class="grow">
			{#each builder.steps as step (step)}
				<div>
					<h1>{step.type}</h1>
				</div>
			{/each}
		</ScrollArea>
	</Column>

	<Column title={m.YAML_preview()} class="card">YAML PREview</Column>
</div>

<!--  -->

{#snippet stepButtons()}
	<div class="flex flex-col gap-2" in:fly>
		{#each Object.values(StepType) as step (step)}
			{@const { icon, label, textClass, outlineClass } = getStepDisplayData(step)}
			<Button
				variant="outline"
				class={['!justify-start', `hover:${textClass}`, textClass, outlineClass]}
				onclick={() => builder.initAddStep(step)}
			>
				<Icon src={icon} />
				{label}
			</Button>
		{/each}
	</div>
{/snippet}

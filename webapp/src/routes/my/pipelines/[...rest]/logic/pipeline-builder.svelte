<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ArrowLeftIcon } from 'lucide-svelte';
	import { fly } from 'svelte/transition';

	import Button from '@/components/ui-custom/button.svelte';
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

	//

	let { builder }: { builder: PipelineBuilder } = $props();
</script>

<div class="grid grow grid-cols-3 gap-4">
	<Column title="Add step">
		{#if builder.state instanceof IdleState}
			<div class="grid grid-cols-2 gap-2" in:fly>
				{#each Object.values(StepType) as step (step)}
					<Button
						variant="outline"
						class="shrink-0 grow basis-1"
						onclick={() => builder.initAddStep(step)}
					>
						{step}
					</Button>
				{/each}
			</div>
		{:else if builder.state instanceof AddStepState}
			<div in:fly>
				<Button variant="link" class="p-0" onclick={() => builder.discardAddStep()}>
					<ArrowLeftIcon />
					{m.Back()}
				</Button>
				<AddStepForm state={builder.state} />
			</div>
		{/if}
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

<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ArrowLeftIcon } from 'lucide-svelte';

	import Button from '@/components/ui-custom/button.svelte';
	import { m } from '@/i18n';

	import AddStepForm from './add-step-form.svelte';
	import { AddStepState, IdleState, type PipelineBuilder } from './pipeline-builder.svelte.js';
	import { StepType } from './types';

	//

	let { builder }: { builder: PipelineBuilder } = $props();
</script>

<div class="grid grid-cols-3 gap-4">
	<div class="card">
		{#if builder.state instanceof IdleState}
			<div class="grid grid-cols-2">
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
			<Button variant="link" onclick={() => builder.discardAddStep()}>
				<ArrowLeftIcon />
				{m.Back()}
			</Button>
			<AddStepForm state={builder.state} />
		{/if}
	</div>

	<div class="card">
		{#each builder.steps as step (step.id)}
			<div>
				<h1>{step.type}</h1>
			</div>
		{/each}
	</div>

	<div class="card">YAML PREview</div>
</div>

<style lang="postcss">
	.card {
		@apply rounded-lg border bg-white p-6 shadow-sm;
	}
</style>

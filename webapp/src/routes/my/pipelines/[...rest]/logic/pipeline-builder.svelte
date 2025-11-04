<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ArrowLeftIcon } from 'lucide-svelte';
	import { flip } from 'svelte/animate';
	import { fly } from 'svelte/transition';

	import Button from '@/components/ui-custom/button.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import ScrollArea from '@/components/ui/scroll-area/scroll-area.svelte';
	import { m } from '@/i18n';

	import type { PipelineBuilder } from './pipeline-builder.svelte.js';

	import BaseStepFormComponent from './base-step-form.svelte';
	import { BaseStepForm } from './base-step-form.svelte.js';
	import StepCard from './step-card.svelte';
	import { IdleState, StepFormState, StepType } from './types';
	import Column from './utils/column.svelte';
	import { getStepDisplayData } from './utils/display-data';
	import WalletStepFormComponent from './wallet-step-form.svelte';
	import { WalletStepForm } from './wallet-step-form.svelte.js';

	//

	let { builder }: { builder: PipelineBuilder } = $props();
</script>

<div class="grid grow grid-cols-3 gap-4 overflow-hidden">
	<Column title="Add step">
		{#if builder.state instanceof IdleState}
			{@render stepButtons()}
		{:else if builder.state instanceof StepFormState}
			<div class="flex grow flex-col overflow-hidden" in:fly>
				{#if builder.state instanceof WalletStepForm}
					<WalletStepFormComponent form={builder.state} />
				{:else if builder.state instanceof BaseStepForm}
					<BaseStepFormComponent form={builder.state} />
				{/if}
			</div>
		{/if}

		{#snippet titleRight()}
			{#if builder.state instanceof StepFormState}
				<Button variant="link" class="h-6 !p-0" onclick={() => builder.discardAddStep()}>
					<ArrowLeftIcon />
					{m.Back()}
				</Button>
			{/if}
		{/snippet}
	</Column>

	<Column title={m.Steps_sequence()}>
		<ScrollArea class="grow [&>div>div]:space-y-2 [&>div>div]:p-4">
			{#each builder.steps as step (step.id)}
				<div animate:flip={{ duration: 300 }}>
					<StepCard {step} {builder} />
				</div>
			{/each}
		</ScrollArea>
	</Column>

	<Column title={m.YAML_preview()} class="card">YAML PREview</Column>
</div>

<!--  -->

{#snippet stepButtons()}
	<div class="flex flex-col gap-2 p-4" in:fly>
		{#each Object.values(StepType) as step (step)}
			{@const { icon, label, textClass } = getStepDisplayData(step)}
			<Button
				variant="outline"
				class={['!justify-start']}
				onclick={() => builder.initAddStep(step)}
			>
				<Icon src={icon} class={textClass} />
				{label}
			</Button>
		{/each}
	</div>
{/snippet}

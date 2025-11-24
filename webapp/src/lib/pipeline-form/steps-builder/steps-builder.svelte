<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import CodeDisplay from '$lib/layout/codeDisplay.svelte';
	import { String } from 'effect';
	import { ArrowLeftIcon } from 'lucide-svelte';
	import { flip } from 'svelte/animate';
	import { fly } from 'svelte/transition';

	import Button from '@/components/ui-custom/button.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import ScrollArea from '@/components/ui/scroll-area/scroll-area.svelte';
	import { m } from '@/i18n';

	import type { StepsBuilder } from './steps-builder.svelte.js';

	import BaseStepFormComponent from './steps/base-step-form.svelte';
	import { BaseStepForm } from './steps/base-step-form.svelte.js';
	import ConformanceCheckStepFormComponent from './steps/conformance-check-step-form.svelte';
	import { ConformanceCheckStepForm } from './steps/conformance-check-step-form.svelte.js';
	import WalletStepFormComponent from './steps/wallet-step-form.svelte';
	import { WalletStepForm } from './steps/wallet-step-form.svelte.js';
	import { IdleState, StepFormState, StepType } from './types.js';
	import Column from './utils/column.svelte';
	import { getStepDisplayData } from './utils/display-data.js';
	import EmptyState from './utils/empty-state.svelte';
	import StepCard from './utils/step-card.svelte';

	//

	let { builder }: { builder: StepsBuilder } = $props();
</script>

<div class="grid grow grid-cols-3 gap-4 overflow-hidden xl:grid-cols-[max(400px)_max(400px)_1fr]">
	<Column title="Add step">
		{#if builder.state instanceof IdleState}
			{@render stepButtons()}
		{:else if builder.state instanceof StepFormState}
			<div class="flex grow flex-col overflow-hidden" in:fly>
				{#if builder.state instanceof WalletStepForm}
					<WalletStepFormComponent form={builder.state} />
				{:else if builder.state instanceof ConformanceCheckStepForm}
					<ConformanceCheckStepFormComponent form={builder.state} />
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
		{#if builder.steps.length > 0}
			<ScrollArea class="grow [&>div>div]:space-y-2 [&>div>div]:p-4">
				{#each builder.steps as step (step.id)}
					<div animate:flip={{ duration: 300 }}>
						<StepCard {step} {builder} />
					</div>
				{/each}
			</ScrollArea>
		{:else}
			<EmptyState text={m.Pipeline_steps_will_appear_here()} />
		{/if}
	</Column>

	<Column title={m.YAML_preview()} class="card overflow-hidden">
		{#if String.isEmpty(builder.yamlPreview)}
			<EmptyState text={m.YAML_preview_will_appear_here()} />
		{:else}
			<CodeDisplay
				content={builder.yamlPreview}
				language="yaml"
				containerClass="rounded-none grow"
				contentClass="text-sm"
			/>
		{/if}
	</Column>
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

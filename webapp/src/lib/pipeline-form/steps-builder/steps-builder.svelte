<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import CodeDisplay from '$lib/layout/codeDisplay.svelte';
	import { Render, type SelfProp } from '$lib/renderable';
	import { String } from 'effect';
	import { ArrowLeftIcon } from 'lucide-svelte';
	import { flip } from 'svelte/animate';
	import { fly } from 'svelte/transition';

	import Button from '@/components/ui-custom/button.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import * as Resizable from '@/components/ui/resizable/index.js';
	import ScrollArea from '@/components/ui/scroll-area/scroll-area.svelte';
	import { m } from '@/i18n';

	import type { StepsBuilder } from './steps-builder.svelte.js';

	import { configs } from '../steps/index.js';
	import Column from './_partials/column.svelte';
	import EmptyState from './_partials/empty-state.svelte';
	import StepCard from './_partials/step-card.svelte';
	import { debugEntityData, getStepDisplayData } from './_partials/utils.js';

	//

	let { self: builder }: SelfProp<StepsBuilder> = $props();
</script>

<Resizable.PaneGroup direction="horizontal" class="gap-2">
	<Column title="Add step">
		{#if builder.state.id == 'idle'}
			{@render stepButtons()}
		{:else if builder.state.id == 'form'}
			<div class="flex grow flex-col overflow-hidden" in:fly>
				<Render item={builder.state.form} />
			</div>
		{/if}

		{#snippet titleRight()}
			{#if builder.state.id == 'form'}
				<Button variant="link" class="h-6 !p-0" onclick={() => builder.exitFormState()}>
					<ArrowLeftIcon />
					{m.Back()}
				</Button>
			{/if}
		{/snippet}
	</Column>

	<Resizable.Handle class="hover:bg-primary" />

	<Column title={m.Steps_sequence()}>
		{#if builder.steps.length > 0}
			<ScrollArea class="grow [&>div>div]:space-y-2 [&>div>div]:p-4">
				{#each builder.steps as step, index (step)}
					<div animate:flip={{ duration: 300 }}>
						<StepCard {builder} {step} {index} />
					</div>
				{/each}
			</ScrollArea>
		{:else}
			<EmptyState text={m.Pipeline_steps_will_appear_here()} />
		{/if}
	</Column>

	<Resizable.Handle class="hover:bg-primary" />

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
</Resizable.PaneGroup>

<!--  -->

{#snippet stepButtons()}
	<div class="flex flex-col gap-2 p-4" in:fly>
		{#each configs as config (config.id)}
			{@const { icon, labels, classes } = getStepDisplayData(config.id)}
			<Button
				variant="outline"
				class="!justify-start"
				onclick={() => builder.initAddStep(config.id)}
			>
				<Icon src={icon} class={classes.text} />
				{labels.singular}
			</Button>
		{/each}

		<Button variant="outline" class="!justify-start" onclick={() => builder.addDebugStep()}>
			<Icon src={debugEntityData.icon} class={debugEntityData.classes.text} />
			{debugEntityData.labels.singular}
		</Button>
	</div>
{/snippet}

<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { EntityData } from '$lib/global/entities.js';

	import CodeDisplay from '$lib/layout/codeDisplay.svelte';
	import { Render, type SelfProp } from '$lib/renderable';
	import { ArrowLeftIcon } from '@lucide/svelte';
	import { String } from 'effect';
	import { flip } from 'svelte/animate';
	import { fly } from 'svelte/transition';

	import Button from '@/components/ui-custom/button.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import * as Resizable from '@/components/ui/resizable/index.js';
	import ScrollArea from '@/components/ui/scroll-area/scroll-area.svelte';
	import { m } from '@/i18n';

	import type { StepsBuilder } from './steps-builder.svelte.js';

	import * as steps from '../steps';
	import Column from './_partials/column.svelte';
	import EmptyState from './_partials/empty-state.svelte';
	import StepCard from './_partials/step-card.svelte';

	//

	let { self: builder }: SelfProp<StepsBuilder> = $props();

	const { debugEntityData } = steps;
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
		{#each steps.coreConfigs as config (config.use)}
			{@render stepButton(config)}
		{/each}

		<div
			class="text-muted-foreground -mb-1 pt-2 text-[10px] font-medium uppercase tracking-normal"
		>
			{m.utils()}
		</div>

		{@render baseStepButton(debugEntityData, () => builder.addDebugStep())}

		{#each steps.utilsConfigs as config (config.use)}
			{@render stepButton(config)}
		{/each}
	</div>
{/snippet}

{#snippet stepButton(config: steps.AnyConfig)}
	{@render baseStepButton(steps.getDisplayData(config.use), () =>
		builder.initAddStep(config.use)
	)}
{/snippet}

{#snippet baseStepButton(displayData: EntityData, onClick: () => void)}
	<Button variant="outline" class="!justify-start" onclick={onClick}>
		<Icon src={displayData.icon} class={displayData.classes.text} />
		{displayData.labels.singular}
	</Button>
{/snippet}

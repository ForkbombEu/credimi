<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { EntityData } from '$lib/global/entities.js';

	import { HelpCircle, XIcon } from '@lucide/svelte';
	import CodeDisplay from '$lib/layout/codeDisplay.svelte';
	import { Render, type SelfProp } from '$lib/renderable';
	import { String } from 'effect';
	import { flip } from 'svelte/animate';
	import { fly } from 'svelte/transition';

	import Button from '@/components/ui-custom/button.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import * as Resizable from '@/components/ui/resizable/index.js';
	import { m } from '@/i18n';

	import type { StepsBuilder } from './steps-builder.svelte.js';

	import * as steps from '../steps';
	import BulkWalletVersionChange from './_partials/bulk-wallet-version-change.svelte';
	import Column from './_partials/column.svelte';
	import EmptyState from './_partials/empty-state.svelte';
	import StepCard from './_partials/step-card.svelte';

	//

	let { self: builder }: SelfProp<StepsBuilder> = $props();

	const { debugEntityData } = steps;

	const formMode = $derived(builder.mode.id === 'form' ? builder.mode : null);
	const editingIndex = $derived(formMode?.intent === 'edit' ? formMode.stepIndex : undefined);
	const columnTitle = $derived(formMode?.intent === 'edit' ? m.Edit_step() : m.Add_step());
	const stepDocsUrl = $derived(formMode?.config.docsUrl);
</script>

<Resizable.PaneGroup direction="horizontal" class="gap-2">
	<Column title={columnTitle}>
		{#if builder.mode.id == 'idle'}
			{@render stepButtons()}
		{:else if builder.mode.id == 'form'}
			<div class="flex grow flex-col" in:fly>
				<Render item={builder.mode.form} />
				{#if formMode?.intent === 'edit'}
					<div class="mt-auto border-t p-4">
						<Button
							class="w-full"
							disabled={!formMode.form.canSave()}
							onclick={() => formMode.form.commit()}
						>
							{m.Save()}
						</Button>
					</div>
				{/if}
			</div>
		{/if}

		{#snippet titleRight()}
			{#if builder.mode.id == 'form'}
				<div class="flex items-center gap-1">
					{#if stepDocsUrl}
						<IconButton
							variant="outline"
							href={stepDocsUrl}
							target="_blank"
							rel="noopener noreferrer"
							icon={HelpCircle}
							size="xs"
							tooltip={m.Documentation()}
						/>
					{/if}
					<IconButton
						variant="outline"
						onclick={() => builder.exitFormState()}
						icon={XIcon}
						size="xs"
					/>
				</div>
			{/if}
		{/snippet}
	</Column>

	<Resizable.Handle class="hover:bg-primary" />

	<Column title={m.Steps_sequence()}>
		{#snippet titleRight()}
			<BulkWalletVersionChange {builder} />
		{/snippet}

		{#if builder.steps.length > 0}
			<div class="space-y-3 p-4">
				{#each builder.steps as step, index (step)}
					<div animate:flip={{ duration: 300 }}>
						<StepCard {builder} {step} {index} editing={editingIndex === index} />
					</div>
				{/each}
			</div>
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
			class="-mb-1 pt-2 text-[10px] font-medium tracking-normal text-muted-foreground uppercase"
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

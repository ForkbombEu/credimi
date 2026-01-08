<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ArrowDownIcon, ArrowUpIcon, TrashIcon } from 'lucide-svelte';

	import Icon from '@/components/ui-custom/icon.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import Checkbox from '@/components/ui/checkbox/checkbox.svelte';
	import Label from '@/components/ui/label/label.svelte';
	import { m } from '@/i18n/index.js';

	import type { EnrichedStep, StepsBuilder } from '../steps-builder.svelte.js';

	//

	type Props = {
		index: number;
		step: EnrichedStep;
		builder: StepsBuilder;
	};

	let { builder, step, index }: Props = $props();

	const config = $derived(builder.getConfig(step[0].use));
	const display = $derived(config?.display);
	const classes = $derived(display?.classes);
</script>

<div class={['bg-card group overflow-hidden rounded-md border hover:ring', classes?.border]}>
	<div class={['h-1', classes?.bg]}></div>
	<div>
		<div class="flex items-center justify-between py-1 pl-3 pr-1">
			{#if display}
				<div class={['flex items-center gap-1', classes?.text]}>
					<Icon src={display.icon} size={12} />
					<p class="text-xs">{display.labels.singular}</p>
				</div>
			{/if}

			<div class="flex items-center">
				<IconButton
					icon={ArrowUpIcon}
					variant="ghost"
					size="sm"
					onclick={() => builder.shiftStep(index, -1)}
					disabled={!builder.canShiftStep(index, -1)}
				/>
				<IconButton
					icon={ArrowDownIcon}
					variant="ghost"
					size="sm"
					onclick={() => builder.shiftStep(index, 1)}
					disabled={!builder.canShiftStep(index, 1)}
				/>
				<IconButton
					icon={TrashIcon}
					variant="ghost"
					size="sm"
					onclick={() => builder.deleteStep(index)}
				/>
			</div>
		</div>

		<div class="p-3 pb-4 pt-1">
			{#if config && display}
				{@render config.snippet?.({ data: step, display })}
			{:else}
				<h1>{step[0].use}</h1>
			{/if}
			<!-- <Avatar src={step.avatar} fallback={step.name} class="size-8 rounded-lg border" />
			<div class="space-y-1">
				<div class="flex items-center gap-1">
					<h1>{step.name}</h1>
					<CopyButtonSmall
						textToCopy={step.path}
						variant="ghost"
						square
						size="mini"
						class="text-gray-400"
					/>
				</div>
			</div> -->
		</div>

		{#if step[0].use !== 'debug'}
			<Label class="flex cursor-pointer items-center gap-1 bg-slate-50 px-3 py-1">
				<Checkbox
					class="flex size-[10px] items-center justify-center"
					checked={step[0].continue_on_error}
					onCheckedChange={(checked) => builder.setContinueOnError(index, checked)}
				/>
				<span class="text-xs text-slate-500">{m.Continue_on_error()}</span>
			</Label>
		{/if}
	</div>
</div>

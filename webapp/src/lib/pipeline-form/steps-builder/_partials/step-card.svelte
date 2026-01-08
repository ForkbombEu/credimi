<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ArrowDownIcon, ArrowUpIcon, TrashIcon } from 'lucide-svelte';

	import Avatar from '@/components/ui-custom/avatar.svelte';
	import CopyButtonSmall from '@/components/ui-custom/copy-button-small.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import Checkbox from '@/components/ui/checkbox/checkbox.svelte';
	import Label from '@/components/ui/label/label.svelte';
	import { m } from '@/i18n/index.js';

	import type { EnrichedStep, StepsBuilder } from '../steps-builder.svelte.js';

	import { getStepCardData, getStepDisplayData } from './utils.js';

	//

	type Props = {
		index: number;
		step: EnrichedStep;
		builder: StepsBuilder;
	};

	let { builder, step, index }: Props = $props();

	const { classes, labels, icon } = $derived(getStepDisplayData(step[0].use));
	const { title, copyText, avatar } = $derived(getStepCardData(step));
</script>

<div class={['bg-card group overflow-hidden rounded-md border hover:ring', classes.border]}>
	<div class={['h-1', classes?.bg]}></div>
	<div>
		<div class="flex items-center justify-between py-1 pl-3 pr-1">
			<div class={['flex items-center gap-1', classes.text]}>
				<Icon src={icon} size={12} />
				<p class="text-xs">{labels.singular}</p>
			</div>

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

		{#if step[0].use !== 'debug'}
			<div class="flex items-center gap-3 p-3 pb-4 pt-1">
				<Avatar src={avatar} fallback={title} class="size-8 rounded-md border" />
				<div class="space-y-1">
					<div class="flex items-center gap-1">
						<h1>{title}</h1>
						{#if copyText}
							<CopyButtonSmall
								textToCopy={copyText}
								variant="ghost"
								square
								size="mini"
								class="text-gray-400"
							/>
						{/if}
					</div>
				</div>
			</div>
		{/if}

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

<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import * as steps from '$lib/pipeline-form/steps';
	import { TriangleAlert } from '@lucide/svelte';

	import Avatar from '@/components/ui-custom/avatar.svelte';
	import CopyButtonSmall from '@/components/ui-custom/copy-button-small.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import Checkbox from '@/components/ui/checkbox/checkbox.svelte';
	import Label from '@/components/ui/label/label.svelte';
	import { m } from '@/i18n/index.js';

	import { type EnrichedStep, Enrich404Error } from '../types';

	//

	type Props = {
		step: EnrichedStep;
		topRight?: Snippet;
		onContinueOnErrorChange?: (checked: boolean) => void;
		readonly?: boolean;
	};

	let { step, topRight, onContinueOnErrorChange, readonly = false }: Props = $props();

	const { classes, labels, icon } = $derived(steps.getDisplayData(step[0].use));

	const { title, copyText, avatar, meta } = $derived.by(() => {
		if (step[0].use === 'debug') {
			return { title: m.Debug() };
		} else {
			const config = steps.configs.find((c) => c.use === step[0].use);
			if (!config) throw new Error(`Unknown step type: ${step[0].use}`);
			return config.cardData(step[1]);
		}
	});
</script>

<div
	class={[
		'bg-card group flex flex-col overflow-hidden rounded-md border',
		classes.border,
		!readonly && 'hover:ring'
	]}
>
	<div class={['h-1', classes?.bg]}></div>

	<div class="grow">
		<div class="flex items-center justify-between py-1 pl-3 pr-1">
			<div class={['flex items-center gap-1', classes.text]}>
				<Icon src={icon} size={12} />
				<p class="text-xs">{labels.singular}</p>
			</div>

			{@render topRight?.()}
		</div>

		<div class="p-3 pb-4 pt-2">
			{#if step[1] instanceof Enrich404Error || step[1] instanceof Error}
				<div class="rounded-md bg-red-700 p-3 text-white">
					<div class="flex items-center gap-2">
						<TriangleAlert size={12} />
						<p class="text-xs">{step[1].message}</p>
					</div>
					{#if step[1] instanceof Enrich404Error}
						<p class="pt-2 text-xs opacity-60">{step[1].description}</p>
					{/if}
				</div>
			{:else if step[0].use === 'debug'}
				<div class="text-xs text-gray-500">{m.debug_step_description()}</div>
			{:else}
				<div class="flex items-center gap-3">
					<Avatar src={avatar} fallback={title} class="size-8 rounded-sm border" />
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
		</div>

		{#if meta}
			{#each Object.entries(meta) as [key, value] (key)}
				<div class="p-3 pt-0">
					<p class="text-muted-foreground text-xs">
						<span class="font-medium uppercase">{key}:</span>
						{value}
					</p>
				</div>
			{/each}
		{/if}
	</div>

	{#if step[0].use !== 'debug'}
		<Label
			class={[
				'flex items-center gap-1 bg-slate-50 px-3 py-1',
				{ 'cursor-pointer': !readonly }
			]}
		>
			<Checkbox
				class="flex size-[10px] items-center justify-center disabled:cursor-default"
				checked={step[0].continue_on_error}
				disabled={readonly}
				onCheckedChange={(checked) => onContinueOnErrorChange?.(checked)}
			/>
			<span class="text-xs text-slate-500">{m.Continue_on_error()}</span>
		</Label>
	{/if}
</div>

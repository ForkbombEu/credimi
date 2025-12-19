<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import Avatar from '@/components/ui-custom/avatar.svelte';
	import CopyButtonSmall from '@/components/ui-custom/copy-button-small.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import { m } from '@/i18n/index.js';

	import type { BuilderStep } from '../types.js';

	import { getStepDisplayData } from './display-data.js';

	//

	type Props = {
		step: BuilderStep;
		index?: number;
		showContinueOnError?: boolean;
	};

	let { step, index, showContinueOnError = true }: Props = $props();

	const { icon, label, textClass, outlineClass, backgroundClass } = getStepDisplayData(step.type);
</script>

<div class={['bg-card overflow-hidden rounded-md border', outlineClass]}>
	<div class={['h-1', backgroundClass]}></div>
	<div>
		<div class="flex items-center justify-between py-1 pl-3 pr-3">
			<div class="flex items-center gap-2">
				{#if index !== undefined}
					<span class="text-muted-foreground text-xs font-medium">#{index + 1}</span>
				{/if}
				<div class={['flex items-center gap-1', textClass]}>
					<Icon src={icon} size={12} />
					<p class="text-xs">{label}</p>
				</div>
			</div>
			{#if showContinueOnError && step.continueOnError}
				<span class="text-muted-foreground text-xs italic">{m.Continue_on_error()}</span>
			{/if}
		</div>

		<div class="flex items-center gap-3 p-3 pb-4 pt-1">
			<Avatar src={step.avatar} fallback={step.name} class="size-8 rounded-lg border" />
			<div class="min-w-0 flex-1 space-y-1">
				<div class="flex items-center gap-1">
					<h1 class="truncate">{step.name}</h1>
					<CopyButtonSmall
						textToCopy={step.path}
						variant="ghost"
						square
						size="mini"
						class="shrink-0 text-gray-400"
					/>
				</div>
			</div>
		</div>

		<p class="text-muted-foreground block truncate px-3 pb-2 font-mono text-[10px]">
			{step.path}
		</p>
	</div>
</div>

<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ArrowDownIcon, ArrowUpIcon, TrashIcon } from 'lucide-svelte';

	import CopyButtonSmall from '@/components/ui-custom/copy-button-small.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';

	import type { StepsBuilder } from '../steps-builder.svelte.js';
	import type { BuilderStep } from '../types.js';

	import { getStepDisplayData } from './display-data';

	//

	type Props = {
		step: BuilderStep;
		builder: StepsBuilder;
	};

	let { step, builder }: Props = $props();

	const { icon, label, textClass, outlineClass, backgroundClass } = getStepDisplayData(step.type);
	const pathWithoutType = $derived(step.path.split('/').slice(1).join('/'));
</script>

<div class={['bg-card group overflow-hidden rounded-md border hover:ring', outlineClass]}>
	<div class={['h-1', backgroundClass]}></div>
	<div>
		<div class="flex items-center justify-between py-1 pl-3 pr-1">
			<div class={['flex items-center gap-1', textClass]}>
				<Icon src={icon} size={12} />
				<p class="text-xs">{label}</p>
			</div>

			<div class="flex items-center">
				<IconButton
					icon={ArrowUpIcon}
					variant="ghost"
					size="sm"
					onclick={() => builder.shiftStep(step, -1)}
					disabled={!builder.canShiftStep(step, -1)}
				/>
				<IconButton
					icon={ArrowDownIcon}
					variant="ghost"
					size="sm"
					onclick={() => builder.shiftStep(step, 1)}
					disabled={!builder.canShiftStep(step, 1)}
				/>
				<IconButton
					icon={TrashIcon}
					variant="ghost"
					size="sm"
					onclick={() => builder.deleteStep(step)}
				/>
				<!-- <IconButton icon={EllipsisIcon} size="sm" class="flex group-hover:hidden" /> -->
			</div>
		</div>
		<div class="space-y-1 p-3 pt-1">
			<h1>{step.name}</h1>
			<div class="flex items-center gap-1">
				<p class="text-muted-foreground truncate font-mono text-[10px]">
					{pathWithoutType}
				</p>
				<CopyButtonSmall
					textToCopy={pathWithoutType}
					variant="ghost"
					square
					size="mini"
					class="text-gray-400"
				/>
			</div>
		</div>
	</div>
</div>

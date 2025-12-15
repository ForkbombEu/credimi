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

	import type { StepsBuilder } from '../steps-builder.svelte.js';
	import type { BuilderStep } from '../types.js';

	import { getStepDisplayData } from './display-data';

	//

	type Props = {
		step: BuilderStep;
		builder: StepsBuilder;
	};

	let { step = $bindable(), builder }: Props = $props();

	const { icon, label, textClass, outlineClass, backgroundClass } = getStepDisplayData(step.type);
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
			</div>
		</div>

		<div class="flex items-center gap-3 p-3 pb-4 pt-1">
			<Avatar src={step.avatar} fallback={step.name} class="size-8 rounded-lg border" />
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
			</div>
		</div>

		<p class="text-muted-foreground block truncate px-3 pb-2 font-mono text-[10px]">
			{step.path}
		</p>

		<Label class="flex cursor-pointer items-center gap-1 bg-slate-50 px-3 py-1">
			<Checkbox
				class="flex size-[10px] items-center justify-center"
				checked={step.continueOnError}
				onCheckedChange={(checked) => builder.setContinueOnError(step, checked)}
			/>
			<span class="text-xs text-slate-500">{m.Continue_on_error()}</span>
		</Label>
	</div>
</div>

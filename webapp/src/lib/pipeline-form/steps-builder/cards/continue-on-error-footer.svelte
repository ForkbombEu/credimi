<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { EnrichedStep } from '$pipeline-form/shared/enriched-step.js';

	import Checkbox from '@/components/ui/checkbox/checkbox.svelte';
	import Label from '@/components/ui/label/label.svelte';
	import { m } from '@/i18n/index.js';

	//

	type Props = {
		step: EnrichedStep;
		readonly?: boolean;
		onCheckedChange?: (checked: boolean) => void;
		linked?: boolean;
		canLink?: boolean;
		linkReadonly?: boolean;
		onLinkedChange?: (checked: boolean) => void;
	};

	let {
		step,
		readonly = false,
		onCheckedChange,
		linked = false,
		canLink = false,
		linkReadonly = false,
		onLinkedChange
	}: Props = $props();
</script>

{#if step[0].use !== 'debug'}
	<Label
		class={['flex items-center gap-1 bg-slate-50 px-3 py-1', { 'cursor-pointer': !readonly }]}
	>
		<Checkbox
			class="flex size-2.5 items-center justify-center disabled:cursor-default"
			checked={step[0].continue_on_error}
			disabled={readonly}
			onCheckedChange={(value) => onCheckedChange?.(value)}
		/>
		<span class="text-xs text-slate-500">{m.Continue_on_error()}</span>
	</Label>
	{#if canLink && onLinkedChange}
		<Label
			class={[
				'flex items-center gap-1 bg-slate-50 px-3 py-1',
				{ 'cursor-pointer': !linkReadonly }
			]}
		>
			<Checkbox
				class="flex size-2.5 items-center justify-center disabled:cursor-default"
				checked={linked}
				disabled={linkReadonly}
				onCheckedChange={(value) => onLinkedChange(value)}
			/>
			<span class="text-xs text-slate-500">{m.Linked_to_previous_step()}</span>
		</Label>
	{/if}
{/if}

<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';

	import type { ScheduleState } from './types';

	//

	type Props = {
		state: ScheduleState;
	};

	let { state }: Props = $props();
</script>

<T>
	<div
		class={[
			'mr-0.5 inline-block size-2 -translate-y-px rounded-full',
			{
				'bg-green-500': state === 'active',
				'bg-yellow-500': state === 'paused',
				'border bg-gray-50': state === 'not-scheduled'
			}
		]}
	></div>
	{#if state === 'paused'}
		<span class="font-bold">{m.Scheduling_paused()}</span>
	{:else if state === 'active'}
		<span class="font-bold">{m.scheduled()}</span>
	{:else}
		{m.Pipeline_execution_is_not_scheduled()}
	{/if}
</T>

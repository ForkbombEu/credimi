<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { AppleIcon, ClockIcon, SmartphoneIcon } from '@lucide/svelte';
	import { entities } from '$lib/global';
	import { fromEnrichedRecord } from '$lib/pipeline/execution-artifacts';
	import ExecutionArtifactsPreview from '$lib/pipeline/results/execution-artifacts-preview.svelte';

	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';

	import type { ScoreboardRow } from './types';

	import * as conformanceChecks from './columns/conformance-checks.svelte';
	import * as customIntegrations from './columns/custom-integrations.svelte';
	import * as issuers from './columns/issuers.svelte';
	import * as verifiers from './columns/verifiers.svelte';
	import * as EntityDisplay from './entity-display';
	import ExecutionModes from './extras/execution-modes.svelte';
	import { fromScoreboardRow } from './extras/from-scoreboard-row';

	//

	type Props = {
		record: ScoreboardRow;
	};

	let { record }: Props = $props();

	const issuersItems = $derived(issuers.column.fn(record));
	const verifiersItems = $derived(verifiers.column.fn(record));
	const conformanceItems = $derived(conformanceChecks.column.fn(record));
	const customItems = $derived(customIntegrations.column.fn(record));
	const stats = $derived(fromScoreboardRow(record));
	const runners = $derived(record.expand?.mobile_runners ?? []);

	const artifacts = $derived(
		record.expand.latest_execution
			? fromEnrichedRecord(
					record.expand.latest_execution as Parameters<typeof fromEnrichedRecord>[0]
				)
			: undefined
	);

	function platformLabel(type: string | undefined) {
		if (!type) return '';
		return type.startsWith('ios') ? 'iOS' : 'Android';
	}
</script>

{#snippet group(label: string, items: EntityDisplay.Item[])}
	<div class="flex min-w-0 flex-col gap-2.5">
		<T
			tag="h4"
			class="text-[11px] font-semibold tracking-[0.14em] text-muted-foreground uppercase"
		>
			{label}
		</T>
		<EntityDisplay.List {items} layout="full" />
	</div>
{/snippet}

<div class="grid grid-cols-1 gap-x-10 gap-y-6 p-5 lg:grid-cols-[minmax(220px,280px)_1fr]">
	<div class="flex flex-col gap-3">
		<T
			tag="h4"
			class="text-[11px] font-semibold tracking-[0.14em] text-muted-foreground uppercase"
		>
			{m.Execution_mode()}
		</T>
		{#if artifacts}
			<ExecutionArtifactsPreview
				{artifacts}
				variant="preview"
				previewClass="size-14!"
				hideLogs
			/>
		{:else}
			<EntityDisplay.Na />
		{/if}
		{#if stats}
			<ExecutionModes {stats} variant="list" />
		{/if}
		{#if record.minimum_running_time}
			<p class="flex items-center gap-2 text-sm">
				<ClockIcon class="size-4 shrink-0 text-muted-foreground" />
				<span class="font-semibold">{record.minimum_running_time}</span>
				<span>{m.Min_running_time()}</span>
			</p>
		{/if}
		{#if runners.length > 0}
			<ul class="flex flex-col gap-1.5">
				{#each runners as runner (runner)}
					<li class="flex items-center gap-2 text-sm">
						{#if runner.type?.startsWith('ios')}
							<AppleIcon class="size-4 shrink-0 text-muted-foreground" />
						{:else}
							<SmartphoneIcon class="size-4 shrink-0 text-muted-foreground" />
						{/if}
						<span class="font-semibold">{runner.name.trim()}</span>
						{#if platformLabel(runner.type)}
							<span class="text-xs text-muted-foreground"
								>{platformLabel(runner.type)}</span
							>
						{/if}
					</li>
				{/each}
			</ul>
		{/if}
	</div>

	<div class="grid grid-cols-1 gap-x-10 gap-y-6 md:grid-cols-2">
		{@render group(m.Issuance(), issuersItems)}
		{@render group(entities.conformance_checks.labels.plural, conformanceItems)}
		{@render group(m.Presentations(), verifiersItems)}
		{@render group(entities.custom_checks.labels.plural, customItems)}
	</div>
</div>

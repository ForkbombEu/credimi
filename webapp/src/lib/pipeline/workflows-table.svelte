<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ImageIcon, VideoIcon } from '@lucide/svelte';
	import WorkflowsTable from '$lib/workflows/workflows-table.svelte';

	import type { IconComponent } from '@/components/types';

	import Icon from '@/components/ui-custom/icon.svelte';
	import { m } from '@/i18n';

	import type { ExecutionSummary } from './workflows';

	import WorkflowStatusTag from './workflow-status-tag.svelte';

	//

	type Props = {
		workflows: ExecutionSummary[];
	};

	let { workflows }: Props = $props();
</script>

<WorkflowsTable {workflows} hideColumns={['status']}>
	{#snippet header({ Th })}
		<Th>{m.Status()}</Th>
		<Th>{m.Runner()}</Th>
		<Th>{m.Results()}</Th>
	{/snippet}

	{#snippet row({ workflow, Td })}
		{@const typed = workflow as ExecutionSummary}
		{@const runnerNames = (typed.runner_records ?? []).map((r) => r.name)}

		<Td>
			<WorkflowStatusTag {workflow} />
		</Td>

		<Td>
			{#if runnerNames.length > 0}
				{runnerNames.join(', ')}
			{:else}
				<span class="text-muted-foreground opacity-50">N/A</span>
			{/if}
		</Td>

		<Td>
			{#if typed.results && typed.results.length > 0}
				<div class="flex items-center gap-2">
					{#each typed.results as result (result.video)}
						<div class="flex items-center gap-1">
							{@render mediaPreview({
								image: result.screenshot,
								href: result.video,
								icon: VideoIcon
							})}
							{@render mediaPreview({
								image: result.screenshot,
								href: result.screenshot,
								icon: ImageIcon
							})}
						</div>
					{/each}
				</div>
			{:else}
				<span class="text-muted-foreground opacity-50">N/A</span>
			{/if}
		</Td>
	{/snippet}
</WorkflowsTable>

<!--  -->

{#snippet mediaPreview(props: { image: string; href: string; icon: IconComponent })}
	{@const { image, href, icon } = props}
	<!-- eslint-disable svelte/no-navigation-without-resolve -->
	<a
		{href}
		target="_blank"
		class="relative size-10 shrink-0 overflow-hidden rounded-md border border-slate-300 hover:cursor-pointer hover:ring-2"
	>
		<img src={image} alt="Media" class="size-10 shrink-0" />
		<div class="absolute inset-0 flex items-center justify-center bg-black/30">
			<Icon src={icon} class="size-4  text-white" />
		</div>
	</a>
{/snippet}

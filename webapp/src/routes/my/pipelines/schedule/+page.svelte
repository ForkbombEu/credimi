<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ArrowLeft } from '@lucide/svelte';

	import { CollectionManager } from '@/collections-components';
	import Button from '@/components/ui-custom/button.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import * as Table from '@/components/ui/table';
	import { m } from '@/i18n';

	import ScheduleActions from '../_partials/schedule-actions.svelte';
	import ScheduleStateDisplay from '../_partials/schedule-state-display.svelte';
	import {
		type EnrichedSchedule,
		getScheduleState,
		type ScheduleMode,
		scheduleModeLabel
	} from '../_partials/types';
	import { setDashboardNavbar } from '../../+layout@.svelte';

	//

	let { data } = $props();
	let { organization } = $derived(data);

	setDashboardNavbar({ title: m.Scheduled_pipelines(), right: navbarRight });
</script>

{#snippet navbarRight()}
	<Button variant="outline" href="/my/pipelines">
		<ArrowLeft />
		{m.Back_to_pipelines()}
	</Button>
{/snippet}

<div class="mb-8">
	<CollectionManager
		collection="schedules"
		queryOptions={{
			filter: `owner = '${organization.id}'`,
			sort: ['created', 'DESC'],
			expand: ['pipeline']
		}}
		hide={['pagination']}
	>
		{#snippet records({ records: schedules, reloadRecords })}
			<Table.Root>
				<Table.Header>
					<Table.Row>
						<Table.Head>
							{m.Pipeline()}
						</Table.Head>
						<Table.Head>
							{m.Status()}
						</Table.Head>
						<Table.Head>
							{m.interval()}
						</Table.Head>
						<Table.Head>
							{m.next_run()}
						</Table.Head>
						<Table.Head>
							{m.Actions()}
						</Table.Head>
					</Table.Row>
				</Table.Header>
				<Table.Body>
					{#each schedules as schedule, index (schedule.id)}
						{@const pipeline = schedule.expand?.pipeline}
						{@const status = (schedule as EnrichedSchedule).__schedule_status__}
						{@const mode = schedule.mode as ScheduleMode}
						{@const state = getScheduleState(schedule as EnrichedSchedule)}
						{#if pipeline}
							<Table.Row>
								<Table.Cell>
									<T>{pipeline.name}</T>
								</Table.Cell>
								<Table.Cell>
									<ScheduleStateDisplay {state} />
								</Table.Cell>
								<Table.Cell>
									<T>{scheduleModeLabel(mode)}</T>
								</Table.Cell>
								<Table.Cell>
									{#if state === 'active'}
										{status.next_action_time}
									{:else}
										<span class="text-slate-300">N/A</span>
									{/if}
								</Table.Cell>
								<Table.Cell class="-translate-x-3 !py-2">
									<ScheduleActions
										bind:schedule={schedules[index] as EnrichedSchedule}
										onCancel={() => {
											reloadRecords();
										}}
									/>
								</Table.Cell>
							</Table.Row>
						{/if}
					{/each}
				</Table.Body>
			</Table.Root>
		{/snippet}

		{#snippet emptyState({ EmptyState })}
			<EmptyState title={m.No_items_here()} />
		{/snippet}
	</CollectionManager>
</div>

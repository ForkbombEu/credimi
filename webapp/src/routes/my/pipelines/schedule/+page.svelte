<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ArrowLeft, Ellipsis } from '@lucide/svelte';
	import BlueButton from '$lib/layout/blue-button.svelte';
	import { omit } from 'lodash';

	import { CollectionManager } from '@/collections-components';
	import Button from '@/components/ui-custom/button.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
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
							Runners
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
									{#if status.runners && status.runners.length > 0}
										<div class="flex flex-wrap gap-1">
											{#each status.runners as runner}
												<span class="rounded bg-slate-100 px-2 py-1 text-xs">
													{runner.name}
												</span>
											{/each}
										</div>
									{:else}
										<span class="text-slate-400">-</span>
									{/if}
								</Table.Cell>
								<Table.Cell>
									<T>{scheduleModeLabel(mode)}</T>
								</Table.Cell>
								<Table.Cell>
									{#if state === 'active'}
										{status.next_action_time}
									{:else}
										<span class="text-slate-400">N/A</span>
									{/if}
								</Table.Cell>
								<Table.Cell class="-translate-x-3 py-2!">
									<ScheduleActions
										bind:schedule={schedules[index] as EnrichedSchedule}
										hideDetailsInPopover
										onCancel={() => {
											reloadRecords();
										}}
									>
										{#snippet trigger({ props })}
											<BlueButton {...omit(props, 'class')}>
												<Icon src={Ellipsis} />
												{m.Manage()}
											</BlueButton>
										{/snippet}
									</ScheduleActions>
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

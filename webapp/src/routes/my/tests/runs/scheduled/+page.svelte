<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { runWithLoading } from '$lib/utils';
	import { loadScheduledWorkflows, scheduleActions } from '$lib/workflows/schedule';
	import { getDayLabel } from '$lib/workflows/schedule.utils';
	import { ArrowLeft, EllipsisVerticalIcon } from 'lucide-svelte';

	import Button from '@/components/ui-custom/button.svelte';
	import DropdownMenu from '@/components/ui-custom/dropdown-menu.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import * as Table from '@/components/ui/table';
	import { m } from '@/i18n';

	import { setDashboardNavbar } from '../../../+layout@.svelte';

	//

	let { data } = $props();
	let {
		scheduledWorkflows: { schedules }
	} = $derived(data);

	setDashboardNavbar({
		title: `${m.Test_runs()} / ${m.Scheduled_workflows()}`,
		right: navbarRight
	});

	async function refreshSchedules() {
		// We need to wait in order for temporal to apply the changes
		return new Promise<void>((resolve) => {
			setTimeout(async () => {
				schedules = (await loadScheduledWorkflows()).schedules;
				resolve();
			}, 3000);
		});
	}
</script>

{#snippet navbarRight()}
	<Button variant="outline" href="/my/tests/runs">
		<ArrowLeft />
		{m.Back_to_workflows()}
	</Button>
{/snippet}

<div class="space-y-4">
	<T tag="h3">{m.Scheduled_workflows()}</T>

	<Table.Root>
		<Table.Header>
			<Table.Row>
				<Table.Head>
					{m.Workflow()}
				</Table.Head>
				<Table.Head>
					{m.interval()}
				</Table.Head>
				<Table.Head>
					{m.next_run()}
				</Table.Head>
				<Table.Head>
					{m.Status()}
				</Table.Head>
				<Table.Head>
					{m.Actions()}
				</Table.Head>
			</Table.Row>
		</Table.Header>
		<Table.Body>
			{#each schedules ?? [] as workflow (workflow.id)}
				<Table.Row>
					<Table.Cell>
						<T>{workflow.display_name}</T>
						{#if workflow.display_name != workflow.workflowType?.name}
							<T class="text-muted-foreground text-sm">
								{workflow.workflowType?.name}
							</T>
						{/if}
					</Table.Cell>
					<Table.Cell>
						<T>
							{workflow.schedule_mode?.mode}
							{#if workflow.schedule_mode?.mode === 'weekly'}
								<span class="text-muted-foreground text-sm">
									({getDayLabel(workflow.schedule_mode?.day)?.toLowerCase()})
								</span>
							{:else if workflow.schedule_mode?.mode === 'monthly'}
								<span class="text-muted-foreground text-sm">
									({m.Day().toLowerCase()}: {workflow.schedule_mode?.day})
								</span>
							{/if}
						</T>
					</Table.Cell>
					<Table.Cell>
						{workflow.next_action_time}
					</Table.Cell>
					<Table.Cell>
						{workflow.paused ? m.Paused() : m.Running()}
					</Table.Cell>
					<Table.Cell>
						<DropdownMenu
							buttonVariants={{ size: 'icon', variant: 'ghost' }}
							items={scheduleActions.map((action) => ({
								label: action.label,
								icon: action.icon,
								disabled: !action.disabled ? false : action.disabled(workflow),
								onclick: () =>
									runWithLoading({
										fn: async () => {
											await action.action(workflow.id);
											await refreshSchedules();
										},
										successText: action.successMessage,
										showSuccessToast: true
									})
							}))}
						>
							{#snippet trigger()}
								<EllipsisVerticalIcon />
							{/snippet}
						</DropdownMenu>
					</Table.Cell>
				</Table.Row>
			{:else}
				<Table.Row>
					<Table.Cell colspan={3} class="text-center text-slate-300 bg-slate-50 py-20">
						{m.No_scheduled_workflows()}
					</Table.Cell>
				</Table.Row>
			{/each}
		</Table.Body>
	</Table.Root>
</div>

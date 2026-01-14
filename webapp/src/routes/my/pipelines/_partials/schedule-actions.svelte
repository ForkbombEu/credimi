<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<!-- <Table.Row>
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
</Table.Row> -->

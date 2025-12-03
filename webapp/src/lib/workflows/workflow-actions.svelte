<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { WorkflowStatus } from '@forkbombeu/temporal-ui/dist/types/workflows';
	import type { Snippet } from 'svelte';
	import type { ClassValue } from 'svelte/elements';

	import { runWithLoading } from '$lib/layout/global-loading.svelte';
	import { Code, Hourglass, XIcon } from 'lucide-svelte';

	import type { IconComponent } from '@/components/types';
	import type { buttonVariants } from '@/components/ui/button';

	import Button from '@/components/ui-custom/button.svelte';
	import DropdownMenu from '@/components/ui-custom/dropdown-menu.svelte';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	import ScheduleWorkflowForm from './schedule-workflow-form.svelte';

	//

	type Props = {
		workflow: { workflowId: string; runId: string; status: WorkflowStatus; name: string };
		mode: 'buttons' | 'dropdown';
		containerClass?: ClassValue;
		dropdownTrigger?: Snippet;
		dropdownTriggerVariants?: Parameters<typeof buttonVariants>[0];
	};

	let { workflow, containerClass, mode, dropdownTrigger, dropdownTriggerVariants }: Props =
		$props();

	//

	let isScheduleWorkflowFormOpen = $state(false);

	type WorkflowAction = {
		label: string;
		icon: IconComponent;
		onclick: (workflow: Props['workflow']) => void | Promise<void>;
		disabled?: (workflow: Props['workflow']) => boolean;
	};

	const actions: WorkflowAction[] = [
		{
			label: m.Terminate(),
			icon: XIcon,
			onclick: ({ workflowId, runId }) =>
				runWithLoading({
					fn: async () => {
						await pb.send(`/api/my/checks/${workflowId}/runs/${runId}/terminate`, {
							method: 'POST'
						});
					}
				}),
			disabled: (workflow) => workflow.status !== 'Running'
		},
		{
			label: m.Schedule(),
			icon: Hourglass,
			onclick: () => {
				isScheduleWorkflowFormOpen = true;
			}
		},
		{
			label: m.Swagger(),
			icon: Code,
			onclick: () => {},
			disabled: () => true
		}
	];
</script>

{#if mode === 'buttons'}
	<div class={['flex gap-2', containerClass]}>
		{#each actions as action (action.label)}
			<Button
				variant="outline"
				onclick={() => action.onclick(workflow)}
				disabled={action.disabled?.(workflow)}
				size="sm"
			>
				<action.icon />
				{action.label}
			</Button>
		{/each}
	</div>
{:else if mode === 'dropdown'}
	<DropdownMenu
		buttonVariants={dropdownTriggerVariants}
		items={actions.map((action) => ({
			label: action.label,
			icon: action.icon,
			onclick: () => action.onclick(workflow),
			disabled: action.disabled?.(workflow)
		}))}
		trigger={dropdownTrigger}
	/>
{/if}

<ScheduleWorkflowForm
	workflowID={workflow.workflowId}
	runID={workflow.runId}
	workflowName={workflow.name}
	bind:isOpen={isScheduleWorkflowFormOpen}
/>

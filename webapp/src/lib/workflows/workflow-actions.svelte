<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { WorkflowStatus } from '@forkbombeu/temporal-ui/dist/types/workflows';
	import type { Snippet } from 'svelte';
	import type { ClassValue } from 'svelte/elements';

	import { Code, XIcon } from '@lucide/svelte';
	import { runWithLoading } from '$lib/layout/global-loading.svelte';
	import { toast } from 'svelte-sonner';

	import type { IconComponent } from '@/components/types';
	import type { buttonVariants } from '@/components/ui/button';

	import Button from '@/components/ui-custom/button.svelte';
	import DropdownMenu from '@/components/ui-custom/dropdown-menu.svelte';
	import { emitQueueCancelRequested } from '$lib/pipeline/queue-events';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	//

	type WorkflowQueueMeta = {
		ticket_id: string;
		runner_ids: string[];
	};

	type Props = {
		workflow: {
			workflowId: string;
			runId: string;
			status: WorkflowStatus | null;
			name: string;
			queue?: WorkflowQueueMeta;
		};
		mode: 'buttons' | 'dropdown';
		containerClass?: ClassValue;
		dropdownTrigger?: Snippet<[{ props: Record<string, unknown> }]>;
		dropdownTriggerContent?: Snippet;
		dropdownTriggerVariants?: Parameters<typeof buttonVariants>[0];
	};

	let {
		workflow,
		containerClass,
		mode,
		dropdownTrigger,
		dropdownTriggerContent,
		dropdownTriggerVariants
	}: Props = $props();

	//

	type WorkflowAction = {
		label: string;
		icon: IconComponent;
		onclick: (workflow: Props['workflow']) => void | Promise<void>;
		disabled?: (workflow: Props['workflow']) => boolean;
	};

	const actions: WorkflowAction[] = [
		{
			label: m.Cancel(),
			icon: XIcon,
			onclick: ({ workflowId, runId, queue }) =>
				runWithLoading({
					fn: async () => {
						if (queue) {
							emitQueueCancelRequested(queue.ticket_id);
							const runnerIDs = queue.runner_ids ?? [];
							const params =
								runnerIDs.length > 0
									? `?runner_ids=${encodeURIComponent(runnerIDs.join(','))}`
									: '';
							await pb.send(`/api/pipeline/queue/${queue.ticket_id}${params}`, {
								method: 'DELETE'
							});
							toast.message('Queue canceled');
							return;
						}

						await pb.send(`/api/my/checks/${workflowId}/runs/${runId}/cancel`, {
							method: 'POST'
						});
					},
					showSuccessToast: queue ? false : undefined
				}),
			disabled: (workflow) => {
				if (workflow.queue) return false;
				if (!workflow.status) return true;
				return workflow.status !== 'Running';
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
				disabled={action.disabled ? action.disabled(workflow) : false}
				size="sm"
			>
				<action.icon />
				{action.label}
			</Button>
		{/each}
	</div>
{:else if mode === 'dropdown'}
	<DropdownMenu
		triggerVariants={dropdownTriggerVariants}
		items={actions.map((action) => ({
			label: action.label,
			icon: action.icon,
			onclick: () => action.onclick(workflow),
			disabled: action.disabled ? action.disabled(workflow) : false
		}))}
		triggerContent={dropdownTriggerContent}
		trigger={dropdownTrigger}
	/>
{/if}

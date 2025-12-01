<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ArrowLeft } from 'lucide-svelte';

	import Button from '@/components/ui-custom/button.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import * as Table from '@/components/ui/table';
	import { m } from '@/i18n';

	import { setDashboardNavbar } from '../../../+layout@.svelte';

	//

	let { data } = $props();
	let { scheduledWorkflows } = $derived(data);

	setDashboardNavbar({
		title: `${m.Test_runs()} / ${m.Scheduled_workflows()}`,
		right: navbarRight
	});
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
			</Table.Row>
		</Table.Header>
		<Table.Body>
			{#each scheduledWorkflows as scheduledWorkflow (scheduledWorkflow)}
				<Table.Row>
					<Table.Cell>
						{scheduledWorkflow}
					</Table.Cell>
				</Table.Row>
			{/each}
		</Table.Body>
	</Table.Root>
</div>

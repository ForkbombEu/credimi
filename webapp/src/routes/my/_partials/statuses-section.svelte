<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script module lang="ts">
	import { WorkflowStatus } from '@forkbombeu/temporal-ui';
	import { page } from '$app/state';
	import { TemporalI18nProvider, workflowStatuses } from '$lib/temporal';
	import { WORKFLOW_STATUS_QUERY_PARAM } from '$lib/workflows';
	import { slide } from 'svelte/transition';

	import * as Sidebar from '@/components/ui/sidebar/index.js';

	export { WorkflowStatusesSidebarSection };

	//

	const statuses = workflowStatuses.filter((status) => status !== null);
</script>

{#snippet WorkflowStatusesSidebarSection()}
	<TemporalI18nProvider>
		{#if page.url.pathname.endsWith('/my/tests/runs')}
			<div transition:slide class="pl-4">
				{#each statuses as status (status)}
					{@const isActive = false}
					<Sidebar.MenuItem>
						<Sidebar.MenuButton {isActive}>
							{#snippet child({ props })}
								{@const href = `/my/tests/runs?${WORKFLOW_STATUS_QUERY_PARAM}=${status}`}
								<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
								<a {href} {...props}>
									<WorkflowStatus {status} />
								</a>
							{/snippet}
						</Sidebar.MenuButton>
					</Sidebar.MenuItem>
				{/each}
			</div>
		{/if}
	</TemporalI18nProvider>
{/snippet}

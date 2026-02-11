<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Pipeline } from '$lib';
	import { isWorkflowStatus, TemporalI18nProvider } from '$lib/temporal';

	import type { SidebarItemComponentProps } from '../sidebar';

	//

	let { title }: SidebarItemComponentProps = $props();

	const status = $derived.by(() => {
		if (isWorkflowStatus(title) || title === Pipeline.Workflows.QUEUED_STATUS) return title;
		return undefined;
	});
</script>

{#if status}
	<TemporalI18nProvider>
		<Pipeline.Workflows.StatusTag {status} />
	</TemporalI18nProvider>
{/if}

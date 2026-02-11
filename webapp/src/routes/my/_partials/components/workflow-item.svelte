<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { TemporalI18nProvider } from '$lib/temporal';

	import type { SidebarItemComponentProps } from '../sidebar';

	import { ExtendedStatusTag, isExtendedWorkflowStatus } from '../../tests/runs/_partials';

	//

	let { title }: SidebarItemComponentProps = $props();

	const status = $derived.by(() => {
		if (isExtendedWorkflowStatus(title)) return title;
		return undefined;
	});
</script>

{#if status}
	<TemporalI18nProvider>
		<ExtendedStatusTag {status} />
	</TemporalI18nProvider>
{/if}

<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { SidebarItemComponentProps } from '../sidebar';

	import { ExtendedStatusTag, isExtendedWorkflowStatus } from '../../tests/runs/_partials';

	let { title }: SidebarItemComponentProps = $props();

	const status = $derived.by(() => {
		if (isExtendedWorkflowStatus(title)) return title;
		return undefined;
	});
</script>

{#if status}
	<ExtendedStatusTag {status} />
{/if}

<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	export const editFormState = $state({
		open: false
	});
</script>

<script lang="ts">
	import type { Snippet } from 'svelte';

	import Sheet from '@/components/ui-custom/sheet.svelte';

	type Props = {
		children: Snippet<[{ closeSheet: () => void }]>;
	};

	let { children }: Props = $props();
</script>

<Sheet bind:open={editFormState.open} hideTrigger>
	{#snippet content({ closeSheet })}
		{@render children({ closeSheet })}
	{/snippet}
</Sheet>

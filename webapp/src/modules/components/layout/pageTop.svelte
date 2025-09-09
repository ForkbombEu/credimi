<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	interface Props {
		children?: import('svelte').Snippet;
		bottom?: import('svelte').Snippet;
	}

	let { children, bottom }: Props = $props();
</script>

<div class="border-b bg-background">
	<div class="space-y-6 px-10 py-6">
		{@render children?.()}
	</div>
	{#if bottom}
		<div class="px-10">
			{@render bottom?.()}
		</div>
	{/if}
</div>

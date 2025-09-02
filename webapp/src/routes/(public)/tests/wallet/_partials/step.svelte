<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->
<script lang="ts">
	import type { Snippet } from 'svelte';

	import T from '@/components/ui-custom/t.svelte';

	type Props = {
		n?: string;
		text: string;
		children?: Snippet;
		class?: string;
	};
	const { n, text, children, class: className }: Props = $props();
</script>

<div class={['rounded-xl bg-secondary p-4', className]}>
	<div class="flex items-center gap-4">
		{#if n}
			<div
				class="flex size-12 shrink-0 items-center justify-center rounded-full bg-primary text-lg font-semibold text-primary-foreground"
			>
				<p>{n}</p>
			</div>
		{/if}
		<T class="font-semibold text-primary">{text}</T>
	</div>
	{@render children?.()}
</div>

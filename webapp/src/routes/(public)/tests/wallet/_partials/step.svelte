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

<div class={['bg-secondary rounded-xl p-4', className]}>
	<div class="flex items-center gap-4">
		{#if n}
			<div
				class="bg-primary text-primary-foreground flex size-12 shrink-0 items-center justify-center rounded-full text-lg font-semibold"
			>
				<p>{n}</p>
			</div>
		{/if}
		<T class="text-primary font-semibold">{text}</T>
	</div>
	{@render children?.()}
</div>

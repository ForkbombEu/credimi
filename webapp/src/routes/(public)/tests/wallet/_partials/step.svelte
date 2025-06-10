<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->
<script>
	import T from '@/components/ui-custom/t.svelte';

	const { n, text, children } = $props();
</script>

<div class="step-container">
	<div class="flex items-center gap-4">
		<div
			class="bg-primary text-primary-foreground flex size-12 shrink-0 items-center justify-center rounded-full text-lg font-semibold"
		>
			<p>{n}</p>
		</div>
		<T class="text-primary font-semibold">{text}</T>
	</div>
	{@render children?.()}
</div>

<style lang="postcss">
	.step-container {
		@apply bg-secondary rounded-xl p-4;
	}
</style>

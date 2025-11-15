<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import T from '@/components/ui-custom/t.svelte';

	type Props = {
		label: string;
		children?: Snippet;
		class?: string;
	};

	let { label, children, class: className }: Props = $props();
</script>

<div class={['space-y-1', className]}>
	<T class="text-muted-foreground text-sm">{label}</T>
	{@render children?.()}
</div>

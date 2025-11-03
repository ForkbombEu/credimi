<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import T from '@/components/ui-custom/t.svelte';

	type Props = {
		label: string;
		children: Snippet;
	};

	let { label, children }: Props = $props();
</script>

<div class="space-y-1">
	<T class="text-muted-foreground text-sm">{label}</T>
	{@render children?.()}
</div>

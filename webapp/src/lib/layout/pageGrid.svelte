<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import { cn } from '@/components/ui/utils';

	type Props = {
		class?: string;
		children?: Snippet;
	};

	let { class: className = '', children }: Props = $props();
</script>

<div class={cn('grid grid-cols-1 gap-4 md:grid-cols-2 md:gap-6', className)}>
	{@render children?.()}
</div>

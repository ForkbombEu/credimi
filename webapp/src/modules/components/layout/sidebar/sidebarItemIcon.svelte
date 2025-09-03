<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import type { IconComponent } from '@/components/types';

	import Icon from '@/components/ui-custom/icon.svelte';

	const { children, src }: { children?: Snippet; src?: IconComponent } = $props();
</script>

{#if children}
	<div class="flex size-4 items-center justify-center overflow-visible">
		{@render children?.()}
	</div>
{:else if src}
	<Icon {src} size={16} />
{/if}

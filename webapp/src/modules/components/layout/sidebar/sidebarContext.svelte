<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import { useSidebar } from '@/components/ui/sidebar';

	interface Props {
		content: Snippet<[{ isMobile: boolean }]>;
	}

	let { content }: Props = $props();

	const sidebar = useSidebar();
</script>

{@render content({ isMobile: sidebar.isMobile })}

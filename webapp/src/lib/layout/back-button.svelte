<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import { ArrowLeft } from 'lucide-svelte';

	import Button from '@/components/ui-custom/button.svelte';

	type Props = {
		href: string;
		children: Snippet;
		class?: string;
	};

	let { href, children, class: className = '' }: Props = $props();
</script>

<Button {href} variant="link" class="p-0 {className}">
	<ArrowLeft size={16} />
	{@render children()}
</Button>

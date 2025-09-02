<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import { localizeHref } from '@/i18n';

	//

	type Props = {
		children: Snippet;
		href: string;
		class?: string;
	};

	const { children, href, class: className = '' }: Props = $props();
</script>

<a
	href={localizeHref(href)}
	class="border-primary bg-card text-card-foreground ring-primary block rounded-lg border p-6 shadow-sm transition-all hover:-translate-y-2 hover:ring-2 {className}"
>
	{@render children()}
</a>

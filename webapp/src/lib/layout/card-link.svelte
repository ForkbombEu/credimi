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
	class="block rounded-lg border border-primary bg-card p-6 text-card-foreground shadow-sm ring-primary transition-all hover:-translate-y-2 hover:ring-2 {className}"
>
	{@render children()}
</a>

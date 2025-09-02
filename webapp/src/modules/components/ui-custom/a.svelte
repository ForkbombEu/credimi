<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLAnchorAttributes } from 'svelte/elements';

	import { localizeHref } from '@/i18n';

	import { cn } from '../ui/utils';

	type Props = HTMLAnchorAttributes & {
		children?: Snippet;
	};

	let { class: className, href, children, ...rest }: Props = $props();
</script>

<a
	href={href ? localizeHref(href) : undefined}
	class={cn(className, 'text-primary hover:underline')}
	{...rest}
>
	{@render children?.()}
</a>

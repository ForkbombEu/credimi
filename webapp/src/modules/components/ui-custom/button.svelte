<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Button, type ButtonProps } from '@/components/ui/button';
	import { cn } from '@/components/ui/utils.js';
	import { localizeHref } from '@/i18n';

	let { class: className, href, children, ...restProps }: ButtonProps = $props();

	const properHref = $derived.by(() => {
		if (href?.startsWith('#')) return href;
		else if (typeof href === 'string') return localizeHref(href);
		return undefined;
	});
</script>

<Button href={properHref} class={cn(className, 'cursor-pointer')} {...restProps}>
	{@render children?.()}
</Button>

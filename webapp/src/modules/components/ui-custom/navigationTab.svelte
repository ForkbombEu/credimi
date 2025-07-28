<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	export type Props = LinkWithIcon & {
		notification?: boolean;
	};
</script>

<script lang="ts">
	import { page } from '$app/state';
	import Icon from '@/components/ui-custom/icon.svelte';
	import { cn } from '@/components/ui/utils';
	import type { LinkWithIcon } from '../types';
	import { localizeHref } from '@/i18n';

	//

	let { href, icon, title, notification, ...rest }: Props = $props();

	//

	const isActive = $derived(page.url.pathname == href);

	const classes = $derived(
		cn(
			rest.class,
			'relative inline-block text-sm font-medium text-center border-b-2 flex items-center justify-center whitespace-nowrap',
			'p-2 lg:px-4 lg:py-3', // Responsive padding
			'text-xs lg:text-sm', // Responsive text size
			'min-w-[2.5rem] lg:min-w-fit', // Minimum width for icon-only view on mobile
			{
				'border-transparent hover:border-primary/20': !isActive,
				'text-primary border-primary border-b-2 bg-secondary rounded-t-sm': isActive
			}
		)
	);
</script>

<a href={href ? localizeHref(href) : undefined} {...rest} role="tab" class={classes}>
	{#if icon}
		<Icon src={icon} class="block lg:mr-2"></Icon>
	{/if}
	<span class="hidden lg:inline">{title}</span>
	{#if notification}
		<div
			class="text-primary-600 absolute right-1 top-1 size-2 rounded-full bg-red-500 text-xs shadow-md ring-1 ring-white"
		></div>
	{/if}
</a>

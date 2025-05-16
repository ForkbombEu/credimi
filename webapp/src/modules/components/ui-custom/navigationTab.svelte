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
			'relative inline-block text-sm font-medium text-center p-4 py-3 border-b-2 flex items-center whitespace-nowrap',
			{
				'border-transparent hover:border-primary/20': !isActive,
				'text-primary border-primary border-b-2 bg-secondary rounded-t-sm': isActive
			}
		)
	);
</script>

<a href={href ? localizeHref(href) : undefined} {...rest} role="tab" class={classes}>
	{#if icon}
		<Icon src={icon} mr></Icon>
	{/if}
	{title}
	{#if notification}
		<div
			class="text-primary-600 absolute right-1 top-1 size-2 rounded-full bg-red-500 text-xs shadow-md ring-1 ring-white"
		></div>
	{/if}
</a>

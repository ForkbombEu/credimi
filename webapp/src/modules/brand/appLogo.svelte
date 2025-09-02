<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { appName } from '@/brand';
	import { cn } from '@/components/ui/utils';
	import { localizeHref } from '@/i18n';

	interface Props {
		/*
		 * The url to navigate to when the logo is clicked.
		 */
		href?: string;
		version?: 'icon' | 'full';
	}

	let { href = '/', version = 'icon' }: Props = $props();

	const logo =
		version === 'icon' ? '/logos/credimi_logo.svg' : '/logos/credimi_logo-transp_white.svg';
</script>

<a href={localizeHref(href)} class="flex items-center justify-center gap-2">
	<img
		src={logo}
		class={cn('object-cover', version === 'icon' ? 'h-14 w-14' : 'h-10 w-auto')}
		alt={appName}
	/>
</a>

<!--  
    @component 
    Simple logo component. Change logo.svg to your logo.

    ### Props
    - `href` - The url to navigate to when the logo is clicked.
-->

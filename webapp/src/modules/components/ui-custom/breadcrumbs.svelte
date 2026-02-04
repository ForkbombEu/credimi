<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import type { CalcBreadcrumbsOptions } from './breadcrumbs';
	export { type CalcBreadcrumbsOptions as BreadcrumbsOptions };
</script>

<script lang="ts">
	import { page } from '$app/state';
	import { Home } from '@lucide/svelte';

	import type { Link } from '@/components/types';

	import * as Breadcrumb from '@/components/ui/breadcrumb/index.js';

	import { calcBreadcrumbs } from './breadcrumbs';
	import Icon from './icon.svelte';

	//

	interface Props {
		options?: Partial<CalcBreadcrumbsOptions>;
	}

	const { options = {} }: Props = $props();

	//

	let breadcrumbs: Link[] = $state([]);

	$effect(() => {
		calcBreadcrumbs(page, options).then((newBreadcrumbs) => (breadcrumbs = newBreadcrumbs));
	});
</script>

<Breadcrumb.Root>
	<Breadcrumb.List>
		{#each breadcrumbs as { href, title }, i}
			{@const isHome = i == 0}
			{@const isLast = i == breadcrumbs.length - 1}

			{#if !isHome}
				<Breadcrumb.Item>
					<Breadcrumb.Link {href}>{title}</Breadcrumb.Link>
				</Breadcrumb.Item>
			{:else}
				<Breadcrumb.Item>
					<Breadcrumb.Link {href}><Icon src={Home} /></Breadcrumb.Link>
				</Breadcrumb.Item>
			{/if}

			<!-- {#if isLast}
			<Breadcrumb.Item>
			<Breadcrumb.Page>Breadcrumb</Breadcrumb.Page>
			</Breadcrumb.Item>
			{/if} -->

			{#if !isLast}
				<Breadcrumb.Separator />
			{/if}
		{/each}
	</Breadcrumb.List>
</Breadcrumb.Root>

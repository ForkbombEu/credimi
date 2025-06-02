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
	import { calcBreadcrumbs } from './breadcrumbs';
	import type { Link } from '@/components/types';
	import * as Breadcrumb from '@/components/ui/breadcrumb/index.js';
	import Icon from './icon.svelte';
	import { Home } from 'lucide-svelte';

	//

	interface Props {
		options?: Partial<CalcBreadcrumbsOptions>;
		contentClass?: string;
		activeLinkClass?: string;
	}

	const { options = {}, activeLinkClass, contentClass }: Props = $props();

	//

	let breadcrumbs: Link[] = $state([]);

	$effect(() => {
		calcBreadcrumbs(page, options).then((newBreadcrumbs) => (breadcrumbs = newBreadcrumbs));
	});
</script>

{#if breadcrumbs.length}
	<Breadcrumb.Root class={contentClass}>
		<Breadcrumb.List>
			<Breadcrumb.Item>
				<Breadcrumb.Link class={activeLinkClass} href="/"
					><Icon src={Home} /></Breadcrumb.Link
				>
			</Breadcrumb.Item>
			<Breadcrumb.Separator class={activeLinkClass} />
			{#each breadcrumbs as { href, title }, i}
				{@const isLast = i == breadcrumbs.length - 1}
				<Breadcrumb.Item>
					{#if !isLast}
						<Breadcrumb.Link {href} class={activeLinkClass}>{title}</Breadcrumb.Link>
					{:else}
						<Breadcrumb.Link {href} class="cursor-default" aria-disabled>{title}</Breadcrumb.Link>
					{/if}
				</Breadcrumb.Item>

				{#if !isLast}
					<Breadcrumb.Separator class={activeLinkClass} />
				{/if}
			{/each}
		</Breadcrumb.List>
	</Breadcrumb.Root>
{/if}

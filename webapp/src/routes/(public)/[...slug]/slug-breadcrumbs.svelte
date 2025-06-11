<script lang="ts">
	import { page } from '$app/state';
	import Icon from '@/components/ui-custom/icon.svelte';
	import * as Breadcrumb from '@/components/ui/breadcrumb/index.js';
	import { Home } from 'lucide-svelte';
    import { String } from 'effect';

	interface Link {
		href: string;
		title: string;
	}

	const breadcrumbs = $derived.by(() => {
		const url = page.url;
		const segments = url.pathname.split('/').filter(Boolean);
		const crumbs: Link[] = [{ href: '/', title: 'Home' }];
		segments.forEach((seg, i) => {
			const href = '/' + segments.slice(0, i + 1).join('/');
			const title =  String.capitalize(decodeURIComponent(seg.replace(/-/g, ' ')));
			crumbs.push({ href, title });
		});

		return crumbs;
	});
</script>

<Breadcrumb.Root>
	<Breadcrumb.List>
		{#each breadcrumbs as { href, title }, i}
			{@const isLast = i === breadcrumbs.length - 1}

			<Breadcrumb.Item>
				{#if isLast}
					<Breadcrumb.Page>{title}</Breadcrumb.Page>
				{:else if i === 0}
					<Breadcrumb.Link {href}>
						<Icon src={Home} aria-label="Home" />
					</Breadcrumb.Link>
				{:else}
					<!-- intermediate link -->
					<Breadcrumb.Link {href} aria-disabled>{title}</Breadcrumb.Link>
				{/if}
			</Breadcrumb.Item>

			{#if !isLast}
				<Breadcrumb.Separator />
			{/if}
		{/each}
	</Breadcrumb.List>
</Breadcrumb.Root>

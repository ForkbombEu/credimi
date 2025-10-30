<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import * as Pagination from '@/components/ui/pagination';

	import { getCollectionManagerContext } from './collectionManagerContext';

	//

	interface Props {
		class?: string;
		scrollToTop?: boolean;
	}

	let { class: className = '', scrollToTop = true }: Props = $props();

	//

	const { manager } = $derived(getCollectionManagerContext());
	const perPage = $derived(manager.query.getPageSize());
	const show = $derived(perPage && manager.totalItems > perPage);

	$effect(() => {
		if (manager.currentPage && scrollToTop) {
			window.scrollTo({ top: 0, behavior: 'instant' });
		}
	});
</script>

{#if show && perPage}
	<Pagination.Root
		count={manager.totalItems}
		{perPage}
		bind:page={manager.currentPage}
		class={className}
	>
		{#snippet children({ currentPage, pages })}
			<Pagination.Content>
				<Pagination.Item>
					<Pagination.PrevButton />
				</Pagination.Item>
				{#each pages as page (page.key)}
					{#if page.type === 'ellipsis'}
						<Pagination.Item>
							<Pagination.Ellipsis />
						</Pagination.Item>
					{:else}
						<Pagination.Item>
							<Pagination.Link {page} isActive={currentPage == page.value}>
								{page.value}
							</Pagination.Link>
						</Pagination.Item>
					{/if}
				{/each}
				<Pagination.Item>
					<Pagination.NextButton />
				</Pagination.Item>
			</Pagination.Content>
		{/snippet}
	</Pagination.Root>
{/if}

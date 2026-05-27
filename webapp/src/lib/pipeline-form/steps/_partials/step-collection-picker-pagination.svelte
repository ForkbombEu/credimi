<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ChevronLeftIcon, ChevronRightIcon } from '@lucide/svelte';

	import { getCollectionManagerContext } from '@/collections-components/manager/collectionManagerContext';
	import IconButton from '@/components/ui-custom/iconButton.svelte';

	//

	const { manager } = $derived(getCollectionManagerContext());
	const canGoPrevious = $derived(manager.currentPage > 1);
	const canGoNext = $derived(manager.currentPage < manager.totalPages);

	function goPrevious() {
		if (!canGoPrevious) return;
		manager.currentPage -= 1;
	}

	function goNext() {
		if (!canGoNext) return;
		manager.currentPage += 1;
	}
</script>

{#if manager.showPagination}
	<div class="flex items-center justify-end px-4">
		<IconButton
			icon={ChevronLeftIcon}
			size="xs"
			aria-label="Previous page"
			disabled={!canGoPrevious}
			onclick={goPrevious}
			variant="ghost"
		/>
		<span class="min-w-3 text-center text-xs text-muted-foreground select-none">
			{manager.currentPage}
		</span>
		<IconButton
			icon={ChevronRightIcon}
			size="xs"
			aria-label="Next page"
			disabled={!canGoNext}
			onclick={goNext}
			variant="ghost"
		/>
	</div>
{/if}

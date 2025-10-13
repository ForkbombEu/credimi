<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { IndexItem } from '$lib/layout/pageIndex.svelte';
	import type { Snippet } from 'svelte';
	import type { ClassValue } from 'svelte/elements';

	import PageHeaderIndexed from '$lib/layout/pageHeaderIndexed.svelte';

	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';

	import EmptyState from './empty-state.svelte';

	//

	type Props = {
		indexItem: IndexItem;
		empty?: boolean;
		children?: Snippet;
		class?: ClassValue;
	};

	let { indexItem, empty = false, children, class: className = '' }: Props = $props();
</script>

<div class={['space-y-6', className]}>
	<PageHeaderIndexed {indexItem} />

	{#if empty}
		<EmptyState>
			<T>{m.No_information_available()}</T>
		</EmptyState>
	{:else}
		{@render children?.()}
	{/if}
</div>

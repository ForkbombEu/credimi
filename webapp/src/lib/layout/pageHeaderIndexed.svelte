<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ComponentProps } from 'svelte';
	import type { Simplify } from 'type-fest';

	import type { IndexItem } from './pageIndex.svelte';

	import PageHeader from './pageHeader.svelte';

	//

	type HeaderProps = Simplify<ComponentProps<typeof PageHeader>>;

	type Props = {
		indexItem: IndexItem;
	} & Omit<HeaderProps, 'id' | 'title'>;

	let { indexItem, ...rest }: Props = $props();
</script>

<PageHeader id={indexItem.anchor} title={indexItem.label} {...rest} />

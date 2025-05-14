<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { LinkWithIcon } from '../types';
	import NavigationTab, { type Props as NavigationTabProps } from './navigationTab.svelte';

	interface Props {
		tabs?: NavigationTabProps[];
	}

	let { tabs = [] }: Props = $props();
</script>

<ul class="flex gap-1 overflow-x-scroll">
	{#each tabs as tab}
		<li role="presentation">
			<NavigationTab {...tab} />
		</li>
	{/each}
</ul>

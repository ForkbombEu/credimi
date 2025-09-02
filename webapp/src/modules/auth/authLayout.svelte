<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import { AppLogo } from '@/brand';
	import { Topbar } from '@/components/layout';

	//

	interface Props {
		children?: Snippet;
		topbarRight?: Snippet;
	}

	let { children, topbarRight }: Props = $props();
</script>

<div class="fixed left-2 top-3 !m-0 hidden md:block">
	<AppLogo />
</div>

<div class="flex h-screen w-screen items-stretch">
	<div class="hidden grow basis-1 bg-primary p-2 md:block"></div>
	<div class="flex grow basis-1 flex-col overflow-scroll">
		<Topbar class="border-none">
			{#snippet left()}
				<div class="block md:hidden"><AppLogo /></div>
			{/snippet}

			{#snippet right()}
				{@render topbarRight?.()}
			{/snippet}
		</Topbar>

		<div class="flex grow basis-1 flex-col items-center justify-center">
			<div class="w-[320px] space-y-6 p-8 md:w-[400px]">
				{@render children?.()}
			</div>
		</div>
	</div>
</div>

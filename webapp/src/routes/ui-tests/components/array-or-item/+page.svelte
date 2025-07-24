<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import ArrayOrItemManager from '@/components/ui-custom/arrayOrItemManager.svelte';
	import PlainCard from '@/components/ui-custom/plainCard.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import CodeDisplay from '$lib/layout/codeDisplay.svelte';

	let items = $state(['ciao', 'no', 'ok']);

	let item: string | undefined = $state('ok');
</script>

<h1 class="scroll-m-20 text-4xl font-extrabold tracking-tight lg:text-5xl">Array test</h1>

<ArrayOrItemManager bind:data={items}>
	{#snippet children({ item, removeItem })}
		<PlainCard class="flex items-center justify-between">
			<p>{item}</p>
			<Button onclick={removeItem}>ok</Button>
		</PlainCard>
	{/snippet}
</ArrayOrItemManager>

<CodeDisplay content={JSON.stringify(items)} language="json" />

<h1 class="scroll-m-20 text-4xl font-extrabold tracking-tight lg:text-5xl">Item test</h1>

<ArrayOrItemManager bind:data={item}>
	{#snippet children({ item, removeItem })}
		<PlainCard class="flex items-center justify-between">
			<p>{item}</p>
			<Button onclick={removeItem}>ok</Button>
		</PlainCard>
	{/snippet}
</ArrayOrItemManager>

<CodeDisplay content={JSON.stringify(item)} language="json" />

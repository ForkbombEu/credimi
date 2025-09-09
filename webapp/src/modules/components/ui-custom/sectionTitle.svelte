<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ComponentProps, Snippet } from 'svelte';

	import Separator from '@/components/ui/separator/separator.svelte';

	import T from './t.svelte';

	interface Props {
		title: string;
		tag?: ComponentProps<typeof T>['tag'];
		description?: string;
		hideLine?: boolean;
		right?: Snippet;
		bottom?: Snippet;
		id?: string;
	}

	const { title, tag = 'h4', description, hideLine = false, right, bottom, id }: Props = $props();
</script>

<div class="space-y-2" {id}>
	<div class="flex flex-wrap items-center justify-between gap-2">
		<div class="w-fit">
			<T {tag}>{title}</T>
		</div>
		<div class="flex flex-wrap justify-end gap-2">
			{@render right?.()}
		</div>
	</div>

	{#if !hideLine}
		<Separator />
	{/if}

	{#if description}
		<T class="text-sm text-gray-500">{description}</T>
	{/if}

	{@render bottom?.()}
</div>

<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import Spinner from '@/components/ui-custom/spinner.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';
	import { getExceptionMessage } from '@/utils/errors';
	import type { BitsDivAttributes } from 'bits-ui';
	import { TriangleAlert } from 'lucide-svelte';
	import { generateQrCode } from './qr';

	type Props = BitsDivAttributes & {
		src?: string;
		isLoading?: boolean;
		loadingText?: string;
		placeholder?: string;
		error?: string;
	};

	let {
		src = $bindable(),
		isLoading = $bindable(false),
		error = $bindable(),
		loadingText = m.Loading_QR_code(),
		placeholder,
		class: className,
		...rest
	}: Props = $props();
</script>

<div
	{...rest}
	class={[
		'aspect-square size-60 shrink-0 overflow-hidden rounded-md border',
		'flex items-center justify-center',
		'text-muted-foreground bg-gray-50',
		'text-center text-sm',
		{
			'animate-pulse': isLoading
		},
		className
	]}
>
	<svelte:boundary
		onerror={(e) => {
			error = getExceptionMessage(e);
		}}
	>
		{#if src}
			{@const qr = generateQrCode(src)}
			<img src={qr} class="h-full w-full object-contain" alt="qr code" />
		{/if}
	</svelte:boundary>

	{#if error}
		<div class="flex flex-col items-center justify-center gap-2 p-3">
			<TriangleAlert size={20} />
			<T>{error}</T>
		</div>
	{:else if isLoading}
		<div class="flex items-center justify-center gap-2 p-3">
			<Spinner size={20} />
			<T>{loadingText}</T>
		</div>
	{:else if placeholder && !src}
		<T class="p-3">{placeholder}</T>
	{/if}
</div>

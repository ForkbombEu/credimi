<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';

	import { TriangleAlert } from 'lucide-svelte';

	import Spinner from '@/components/ui-custom/spinner.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';

	import { generateQrCode } from './qr';

	interface Props extends HTMLAttributes<HTMLDivElement> {
		src?: string;
		error?: string;
		placeholder?: string;
		isLoading?: boolean;
		loadingText?: string;
		children?: Snippet;
		hasStructuredError?: boolean;
	}
	let {
		src = $bindable(),
		isLoading = $bindable(false),
		error = $bindable(),
		children,
		hasStructuredError = false,
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
	{#if src}
		{@const qr = generateQrCode(src)}
		<img src={qr} class="h-full w-full object-contain" alt="qr code" />
	{:else if isLoading}
		<div class="flex items-center justify-center gap-2 p-3">
			<Spinner size={20} />
			<T>{loadingText}</T>
		</div>
	{:else if error}
		<div class="flex flex-col items-center justify-center gap-2 p-3">
			<TriangleAlert size={20} />
			<T>{error}</T>
		</div>
	{:else if hasStructuredError && children}
		<div class="flex flex-col items-center justify-center gap-2 p-3">
			<TriangleAlert size={20} />
			{@render children()}
		</div>
	{:else if placeholder}
		<T class="p-3">{placeholder}</T>
	{/if}
</div>

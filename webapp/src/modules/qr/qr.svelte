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
		cellSize?: number;
		error?: string;
		placeholder?: string;
		isLoading?: boolean;
		loadingText?: string;
		children?: Snippet;
		hasStructuredError?: boolean;
		showLink?: boolean;
		linkClass?: string;
	}
	let {
		src = $bindable(),
		cellSize,
		isLoading = $bindable(false),
		error = $bindable(),
		children,
		hasStructuredError = false,
		loadingText = m.Loading_QR_code(),
		placeholder,
		showLink = false,
		linkClass,
		class: className,
		...rest
	}: Props = $props();

	// Determine if we're in "stateful" mode (with loading/error states) or simple mode
	const isStateful = $derived(isLoading || error || placeholder || hasStructuredError);
</script>

<svelte:boundary>
	{#if isStateful || showLink}
		<div class="flex flex-col items-center space-y-2">
			<div
				{...rest}
				class={[
					'aspect-square shrink-0 overflow-hidden',
					isStateful ? 'text-muted-foreground size-60 rounded-md border bg-gray-50' : '',
					'flex items-center justify-center',
					'text-center text-sm',
					{
						'animate-pulse': isLoading
					},
					className
				]}
			>
				{#if src}
					{@const qr = generateQrCode(src, cellSize ?? 20)}
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
			{#if showLink && src}
				<div class={linkClass || 'w-60 break-all text-xs'}>
					<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
					<a class="text-primary hover:underline" href={src} target="_self">{src}</a>
				</div>
			{/if}
		</div>
	{:else if src}
		{@const qr = generateQrCode(src, cellSize ?? 20)}
		<img src={qr} {...rest} class={[className, 'aspect-square object-contain']} alt="qr code" />
	{/if}

	{#snippet failed()}
		<div class="flex aspect-square items-center justify-center p-4 {className}">
			<T class="text-center text-sm">{m.An_error_happened_while_generating_the_qr_code()}</T>
		</div>
	{/snippet}
</svelte:boundary>

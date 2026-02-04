<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { HTMLAttributes } from 'svelte/elements';

	import { TriangleAlert } from '@lucide/svelte';

	import CopyButtonSmall from '@/components/ui-custom/copy-button-small.svelte';
	import Spinner from '@/components/ui-custom/spinner.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';

	import { generateQrCode } from './qr';

	//

	interface Props extends HTMLAttributes<HTMLDivElement> {
		src?: string;
		error?: string;
		placeholder?: string;
		isLoading?: boolean;
		loadingText?: string;
		showLink?: boolean;
		alt?: string;
	}

	let {
		src,
		isLoading,
		error,
		loadingText = m.Loading_QR_code(),
		placeholder = m.QR_code(),
		showLink = false,
		class: className = '',
		alt = m.QR_code(),
		children,
		...rest
	}: Props = $props();

	//

	const qrDataUrl = $derived.by(() => {
		if (!src) return null;
		try {
			return generateQrCode(src, 20);
		} catch (e) {
			error = m.An_error_happened_while_generating_the_qr_code();
			console.error('Failed to generate QR code:', e);
			return null;
		}
	});
</script>

<!-- QR Code + Link Wrapper -->
<div class="flex flex-col items-start space-y-2">
	<!-- Actual QR Code -->
	<div
		{...rest}
		class={[
			'aspect-square shrink-0 overflow-hidden rounded-md border bg-slate-50',
			'flex flex-col items-center justify-center gap-1',
			'text-center text-sm text-muted-foreground',
			isLoading && 'animate-pulse',
			!qrDataUrl && 'p-3',
			!className?.includes('size-') && 'size-60',
			className
		]}
	>
		{#if qrDataUrl}
			<img src={qrDataUrl} class="aspect-square h-full w-full object-contain" {alt} />
		{:else if isLoading}
			<Spinner size={20} />
			<T>{loadingText}</T>
		{:else if error}
			<TriangleAlert size={20} />
			<T>{error}</T>
		{:else if children}
			{@render children?.()}
		{:else if placeholder}
			<T>{placeholder}</T>
		{/if}
	</div>

	<!-- Link -->
	{#if showLink && src}
		<div class="w-full space-y-1">
			<div class="flex">
				<div class="w-0 grow">
					<!-- eslint-disable svelte/no-navigation-without-resolve -->
					<a
						class="block text-xs leading-relaxed break-all text-primary hover:underline"
						href={src}
					>
						{src}
					</a>
				</div>
			</div>
			<CopyButtonSmall size="mini" class="-translate-x-1 px-1.5!" textToCopy={src}>
				{m.Copy()}
			</CopyButtonSmall>
		</div>
	{/if}
</div>

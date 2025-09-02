<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';
	import type { HTMLImgAttributes } from 'svelte/elements';
	import { generateQrCode } from './qr';

	type Props = HTMLImgAttributes & {
		src: string;
		cellSize?: number;
	};

	let { src, cellSize, ...rest }: Props = $props();
</script>

<svelte:boundary>
	{@const qr = generateQrCode(src, cellSize ?? 20)}
	<img src={qr} {...rest} class={[rest.class, 'aspect-square object-contain']} />

	{#snippet failed()}
		<div class="flex aspect-square items-center justify-center p-4 {rest.class}">
			<T class="text-center text-sm">{m.An_error_happened_while_generating_the_qr_code()}</T>
		</div>
	{/snippet}
</svelte:boundary>

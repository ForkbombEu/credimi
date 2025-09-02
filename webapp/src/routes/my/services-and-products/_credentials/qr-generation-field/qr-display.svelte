<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { QrCode } from '@/qr';

	//

	interface Props {
		src: string;
		showUrl?: boolean;
		cellSize?: number;
		class?: string;
	}

	const {
		src,
		showUrl = true,
		cellSize = 10,
		class: className = 'size-60 rounded-md border'
	}: Props = $props();
</script>

<div class="flex flex-col items-stretch gap-4 pt-3 md:flex-row">
	<QrCode {src} {cellSize} class={className} />
	{#if showUrl}
		<div class="max-w-60 break-all text-xs">
			<a href={src} target="_self">{src}</a>
		</div>
	{/if}
</div>

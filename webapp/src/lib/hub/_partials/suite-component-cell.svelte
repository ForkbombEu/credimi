<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { QrCode } from '@lucide/svelte';

	import type { EntityData } from '$lib/global/entities';
	import EntityTag from '$lib/global/entity-tag.svelte';
	import Tooltip from '@/components/ui-custom/tooltip.svelte';
	import { m } from '@/i18n';

	//

	type Props = {
		data: EntityData;
		hasQr?: boolean;
	};

	let { data, hasQr = false }: Props = $props();
</script>

<div class="flex items-center gap-1.5">
	<EntityTag {data} />
	{#if hasQr}
		<Tooltip>
			<span aria-label={m.suite_has_qr_tooltip()}>
				<QrCode class="h-4 w-4 text-muted-foreground" aria-hidden="true" />
			</span>
			{#snippet content()}
				<p>{m.suite_has_qr_tooltip()}</p>
			{/snippet}
		</Tooltip>
	{/if}
</div>

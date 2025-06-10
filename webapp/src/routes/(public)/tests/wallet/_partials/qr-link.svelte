<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->
<script lang="ts">
	import { QrCode } from '@/qr';
	import Alert from '@/components/ui-custom/alert.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';

	type Props = { qr?: string };
	let { qr }: Props = $props();
</script>

{#if qr}
	<QrCode src={qr} class="size-40 rounded-sm" />
	<p class="text-primary max-w-sm break-all p-4 font-mono text-xs hover:underline">
		{qr}
	</p>
{:else}
	<Alert variant="destructive">
		<T class="font-bold">{m.Error_check_failed()}</T>
		<T>
			{m.An_error_happened_during_the_check_please_read_the_logs_for_more_information()}
		</T>
	</Alert>
{/if}

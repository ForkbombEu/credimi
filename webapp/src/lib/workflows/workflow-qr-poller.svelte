<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { onMount } from 'svelte';
	import { z } from 'zod';

	import Spinner from '@/components/ui-custom/spinner.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';
	import { QrCode } from '@/qr';
	import { warn } from '@/utils/other';

	//

	type Props = {
		workflowId: string;
		runId: string;
		containerClass?: string;
		showQrLink?: boolean;
	};

	let { workflowId, runId, containerClass, showQrLink }: Props = $props();

	let deeplink = $state<string>();
	let attempt = $state(0);
	const maxAttempts = 5;

	onMount(() => {
		const interval = setInterval(async () => {
			attempt++;
			try {
				const res = await pb.send(`/api/compliance/deeplink/${workflowId}/${runId}`, {
					method: 'GET',
					fetch
				});
				const data = z.object({ deeplink: z.string() }).parse(res);
				deeplink = data.deeplink;

				clearInterval(interval);
			} catch (error) {
				warn(error);
			}
		}, 2000);

		return () => clearInterval(interval);
	});
</script>

<div class="flex flex-col items-center space-y-2">
	<div
		class={[
			'flex flex-col items-center justify-center rounded-sm border bg-gray-50',
			containerClass
		]}
	>
		{#if deeplink}
			<QrCode src={deeplink} class="h-full w-full" />
		{:else if attempt < maxAttempts}
			<Spinner size={20} />
			<T class="px-3 pt-2 text-center text-xs text-gray-400">
				{m.Loading_QR_code()}
			</T>
		{:else}
			<T class="px-3 text-center text-xs text-gray-400">
				{m.The_QR_code_may_be_not_available_for_this_test()}
			</T>
		{/if}
	</div>
	{#if showQrLink && deeplink}
		<div class="max-w-sm break-all text-center text-xs">
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
			<a class="text-primary hover:underline" href={deeplink} target="_self"> {deeplink}</a>
		</div>
	{/if}
</div>

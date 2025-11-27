<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { onMount } from 'svelte';
	import { z } from 'zod';

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
	let isLoading = $state(true);
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
				isLoading = false;

				clearInterval(interval);
			} catch (error) {
				warn(error);
				if (attempt >= maxAttempts) {
					isLoading = false;
				}
			}
		}, 2000);

		return () => clearInterval(interval);
	});
</script>

<QrCode
	src={deeplink}
	{isLoading}
	placeholder={m.The_QR_code_may_be_not_available_for_this_test()}
	class={containerClass}
	showLink={showQrLink}
/>

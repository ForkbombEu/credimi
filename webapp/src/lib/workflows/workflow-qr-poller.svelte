<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { QrCode } from '@/qr';
	import type { WorkflowExecution } from './types';
	import { pb } from '@/pocketbase';
	import { onMount } from 'svelte';
	import T from '@/components/ui-custom/t.svelte';
	import Spinner from '@/components/ui-custom/spinner.svelte';
	import { z } from 'zod';
	import { warn } from '@/utils/other';
	import { m } from '@/i18n';

	//

	type Props = {
		workflowId: string;
		runId: string;
		containerClass?: string;
	};

	let { workflowId, runId, containerClass }: Props = $props();

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

<div
	class={[
		'flex aspect-square flex-col items-center justify-center rounded-sm border bg-gray-50',
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

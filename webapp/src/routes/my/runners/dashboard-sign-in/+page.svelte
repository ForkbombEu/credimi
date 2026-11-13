<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { CircleAlertIcon } from '@lucide/svelte';
	import { getRecordByCanonifiedPath } from '$lib/canonify/index.js';
	import { requestDashboardCode } from '$lib/runner-dashboard/sign-in';
	import { ClientResponseError } from 'pocketbase';

	import Alert from '@/components/ui-custom/alert.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import Spinner from '@/components/ui-custom/spinner.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';
	import { currentUser } from '@/pocketbase';

	//

	let { data } = $props();
	const request = $derived(data.request);

	let runnerName = $state<string>();
	let failure = $state('');
	let signingIn = $state(false);

	$effect(() => {
		const runner = request?.runner;
		runnerName = undefined;
		if (!runner) return;
		getRecordByCanonifiedPath<{ name?: string }>(runner).then((record) => {
			if (!(record instanceof Error) && request?.runner === runner) runnerName = record.name;
		});
	});

	async function signIn() {
		if (!request) return;
		signingIn = true;
		failure = '';
		try {
			window.location.assign(await requestDashboardCode(request));
		} catch (e) {
			failure =
				e instanceof ClientResponseError
					? (e.response as { message?: string })?.message || e.message
					: String(e);
			signingIn = false;
		}
	}
</script>

<svelte:head>
	<title>{m.Runner_dashboard_sign_in()} | Credimi</title>
</svelte:head>

<div class="space-y-2">
	<T tag="h4">{m.Runner_dashboard_sign_in()}</T>
	{#if request}
		<T class="text-muted-foreground">{m.Runner_dashboard_sign_in_description()}</T>
	{/if}
</div>

{#if !request}
	<Alert variant="destructive" icon={CircleAlertIcon}>
		<T>{m.Runner_dashboard_sign_in_invalid()}</T>
	</Alert>
	<Button variant="outline" href="/my" class="w-full">{m.Cancel()}</Button>
{:else}
	<dl class="space-y-3 rounded-[10px] border bg-secondary/40 p-4 text-sm">
		<div>
			<dt class="text-xs font-medium tracking-[0.14em] text-muted-foreground uppercase">
				{m.Runner()}
			</dt>
			<dd class="mt-1">
				{#if runnerName}<span class="font-medium">{runnerName}</span><br />{/if}
				<span class="font-mono text-xs break-all">{request.runner}</span>
			</dd>
		</div>
		<div>
			<dt class="text-xs font-medium tracking-[0.14em] text-muted-foreground uppercase">
				{m.Dashboard()}
			</dt>
			<dd class="mt-1 font-mono break-all">{data.host}</dd>
		</div>
		<div>
			<dt class="text-xs font-medium tracking-[0.14em] text-muted-foreground uppercase">
				{m.Signed_in_as()}
			</dt>
			<dd class="mt-1 break-all">{$currentUser?.email}</dd>
		</div>
	</dl>

	{#if failure}
		<Alert variant="destructive" icon={CircleAlertIcon}>
			<T>{failure}</T>
		</Alert>
	{/if}

	<div class="space-y-2">
		<Button onclick={signIn} disabled={signingIn} class="w-full">
			{#if signingIn}<Spinner size={16} />{/if}
			{m.Sign_in()}
		</Button>
		<Button variant="outline" href="/my" class="w-full">{m.Cancel()}</Button>
	</div>
{/if}

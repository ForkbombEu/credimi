<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { HistoryIcon } from '@lucide/svelte';
	import { formatExecutionTimestamp } from '$lib/scoreboard/extras/format-date';
	import WorkflowStatusBadge from '$lib/workflows/workflow-status.svelte';
	import { fromStore } from 'svelte/store';

	import type { OrganizationsResponse, WalletsResponse } from '@/pocketbase/types';

	import Button from '@/components/ui-custom/button.svelte';
	import Sheet from '@/components/ui-custom/sheet.svelte';
	import Spinner from '@/components/ui-custom/spinner.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';
	import { currentUser } from '@/pocketbase';

	import { listSessions, type SessionListItem } from './api';
	import { walletPath } from './builds';

	//

	type Props = {
		wallet: WalletsResponse;
		organization: OrganizationsResponse;
	};

	let { wallet, organization }: Props = $props();

	const user = fromStore(currentUser);
	const timezone = $derived(user.current?.Timezone);

	let open = $state(false);
	let sessions = $state<Promise<SessionListItem[]>>();

	$effect(() => {
		if (!open) return;
		sessions = listSessions(walletPath(organization, wallet)).then((res) => {
			if (res.isErr) throw new Error(res.error.message);
			return res.value;
		});
	});
</script>

<Sheet bind:open title={m.Authoring_sessions()} class="!w-[min(100vw,32rem)]">
	{#snippet trigger({ sheetTriggerAttributes })}
		<Button variant="outline" size="sm" {...sheetTriggerAttributes}>
			<HistoryIcon />
			{m.Authoring_sessions()}
		</Button>
	{/snippet}

	{#snippet content()}
		{#await sessions}
			<div class="flex items-center justify-center gap-2 py-6">
				<Spinner size={16} />
				<T class="text-muted-foreground">{m.Loading()}</T>
			</div>
		{:then list}
			{#if list}
				<ul class="space-y-2">
					{#each list as session (session.session_id)}
						<li
							class="flex items-center justify-between gap-3 rounded-md border p-3 text-sm"
						>
							<div class="min-w-0 space-y-1">
								<WorkflowStatusBadge status={session.status} />
								<p>
									{formatExecutionTimestamp(session.start_time, timezone) ?? '-'}
								</p>
								{#if session.device_name}
									<p class="truncate text-muted-foreground">
										{session.device_name}
									</p>
								{/if}
							</div>
							<Button
								variant="outline"
								size="sm"
								href={`/my/action-author/${session.session_id}?device=${encodeURIComponent(session.device_id)}`}
							>
								{m.Open_for_details()}
							</Button>
						</li>
					{:else}
						<T class="text-sm text-muted-foreground">{m.No_authoring_sessions()}</T>
					{/each}
				</ul>
			{/if}
		{:catch error}
			<p class="text-sm text-destructive" role="alert">{error.message}</p>
		{/await}
	{/snippet}
</Sheet>

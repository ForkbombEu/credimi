<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { MonitorPlayIcon, SquareIcon, XIcon } from '@lucide/svelte';
	import { page } from '$app/state';
	import { Pipeline } from '$lib';
	import { closeSession, getSession, type Session } from '$lib/action-author/api';
	import SaveReviewCard from '$lib/action-author/save-review-card.svelte';
	import BackButton from '$lib/layout/back-button.svelte';
	import { runWithLoading } from '$lib/layout/global-loading.svelte';
	import LiveViewSheet from '$lib/pipeline/live-view-sheet.svelte';
	import { formatExecutionTimestamp } from '$lib/scoreboard/extras/format-date';
	import WorkflowStatusBadge from '$lib/workflows/workflow-status.svelte';
	import { untrack } from 'svelte';
	import { SvelteSet } from 'svelte/reactivity';
	import { fromStore } from 'svelte/store';

	import Alert from '@/components/ui-custom/alert.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import Spinner from '@/components/ui-custom/spinner.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';
	import { currentUser } from '@/pocketbase';

	//

	let { data } = $props();
	const sessionId = $derived(data.sessionId);
	const deviceId = $derived(data.deviceId);

	const user = fromStore(currentUser);
	const timezone = $derived(user.current?.Timezone);

	/** Before the session is recorded, its ticket waits in the device queue. */
	type QueueState = { kind: 'queued'; position: number; total: number } | { kind: 'preparing' };

	let session = $state<Session>();
	let queue = $state<QueueState>();
	let queueCancellable = $state(false);
	let stoppedMessage = $state('');
	let pollError = $state('');
	let pollNow = $state<() => void>(() => {});

	const settledReviews = new SvelteSet<string>();

	/* Polling */

	const POLL_INTERVAL_MS = 2000;
	const TICKET_PREFIX = 'ActionAuthorSession-';

	$effect(() => {
		const polled = { sessionId, deviceId };
		return untrack(() => startPolling(polled.sessionId, polled.deviceId));
	});

	// Sequential: a slow request must not overlap the next one.
	function startPolling(polledSessionId: string, polledDeviceId: string) {
		let active = true;
		let inFlight = false;
		let again = false;
		let timer: ReturnType<typeof setTimeout> | undefined;

		session = undefined;
		queue = undefined;
		queueCancellable = false;
		stoppedMessage = '';
		pollError = '';
		settledReviews.clear();

		/** Reads the session or its ticket; returns whether to poll again. */
		async function tick(): Promise<boolean> {
			const res = await getSession(polledSessionId);
			if (!active) return false;
			if (res.isOk) {
				session = res.value;
				queue = undefined;
				queueCancellable = false;
				pollError = '';
				return res.value.status === 'Running';
			}

			const failure = res.error;
			const notStarted =
				failure.status === 404 &&
				(failure.reason === 'session_not_started' ||
					failure.reason === 'session_not_found');
			if (!notStarted) {
				pollError = failure.message;
				return true;
			}
			if (!polledDeviceId) {
				stoppedMessage = m.Session_not_found();
				return false;
			}

			const ticket = polledSessionId.slice(TICKET_PREFIX.length);
			const status = await Pipeline.Queue.status(ticket, [polledDeviceId]);
			if (!active) return false;
			if (status.isErr) {
				pollError = status.error;
				return true;
			}
			pollError = '';
			const q = status.value;
			switch (q.status) {
				case 'queued':
					queue = {
						kind: 'queued',
						position: (q.position ?? 0) + 1,
						total: q.line_len ?? 0
					};
					queueCancellable = true;
					return true;
				case 'starting':
				case 'running':
					queue = { kind: 'preparing' };
					queueCancellable = q.status === 'starting';
					return true;
				case 'failed':
				case 'canceled':
					queue = undefined;
					stoppedMessage = q.error_message || m.Canceled();
					return false;
				case 'not_found':
					queue = undefined;
					stoppedMessage = m.Session_not_found();
					return false;
			}
		}

		async function poll() {
			clearTimeout(timer);
			timer = undefined;
			if (inFlight) {
				again = true;
				return;
			}
			inFlight = true;
			let next = false;
			try {
				next = await tick();
			} finally {
				inFlight = false;
			}
			if (!active) return;
			if (again) {
				again = false;
				void poll();
			} else if (next) {
				timer = setTimeout(poll, POLL_INTERVAL_MS);
			}
		}

		pollNow = () => void poll();
		void poll();

		return () => {
			active = false;
			clearTimeout(timer);
		};
	}

	async function cancelTicket() {
		const ticket = sessionId.slice(TICKET_PREFIX.length);
		await runWithLoading({
			fn: async () => {
				const res = await Pipeline.Queue.cancel(ticket, [deviceId]);
				if (res.isErr) throw new Error(res.error);
			}
		});
		pollNow();
	}

	async function endSession() {
		const id = sessionId;
		await runWithLoading({
			fn: async () => {
				const res = await closeSession(id);
				if (res.isErr) throw new Error(res.error.message);
			}
		});
		pollNow();
	}

	/* Idle warning */

	let now = $state(Date.now());

	$effect(() => {
		const interval = setInterval(() => (now = Date.now()), 1000);
		return () => clearInterval(interval);
	});

	const idleSeconds = $derived.by(() => {
		if (!session?.closes_at || session.status !== 'Running') return undefined;
		const seconds = Math.floor((Date.parse(session.closes_at) - now) / 1000);
		return seconds >= 0 && seconds < 60 ? seconds : undefined;
	});

	/* Embedded harness chat */

	const running = $derived(session?.status === 'Running');

	// The harness UI behind Credimi's session-scoped proxy (pkg/internal/actionauthorui), which
	// authenticates the `pb_auth` cookie and scopes every call to this session's chat.
	const chatUrl = $derived.by(() => {
		if (!session) return undefined;
		const temporalUi = [
			page.url.origin,
			'temporal-ui/namespaces',
			encodeURIComponent(session.namespace),
			'workflows'
		].join('/');
		return (
			`/action-author-ui/${session.session_id}/` +
			`?s=${encodeURIComponent(session.chat_workflow_id)}` +
			`&temporal-ui=${encodeURIComponent(temporalUi)}`
		);
	});
	let loadedChatUrl = $state<string>();
	const isChatLoading = $derived(loadedChatUrl !== chatUrl);

	const pendingReview = $derived(
		session?.pending_review && !settledReviews.has(session.pending_review.tool_id)
			? session.pending_review
			: undefined
	);

	let liveViewOpen = $state(false);
</script>

<svelte:head>
	<style>
		body {
			background-color: rgb(248 250 252);
		}
	</style>
</svelte:head>

<div class="bg-primary">
	<div class="padding-x">
		<BackButton href="/my/wallets" class="text-white" />
	</div>
</div>

<div
	class="bg-temporal padding-x flex flex-wrap items-start justify-between gap-4 py-4 sm:flex-nowrap sm:gap-8"
>
	<div class="min-w-0 space-y-3">
		<T tag="h1">{m.Authoring_session()}</T>

		{#if session}
			<p class="font-mono text-xs break-words text-muted-foreground">
				{[
					session.wallet_version_identifier,
					session.device_name,
					session.runner_name,
					session.runner_owner ? m.Runner_of({ owner: session.runner_owner }) : '',
					session.model
				]
					.filter(Boolean)
					.join(' · ')}
			</p>

			<WorkflowStatusBadge status={session.status} />

			<table class="text-sm">
				<tbody>
					<tr>
						<td class="italic">{m.Start_time()}</td>
						<td class="pl-4"
							>{formatExecutionTimestamp(session.start_time, timezone) ?? '-'}</td
						>
					</tr>
					<tr>
						<td class="italic">{m.End_time()}</td>
						<td class="pl-4"
							>{formatExecutionTimestamp(session.close_time, timezone) ?? '-'}</td
						>
					</tr>
				</tbody>
			</table>
		{/if}

		{#if pollError}
			<Alert variant="destructive" class="block border border-destructive p-3!">
				{pollError}
			</Alert>
		{/if}

		{#if idleSeconds !== undefined}
			<Alert variant="warning" class="block p-3!" role="status">
				{m.Session_idle_warning({ seconds: idleSeconds })}
			</Alert>
		{/if}
	</div>

	{#if session}
		<div class="flex flex-wrap gap-2">
			{#if session.live_view}
				<Button variant="outline" onclick={() => (liveViewOpen = true)}>
					<MonitorPlayIcon />
					{m.View_live()}
				</Button>
			{/if}
			<Button variant="outline" onclick={endSession} disabled={!running}>
				<SquareIcon />
				{m.End_session()}
			</Button>
		</div>
	{/if}
</div>

<div class="bg-temporal padding-x pb-6">
	{#if stoppedMessage}
		<Alert variant="destructive" class="block border border-destructive p-3!">
			{stoppedMessage}
		</Alert>
	{:else if queue}
		<div class="flex flex-wrap items-center gap-4 rounded-lg border bg-slate-200 px-4 py-6">
			<Spinner size={16} />
			<T class="text-muted-foreground">
				{queue.kind === 'queued'
					? m.Session_waiting_for_device({ position: queue.position, total: queue.total })
					: m.Session_preparing_device()}
			</T>
			{#if queueCancellable}
				<Button variant="outline" size="sm" onclick={cancelTicket}>
					<XIcon />
					{m.Cancel()}
				</Button>
			{/if}
		</div>
	{:else if !session}
		<div class="flex items-center gap-2 rounded-lg border bg-slate-200 px-4 py-6">
			<Spinner size={16} />
			<T class="text-muted-foreground">{m.Loading()}</T>
		</div>
	{:else if running && !session.chat_ready}
		<div class="flex items-center gap-2 rounded-lg border bg-slate-200 px-4 py-6">
			<Spinner size={16} />
			<T class="text-muted-foreground">{m.Session_preparing_device()}</T>
		</div>
	{:else if chatUrl}
		<div class="relative" aria-busy={isChatLoading}>
			{#if isChatLoading}
				<div class="absolute inset-0 z-10">
					<div
						class={[
							'rounded-lg border bg-slate-200 py-10 text-center',
							'flex items-center justify-center gap-2',
							'animate-pulse'
						]}
					>
						<Spinner size={16} />
						<T class="text-muted-foreground">{m.Loading()}</T>
					</div>
				</div>
			{/if}

			<!-- Reloaded when the session ends, so the harness replays the closed chat. -->
			{#key running}
				<iframe
					title={m.Authoring_session()}
					src={chatUrl}
					class="block h-[calc(100dvh-16rem)] min-h-[600px] w-full border-0"
					onload={() => (loadedChatUrl = chatUrl)}
				></iframe>
			{/key}
		</div>
	{/if}
</div>

{#if session && pendingReview}
	<div
		class="fixed right-6 bottom-6 z-50 max-h-[calc(100dvh-3rem)] w-[min(28rem,calc(100vw-3rem))] overflow-auto"
	>
		{#key pendingReview.tool_id}
			<SaveReviewCard
				{session}
				review={pendingReview}
				onSettled={() => {
					settledReviews.add(pendingReview.tool_id);
					pollNow();
				}}
			/>
		{/key}
	</div>
{/if}

{#if session}
	<LiveViewSheet
		bind:open={liveViewOpen}
		workflowId={session.session_id}
		runId={session.run_id}
		title={m.Authoring_session()}
		deviceId={session.device_id}
		deviceName={session.device_name}
	/>
{/if}

<style lang="postcss">
	@reference 'tailwindcss';

	.bg-temporal {
		background-color: rgb(248 250 252);
	}

	.padding-x {
		@apply px-4! md:px-8!;
	}
</style>

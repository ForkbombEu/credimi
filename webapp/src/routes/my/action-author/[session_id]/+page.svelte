<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { MonitorPlayIcon, SquareIcon, XIcon } from '@lucide/svelte';
	import { beforeNavigate } from '$app/navigation';
	import { page } from '$app/state';
	import { Pipeline } from '$lib';
	import {
		closeSession,
		endOnExit,
		getSession,
		TICKET_PREFIX,
		type Session
	} from '$lib/action-author/api';
	import SaveReviewCard from '$lib/action-author/save-review-card.svelte';
	import BackButton from '$lib/layout/back-button.svelte';
	import { runWithLoading } from '$lib/layout/global-loading.svelte';
	import LiveViewFrame from '$lib/pipeline/live-view-frame.svelte';
	import { formatExecutionTimestamp } from '$lib/scoreboard/extras/format-date';
	import { confirm } from '$lib/utils';
	import WorkflowStatusBadge from '$lib/workflows/workflow-status.svelte';
	import { untrack } from 'svelte';
	import { SvelteSet } from 'svelte/reactivity';
	import { fromStore } from 'svelte/store';

	import Alert from '@/components/ui-custom/alert.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import Spinner from '@/components/ui-custom/spinner.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { goto, m } from '@/i18n';
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

			// Unrecorded yet: the ticket may still wait in the device queue.
			const failure = res.error;
			const notStarted = failure.status === 404 && failure.reason === 'session_not_found';
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

	/** Cancels the queued ticket; returns whether it succeeded. */
	async function cancelTicket(): Promise<boolean> {
		const ticket = sessionId.slice(TICKET_PREFIX.length);
		const done = await runWithLoading({
			fn: async () => {
				const res = await Pipeline.Queue.cancel(ticket, [deviceId]);
				if (res.isErr) throw new Error(res.error);
				return true;
			}
		});
		pollNow();
		return done === true;
	}

	/** Ends the session; returns whether it succeeded. */
	async function endSession(): Promise<boolean> {
		const id = sessionId;
		const done = await runWithLoading({
			fn: async () => {
				const res = await closeSession(id);
				if (res.isErr) throw new Error(res.error.message);
				return true;
			}
		});
		pollNow();
		return done === true;
	}

	/* Leaving the page ends the session */

	// The queued ticket or the running chat holds a device until it ends; once the chat has
	// ended the session only releases the device.
	const guardActive = $derived(
		(!!queue && !stoppedMessage) ||
			(session?.status === 'Running' && (!session.chat_ready || session.chat_running))
	);
	let exitConfirmed = false;

	/** Ends the session, or the queued ticket before the session starts; returns whether it succeeded. */
	function endBeforeLeaving(): Promise<boolean> {
		return session ? endSession() : cancelTicket();
	}

	beforeNavigate(async (nav) => {
		if (!guardActive || exitConfirmed || !nav.to || nav.type === 'leave') return;
		nav.cancel();
		const target = nav.to.url;
		const ok = await confirm({
			title: m.Leave_session_title(),
			message: m.Leave_session_warning(),
			confirmLabel: m.Leave_and_end_session(),
			cancelLabel: m.Stay_on_page(),
			destructive: true
		});
		if (!ok) return;
		if (!(await endBeforeLeaving())) return;
		exitConfirmed = true;
		await goto(`${target.pathname}${target.search}${target.hash}`);
	});

	// Tab close and reload: the browser shows its own prompt (custom text isn't allowed there),
	// and the session ends as the page goes away.
	$effect(() => {
		const onBeforeUnload = (e: BeforeUnloadEvent) => {
			if (!guardActive || exitConfirmed) return;
			e.preventDefault();
			e.returnValue = '';
		};
		const onPageHide = () => {
			if (!guardActive || exitConfirmed) return;
			endOnExit(sessionId, deviceId, !session);
		};
		window.addEventListener('beforeunload', onBeforeUnload);
		window.addEventListener('pagehide', onPageHide);
		return () => {
			window.removeEventListener('beforeunload', onBeforeUnload);
			window.removeEventListener('pagehide', onPageHide);
		};
	});

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

	const metadata = $derived.by(() => {
		if (!session) return '';
		return [
			session.wallet_version_identifier,
			session.device_name,
			session.runner_name,
			session.runner_owner ? m.Runner_of({ owner: session.runner_owner }) : '',
			session.model,
			formatExecutionTimestamp(session.start_time, timezone),
			running ? undefined : formatExecutionTimestamp(session.close_time, timezone)
		]
			.filter(Boolean)
			.join(' · ');
	});

	const showReview = $derived(!!pendingReview);
	const showLiveView = $derived(liveViewOpen && !!session?.live_view);
</script>

<div class="flex h-dvh flex-col bg-slate-50">
	<header class="shrink-0 border-b bg-white">
		<div class="padding-x flex flex-wrap items-center gap-x-6 gap-y-2 py-3">
			<BackButton href="/my/wallets" />

			<div class="min-w-0 flex-1">
				<div class="flex flex-wrap items-center gap-x-3 gap-y-1">
					<T tag="h1">{m.Authoring_session()}</T>
					{#if session}
						<WorkflowStatusBadge status={session.status} />
					{/if}
				</div>
				{#if metadata}
					<p class="truncate font-mono text-xs text-muted-foreground" title={metadata}>
						{metadata}
					</p>
				{/if}
			</div>

			<div class="flex flex-wrap gap-2">
				{#if session?.live_view}
					<Button
						variant={liveViewOpen ? 'secondary' : 'outline'}
						onclick={() => (liveViewOpen = !liveViewOpen)}
					>
						<MonitorPlayIcon />
						{m.View_live()}
					</Button>
				{/if}
				{#if running}
					<Button variant="outline" onclick={endSession}>
						<SquareIcon />
						{m.End_session()}
					</Button>
				{/if}
			</div>
		</div>

		{#if pollError || idleSeconds !== undefined}
			<div class="padding-x space-y-2 pb-3">
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
		{/if}
	</header>

	<main class="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto p-4 lg:flex-row">
		<section
			class="relative flex min-h-[60dvh] min-w-0 flex-1 flex-col overflow-hidden rounded-[10px] border bg-white lg:min-h-0"
		>
			{#if stoppedMessage}
				<div class="p-4">
					<Alert variant="destructive" class="block border border-destructive p-3!">
						{stoppedMessage}
					</Alert>
				</div>
			{:else if queue}
				<div class="flex flex-1 flex-wrap items-center justify-center gap-4 p-6">
					<Spinner size={16} />
					<T class="text-muted-foreground">
						{queue.kind === 'queued'
							? m.Session_waiting_for_device({
									position: queue.position,
									total: queue.total
								})
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
				<div class="flex flex-1 items-center justify-center gap-2 p-6">
					<Spinner size={16} />
					<T class="text-muted-foreground">{m.Loading()}</T>
				</div>
			{:else if !session.chat_ready}
				<div class="flex flex-1 items-center justify-center gap-2 p-6">
					{#if running}
						<Spinner size={16} />
						<T class="text-muted-foreground">{m.Session_preparing_device()}</T>
					{:else}
						<T class="text-muted-foreground">{m.Session_ended_without_chat()}</T>
					{/if}
				</div>
			{:else if chatUrl}
				{#if running && !session.chat_running}
					<div
						class="flex shrink-0 items-center gap-2 border-b bg-slate-50 px-4 py-2 text-sm text-muted-foreground"
						role="status"
					>
						<Spinner size={14} />
						{m.Session_releasing_device()}
					</div>
				{/if}

				<div class="relative min-h-0 flex-1" aria-busy={isChatLoading}>
					{#if isChatLoading}
						<div
							class="absolute inset-0 z-10 flex animate-pulse items-center justify-center gap-2 bg-white"
						>
							<Spinner size={16} />
							<T class="text-muted-foreground">{m.Loading()}</T>
						</div>
					{/if}

					<!-- Reloaded when the chat ends, so the harness replays the closed chat. -->
					{#key session.chat_running}
						<iframe
							title={m.Authoring_session()}
							src={chatUrl}
							class="absolute inset-0 h-full w-full border-0"
							onload={() => (loadedChatUrl = chatUrl)}
						></iframe>
					{/key}
				</div>
			{/if}
		</section>

		{#if session && (showReview || showLiveView)}
			<aside class="flex w-full shrink-0 flex-col gap-4 overflow-y-auto lg:w-[22rem]">
				{#if pendingReview && showReview}
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
				{/if}

				{#if showLiveView}
					<section class="flex flex-col gap-2 rounded-[10px] border bg-white p-3">
						<div class="flex items-center justify-between gap-2">
							<T class="font-semibold">{m.Live_view()}: {session.device_name}</T>
							<Button
								variant="ghost"
								size="icon-sm"
								aria-label={m.Close()}
								onclick={() => (liveViewOpen = false)}
							>
								<XIcon />
							</Button>
						</div>
						<LiveViewFrame
							workflowId={session.session_id}
							runId={session.run_id}
							deviceId={session.device_id}
							deviceName={session.device_name}
							hideCommands
							class="h-[min(70dvh,44rem)]"
						/>
					</section>
				{/if}
			</aside>
		{/if}
	</main>
</div>

<style lang="postcss">
	@reference 'tailwindcss';

	.padding-x {
		@apply px-4! md:px-8!;
	}
</style>

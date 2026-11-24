<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { untrack } from 'svelte';

	import Alert from '@/components/ui-custom/alert.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import { Label } from '@/components/ui/label';
	import * as RadioGroup from '@/components/ui/radio-group/index.js';
	import * as Tabs from '@/components/ui/tabs/index.js';
	import { Textarea } from '@/components/ui/textarea';
	import { m } from '@/i18n';

	import {
		sendReview,
		type Decision,
		type PendingReview,
		type ReviewAction,
		type Session
	} from './api';
	import {
		blockedBy,
		canAccept,
		canSend,
		hasPlan,
		reviewDecisions,
		startingName,
		type Choices
	} from './review';

	//

	type Props = {
		session: Session;
		review: PendingReview;
		onSettled: () => void;
	};

	let { session, review, onSettled }: Props = $props();

	// The card is keyed on tool_id, and a review's actions never change: each poll brings a fresh
	// but equal review object, which must not reset the choices.
	const actions = untrack(() => review.actions);
	const toolId = untrack(() => review.tool_id);
	// The FCAF profile the agent saves with every action of the review; an accept binds it.
	const profile = untrack(() => review.profile);
	// The agent's FCAF findings about the wallet itself (such as a missing key attestation).
	const warnings = untrack(() => review.warnings);

	let choices = $state<Choices>(
		Object.fromEntries(actions.map((action) => [action.key, { decision: null, comment: '' }]))
	);
	// The YAML shown must match candidate_sha256, the digest Credimi's approval signs.
	let acceptable = $state<Record<string, boolean>>({});
	let selected = $state(actions[0]?.key ?? '');
	let status = $state('');
	let submitting = $state(false);

	$effect(() => {
		for (const action of actions) {
			void canAccept(action).then((ok) => {
				acceptable[action.key] = ok;
			});
		}
	});

	const names = Object.fromEntries(actions.map((action) => [action.key, action.name]));

	function nameOf(action: ReviewAction, key: string): string {
		return startingName(action, key, names);
	}

	function accepts(action: ReviewAction): boolean {
		return !!acceptable[action.key] && blockedBy(action, choices) === undefined;
	}

	/** Drops every accept whose starting action is no longer accepted, down the chain. */
	function dropBlockedAccepts() {
		for (const action of actions) {
			const choice = choices[action.key];
			if (choice.decision === 'accept' && !accepts(action)) choice.decision = null;
		}
	}

	function choose(key: string, decision: Decision) {
		choices[key].decision = decision;
		dropBlockedAccepts();
	}

	function acceptAll() {
		for (const action of actions) {
			if (accepts(action)) choices[action.key].decision = 'accept';
		}
	}

	async function send() {
		if (submitting || !canSend(actions, choices)) return;
		submitting = true;
		const res = await sendReview({
			author_workflow_id: session.chat_workflow_id,
			tool_id: toolId,
			decisions: reviewDecisions({ actions, profile }, choices)
		});
		submitting = false;
		if (res.isOk) {
			status = m.Review_sent();
			onSettled();
			return;
		}
		status = res.error.status === 409 ? m.Review_not_pending() : res.error.message;
		if (res.error.status === 409) onSettled();
	}

	const build = actions[0]?.build ?? '';
</script>

<section
	class="space-y-3 rounded-[10px] border bg-white p-4 text-sm"
	aria-label={m.Review_actions_title()}
>
	<h3 class="font-semibold">{m.Review_actions_title()}</h3>
	<p class="break-words">{m.Review_actions_details({ build })}</p>

	{#if warnings.length}
		<Alert variant="warning" aria-label={m.Review_fcaf_warnings()}>
			<p class="font-bold">{m.Review_fcaf_warnings()}</p>
			{#each warnings as warning, index (index)}
				<p class="break-words">{warning}</p>
			{/each}
		</Alert>
	{/if}

	{#if profile}
		<section class="space-y-1" aria-label={m.Review_wallet_profile()}>
			<h4 class="font-medium">{m.Review_wallet_profile()}</h4>
			<pre
				class="max-h-[40dvh] overflow-auto rounded-md border bg-slate-50 p-2 font-mono text-xs">{JSON.stringify(
					profile,
					null,
					2
				)}</pre>
		</section>
	{/if}

	<Tabs.Root bind:value={selected} class="w-full">
		<Tabs.List class="flex h-auto w-full flex-wrap">
			{#each actions as action (action.key)}
				<Tabs.Trigger value={action.key} class="max-w-full min-w-0">
					<span class="truncate">{action.name || action.key}</span>
				</Tabs.Trigger>
			{/each}
		</Tabs.List>

		{#each actions as action (action.key)}
			{@const choice = choices[action.key]}
			{@const blocker = blockedBy(action, choices)}
			{@const planPresent = hasPlan(action)}
			{@const verified = acceptable[action.key]}
			<Tabs.Content value={action.key} class="space-y-3 pt-2">
				{#if action.replaces}
					<p class="font-medium break-words text-amber-700">
						{m.Review_replaces({ action: action.replaces })}
					</p>
					{#if action.used_by > 0}
						<p class="break-words text-amber-700">
							{m.Review_used_by({ count: action.used_by })}
						</p>
					{/if}
				{/if}

				{#if action.starts_from?.length}
					<p class="break-words">
						{m.Review_starts_from({
							names: action.starts_from.map((key) => nameOf(action, key)).join(', ')
						})}
					</p>
				{/if}

				{#if action.error}
					<Alert variant="destructive">
						<p class="font-bold">{m.Review_previous_save_failed()}</p>
						<p class="break-words">{action.error}</p>
					</Alert>
				{/if}

				{#if action.note}
					<Alert>
						<p class="font-bold">{m.Review_agent_note()}</p>
						<p class="break-words whitespace-pre-wrap">{action.note}</p>
					</Alert>
				{/if}

				<div class="space-y-1">
					<p class="font-medium">{m.Accepted_YAML()}</p>
					<pre
						class="max-h-[40dvh] overflow-auto rounded-md border bg-slate-50 p-2 font-mono text-xs"
						aria-label={m.Accepted_YAML()}>{action.code}</pre>
				</div>

				{#if planPresent}
					<details class="space-y-1">
						<summary class="cursor-pointer font-medium">{m.Plan()}</summary>
						<pre
							class="mt-1 max-h-[40dvh] overflow-auto rounded-md border bg-slate-50 p-2 font-mono text-xs"
							aria-label={m.Plan()}>{JSON.stringify(action.plan, null, 2)}</pre>
					</details>
				{:else}
					<p>{m.Review_no_plan()}</p>
				{/if}

				{#if planPresent && verified === false}
					<p class="break-words">{m.Review_hash_mismatch()}</p>
				{/if}
				{#if blocker}
					<p class="break-words">
						{m.Review_blocked_by({ name: nameOf(action, blocker) })}
					</p>
				{/if}

				<RadioGroup.Root
					bind:value={
						() => choice.decision ?? '',
						(value) => choose(action.key, value as Decision)
					}
					disabled={submitting}
				>
					<div class="flex items-center gap-2">
						<RadioGroup.Item
							value="accept"
							id="{toolId}-{action.key}-accept"
							disabled={!verified || !!blocker}
						/>
						<Label for="{toolId}-{action.key}-accept"
							>{m.Review_decision_accept()}</Label
						>
					</div>
					<div class="flex items-center gap-2">
						<RadioGroup.Item value="refuse" id="{toolId}-{action.key}-refuse" />
						<Label for="{toolId}-{action.key}-refuse"
							>{m.Review_decision_refuse()}</Label
						>
					</div>
					<div class="flex items-center gap-2">
						<RadioGroup.Item value="change" id="{toolId}-{action.key}-change" />
						<Label for="{toolId}-{action.key}-change"
							>{m.Review_decision_change()}</Label
						>
					</div>
				</RadioGroup.Root>

				{#if choice.decision === 'change'}
					<Textarea
						bind:value={choices[action.key].comment}
						placeholder={m.Review_change_placeholder()}
						aria-label={m.Review_change_placeholder()}
						maxlength={2000}
						disabled={submitting}
					/>
				{/if}
			</Tabs.Content>
		{/each}
	</Tabs.Root>

	{#if status}
		<p role="status" aria-live="polite" class="break-words">{status}</p>
	{/if}

	<div class="flex flex-wrap gap-2">
		<Button variant="outline" onclick={acceptAll} disabled={submitting}>
			{m.Review_accept_all()}
		</Button>
		<Button onclick={send} disabled={submitting || !canSend(actions, choices)}>
			{m.Review_send_decisions()}
		</Button>
	</div>
</section>

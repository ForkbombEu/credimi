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
	import { blockedBy, canAccept, canSend, decidedActions, hasPlan, type Choices } from './review';

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

	const names = $derived(Object.fromEntries(actions.map((action) => [action.key, action.name])));

	function nameOf(key: string): string {
		return names[key] || key;
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
			decisions: decidedActions(actions, choices).map((action) => {
				const choice = choices[action.key];
				const decision = choice.decision as Decision;
				return {
					key: action.key,
					name: action.name,
					candidate_sha256: action.candidate_sha256,
					decision,
					comment: decision === 'change' ? choice.comment.trim() : '',
					plan: action.plan
				};
			})
		});
		submitting = false;
		if (res.isOk) {
			if (res.value.tool_id !== toolId) return;
			status = m.Review_sent();
			onSettled();
			return;
		}
		status = res.error.status === 409 ? m.Review_not_pending() : res.error.message;
		if (res.error.status === 409) onSettled();
	}

	const build = $derived(actions[0]?.build ?? '');
</script>

<section
	class="space-y-3 rounded-[10px] border bg-white p-4 text-sm"
	aria-label={m.Review_actions_title()}
>
	<h3 class="font-semibold">{m.Review_actions_title()}</h3>
	<p class="break-words">{m.Review_actions_details({ build })}</p>

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
						{m.Review_starts_from({ names: action.starts_from.map(nameOf).join(', ') })}
					</p>
				{/if}

				{#if action.error}
					<Alert variant="destructive">
						<p class="font-bold">{m.Review_previous_save_failed()}</p>
						<p class="break-words">{action.error}</p>
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
					<p class="break-words">{m.Review_blocked_by({ name: nameOf(blocker) })}</p>
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
							disabled={!verified || !planPresent || !!blocker}
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

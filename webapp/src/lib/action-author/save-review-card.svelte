<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { untrack } from 'svelte';

	import Button from '@/components/ui-custom/button.svelte';
	import { m } from '@/i18n';

	import { reviewCandidate, type PendingReview, type Session } from './api';
	import { canAccept, hasPlan } from './review';

	//

	type Props = {
		session: Session;
		review: PendingReview;
		onSettled: () => void;
	};

	let { session, review, onSettled }: Props = $props();

	// The YAML shown must match candidate_sha256, the digest Credimi's approval signs.
	let acceptable = $state(false);
	let status = $state('');
	let submitting = $state(false);

	// Keyed on primitives: each poll brings a fresh but equal review object, which must not reset
	// the Accept button.
	const toolId = $derived(review.tool_id);
	const code = $derived(review.code);
	const digest = $derived(review.candidate_sha256);
	const planPresent = $derived(hasPlan(review));

	$effect(() => {
		const current = { toolId, code, digest, planPresent };
		acceptable = false;
		canAccept(untrack(() => review)).then((ok) => {
			if (current.toolId !== toolId || current.digest !== digest) return;
			acceptable = ok;
			if (ok) status = '';
			else status = current.planPresent ? m.Review_hash_mismatch() : m.Review_no_plan();
		});
	});

	async function decide(accepted: boolean) {
		if (submitting || (accepted && !acceptable)) return;
		submitting = true;
		const res = await reviewCandidate({
			namespace: session.namespace,
			author_workflow_id: session.chat_workflow_id,
			tool_id: review.tool_id,
			candidate_sha256: review.candidate_sha256,
			plan: review.plan,
			accepted
		});
		submitting = false;
		if (res.isOk) {
			if (res.value.accepted !== accepted || res.value.tool_id !== review.tool_id) return;
			status = accepted ? m.Review_accepted() : m.Review_refused();
			onSettled();
			return;
		}
		status = res.error.status === 409 ? m.Review_not_pending() : res.error.message;
		if (res.error.status === 409) onSettled();
	}
</script>

<section
	class="space-y-3 rounded-[10px] border bg-white p-4 text-sm"
	aria-label={m.Review_save_title({ name: review.name })}
>
	<h3 class="font-semibold">{m.Review_save_title({ name: review.name })}</h3>
	<p class="break-words">{m.Review_save_details({ build: review.build })}</p>

	<div class="space-y-1">
		<p class="font-medium">{m.Accepted_YAML()}</p>
		<pre
			class="max-h-[40dvh] overflow-auto rounded-md border bg-slate-50 p-2 font-mono text-xs"
			aria-label={m.Accepted_YAML()}>{review.code}</pre>
	</div>

	{#if hasPlan(review)}
		<div class="space-y-1">
			<p class="font-medium">{m.Plan()}</p>
			<pre
				class="max-h-[40dvh] overflow-auto rounded-md border bg-slate-50 p-2 font-mono text-xs"
				aria-label={m.Plan()}>{JSON.stringify(review.plan, null, 2)}</pre>
		</div>
	{:else}
		<p>{m.Review_no_plan()}</p>
	{/if}

	{#if status}
		<p role="status" aria-live="polite" class="break-words">{status}</p>
	{/if}

	<div class="flex gap-2">
		<Button onclick={() => decide(true)} disabled={!acceptable || submitting}>
			{m.Accept_action()}
		</Button>
		<Button variant="outline" onclick={() => decide(false)} disabled={submitting}>
			{m.Refuse_action()}
		</Button>
	</div>
</section>

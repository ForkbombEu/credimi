<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { m } from '@/i18n';

	import { formatDurationParts, type PipelineProgress } from './progress';

	//

	type Props = {
		progress?: PipelineProgress | null;
	};

	let { progress }: Props = $props();

	//

	let now = $state(Date.now());
	let snapshot = $state({ elapsed: 0, at: Date.now() });

	$effect(() => {
		if (progress) {
			snapshot = { elapsed: progress.elapsed_seconds, at: Date.now() };
		}
	});

	$effect(() => {
		const interval = setInterval(() => (now = Date.now()), 1000);
		return () => clearInterval(interval);
	});

	const elapsed = $derived(
		progress ? snapshot.elapsed + Math.max(0, (now - snapshot.at) / 1000) : 0
	);
	const percent = $derived(
		progress ? Math.min((elapsed / progress.expected_duration_seconds) * 100, 99) : null
	);
	const eta = $derived(
		progress ? Math.max(progress.expected_duration_seconds - elapsed, 0) : null
	);
</script>

<div class="mt-1 w-32 min-w-24">
	{#if progress && percent !== null && eta !== null}
		<div
			class="h-1.5 w-full overflow-hidden rounded-full bg-slate-200"
			role="progressbar"
			aria-valuenow={Math.round(percent)}
			aria-valuemin={0}
			aria-valuemax={100}
		>
			<div
				class="h-full rounded-full bg-primary transition-[width] duration-1000 ease-linear"
				style="width: {percent}%"
			></div>
		</div>
		<p class="mt-0.5 text-[10px] whitespace-nowrap text-muted-foreground">
			<span class="font-mono">{Math.round(percent)}%</span>
			·
			{m.eta_remaining({ time: formatDurationParts(eta) })}
		</p>
	{:else}
		<div
			class="h-1.5 w-full overflow-hidden rounded-full bg-slate-200"
			role="progressbar"
			aria-label={m.Running()}
		>
			<div class="indeterminate h-full w-1/3 rounded-full bg-primary"></div>
		</div>
	{/if}
</div>

<style lang="postcss">
	@reference "tailwindcss";

	.indeterminate {
		animation: progress-slide 1.5s ease-in-out infinite;
	}

	@keyframes progress-slide {
		0% {
			transform: translateX(-100%);
		}
		100% {
			transform: translateX(300%);
		}
	}
</style>

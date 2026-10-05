<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { m } from '@/i18n';

	import type { WorkflowStatus } from './types';

	type Props = {
		status: WorkflowStatus;
		/** Optional heartbeat animation delay in ms (Running only). */
		delay?: number;
	};

	let { status, delay = 0 }: Props = $props();

	/** Background colors live with the badge — do not rely on a Tailwind safelist elsewhere. */
	const statusClass: Record<WorkflowStatus, string> = {
		Running: 'bg-blue-300',
		TimedOut: 'bg-orange-200',
		Completed: 'bg-green-200',
		Failed: 'bg-red-200',
		ContinuedAsNew: 'bg-purple-200',
		Canceled: 'bg-slate-100',
		Terminated: 'bg-yellow-200',
		Unspecified: 'bg-slate-100'
	};

	const colorClass = $derived(statusClass[status] ?? statusClass.Unspecified);
	const label = $derived.by(() => {
		switch (status) {
			case 'Running':
				return m.Running();
			case 'TimedOut':
				return m.Timed_Out();
			case 'Completed':
				return m.Completed();
			case 'Failed':
				return m.Failed();
			case 'ContinuedAsNew':
				return m.Continued_as_New();
			case 'Canceled':
				return m.Canceled();
			case 'Terminated':
				return m.Terminated();
			case 'Unspecified':
			default:
				return m.Unspecified();
		}
	});
</script>

<span
	class={[
		'inline-flex items-center gap-1 rounded-sm px-1 py-0.5 text-sm leading-4 font-medium whitespace-nowrap text-black',
		colorClass
	]}
	data-testid="workflow-status"
>
	{label}
	{#if status === 'Running'}
		<span class="heart-beat" style={`--animation-delay: ${delay}ms`}>
			<span class="heart-rate">
				<svg
					xmlns="http://www.w3.org/2000/svg"
					width="30"
					height="18"
					viewBox="0 0 150 73"
					aria-hidden="true"
					focusable="false"
				>
					<polyline
						fill="none"
						stroke="#000000"
						stroke-width="3"
						stroke-miterlimit="10"
						points="0,45.486 18.514,45.486 24.595,33.324 32.676,45.486 37.771,45.486 42.838,55.622 51.959,18 56.067,45 60.067,60.729 63.122,45.486 77.297,45.486 83.379,41.419 90.473,45.486 100,45.486"
					/>
				</svg>
				<span class="fade-in"></span>
				<span class="fade-out"></span>
			</span>
		</span>
	{/if}
</span>

<style>
	.heart-beat {
		position: relative;
		display: flex;
		align-items: center;
		justify-content: center;
		background-color: rgb(147 187 253);
		text-align: center;
	}

	.heart-rate {
		position: relative;
		width: 20px;
		height: 18px;
		margin: 0;
	}

	.fade-in {
		position: absolute;
		right: 0;
		top: 0;
		height: 100%;
		width: 100%;
		background-color: rgb(147 187 253);
		animation: heartRateIn 2s linear infinite;
		animation-delay: var(--animation-delay, 0);
	}

	.fade-out {
		position: absolute;
		height: 100%;
		top: 0;
		left: 0;
		animation: heartRateOut 2s linear infinite;
		animation-delay: var(--animation-delay, 0);
		background: linear-gradient(
			to right,
			rgb(147 187 253) 0%,
			rgb(147 187 253) 80%,
			rgb(255 255 255 / 0%) 100%
		);
	}

	@keyframes heartRateIn {
		0% {
			width: 100%;
		}
		50%,
		100% {
			width: 0;
		}
	}

	@keyframes heartRateOut {
		0% {
			width: 0%;
		}
		100% {
			width: 100%;
		}
	}
</style>

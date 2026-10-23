<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { supportLabels, type Support } from './comparison';

	const { value }: { value: Support } = $props();
</script>

<!-- Shape carries the meaning as well as color: full, half, empty ring. -->
<span
	class="dot dot-{value}"
	role="img"
	aria-label={supportLabels[value]}
	title={supportLabels[value]}
></span>

<style>
	.dot {
		position: relative;
		display: inline-block;
		width: 1.125rem;
		height: 1.125rem;
		overflow: hidden;
		vertical-align: middle;
		border-radius: 9999px;
	}

	.dot-core {
		background: var(--color-green-600);
	}

	.dot-partial {
		border: 2px solid var(--color-amber-500);
	}

	.dot-partial::after {
		position: absolute;
		inset: 0 50% 0 0;
		background: var(--color-amber-500);
		content: '';
	}

	.dot-none {
		width: 0.875rem;
		height: 0.875rem;
		border: 2px solid var(--color-slate-400);
	}
</style>

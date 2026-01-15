<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV
SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { m } from '@/i18n';

	interface Props {
		title?: string;
		content: object;
	}

	let { title = m.opentelemetry_data(), content }: Props = $props();
</script>

<div class="details-wrapper">
	<details class="details-container">
		<summary class="details-summary">
			{title}
		</summary>
		<div class="details-content">
			<pre class="details-pre">{JSON.stringify(content, null, 2)}</pre>
		</div>
	</details>
</div>

<style lang="postcss">
	.details-wrapper {
		@apply mt-8;
	}

	.details-container {
		@apply rounded-lg border;
	}

	.details-summary {
		@apply cursor-pointer bg-gray-50 px-6 py-3 font-semibold;
	}

	.details-content {
		@apply p-6;
	}

	.details-pre {
		@apply overflow-x-auto rounded bg-gray-100 p-4 text-sm;
	}
</style>

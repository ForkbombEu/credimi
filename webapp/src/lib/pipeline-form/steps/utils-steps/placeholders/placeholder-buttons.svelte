<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { m } from '@/i18n';

	import { formatPlaceholder, getPlaceholderDescription, Placeholder } from './utils';
</script>

<div class="space-y-2 text-xs text-muted-foreground">
	<p>{m.pipeline_placeholder_description()}</p>

	<ul class="flex flex-col gap-2">
		{#each Object.values(Placeholder) as placeholder (placeholder)}
			<li class="flex flex-col gap-0.5">
				<p
					class="w-fit self-start rounded-md bg-slate-100 px-1.5 py-0.5 font-mono text-[11px]"
				>
					{formatPlaceholder(placeholder)}
				</p>
				<p>{getPlaceholderDescription(placeholder)}</p>
			</li>
		{/each}
	</ul>
</div>

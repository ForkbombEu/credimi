<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import T from '@/components/ui-custom/t.svelte';
	import { Badge } from '@/components/ui/badge';
	import { m } from '@/i18n';

	import type { FcafProgress } from './api';

	import { progressDone, roleStatusLabel, roleStatusVariant } from './fcaf';

	//

	type Props = { progress: FcafProgress };

	let { progress }: Props = $props();
</script>

<section
	class="flex flex-col gap-2 rounded-[10px] border bg-white p-3"
	aria-label={m.Fcaf_progress()}
>
	<div class="flex flex-wrap items-baseline justify-between gap-2">
		<T class="font-semibold">{m.Fcaf_progress()}</T>
		<span class="text-xs text-muted-foreground">
			{m.Fcaf_progress_count({
				done: progressDone(progress.roles),
				total: progress.roles.length
			})}
		</span>
	</div>
	<ol class="space-y-2">
		{#each progress.roles as role (role.role)}
			{@const current = role.role === progress.current_role}
			<li class="space-y-1" aria-current={current ? 'step' : undefined}>
				<div class="flex items-center justify-between gap-2">
					<span class={['truncate font-mono text-xs', { 'font-semibold': current }]}>
						{role.role}
					</span>
					<Badge variant={roleStatusVariant(role.status)}>
						{roleStatusLabel(role.status)}
					</Badge>
				</div>
				{#if role.note}
					<p class="line-clamp-2 text-xs text-muted-foreground" title={role.note}>
						{role.note}
					</p>
				{/if}
			</li>
		{/each}
	</ol>
</section>

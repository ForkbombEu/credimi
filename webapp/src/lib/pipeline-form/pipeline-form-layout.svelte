<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import { ArrowLeftIcon } from '@lucide/svelte';

	import T from '@/components/ui-custom/t.svelte';
	import { Button } from '@/components/ui/button';
	import { Separator } from '@/components/ui/separator';
	import { m } from '@/i18n';

	//

	type Props = {
		children?: Snippet;
		topbarMiddle?: Snippet;
		topbarRight?: Snippet;
		title: string;
		class?: string;
	};

	let { children, title, topbarMiddle, topbarRight, class: className }: Props = $props();
</script>

<div class={['bg-secondary flex flex-col gap-4 px-4 pb-4 pt-2', className]}>
	<div class="flex items-center justify-between">
		<div class="flex items-center gap-3">
			<Button class="h-6 px-0" href="/my/pipelines" variant="link">
				<ArrowLeftIcon />
				{m.Back()}
			</Button>
			<Separator orientation="vertical" class="self-stretch bg-slate-400" />
			<T tag="h3" class="text-xl">
				{title}
			</T>
		</div>

		<div>
			{@render topbarMiddle?.()}
		</div>

		<div class="flex items-center gap-2">
			{@render topbarRight?.()}
		</div>
	</div>

	<div class="flex-1 min-h-0">
		{@render children?.()}
	</div>
</div>

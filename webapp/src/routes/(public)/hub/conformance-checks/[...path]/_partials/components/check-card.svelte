<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Standard, Suite, Version } from '$lib/standards';

	import T from '@/components/ui-custom/t.svelte';
	import { localizeHref } from '@/i18n';

	//

	type Props = {
		standard: Standard;
		version: Version;
		suite: Suite;
		test: string;
	};

	let { standard, version, suite, test }: Props = $props();

	const testName = $derived(test.split('/').at(-1)?.replaceAll('+', ' • '));
	const href = $derived(localizeHref(`/marketplace/conformance-checks/${test}`));
</script>

<a
	{href}
	class={[
		'border-primary bg-card text-card-foreground ring-primary relative',
		'flex flex-col justify-between gap-4',
		'overflow-visible rounded-lg border p-4 shadow-sm transition-all hover:-translate-y-2 hover:ring-2'
	]}
>
	<div class="space-y-1">
		<p class="text-muted-foreground text-xs">
			{standard.name} • {version.name} • {suite.name}
		</p>
		<T class="overflow-hidden text-ellipsis font-semibold">{testName}</T>
	</div>
</a>

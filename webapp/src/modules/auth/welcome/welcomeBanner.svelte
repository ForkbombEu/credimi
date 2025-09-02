<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import { appName } from '@/brand';
	import Alert from '@/components/ui-custom/alert.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';

	interface Props {
		class?: string;
		children?: Snippet;
	}

	let { class: className = '', children }: Props = $props();
</script>

<Alert variant="info" class="space-y-3 {className}">
	<T tag="h2" class="-rotate-2 text-ellipsis text-balance text-center">
		{m.welcome_to_app({ appName })}
	</T>

	{@render children?.()}
</Alert>

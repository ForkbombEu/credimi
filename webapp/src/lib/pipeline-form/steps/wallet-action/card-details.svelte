<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import WalletActionTags from '$lib/components/wallet-action-tags.svelte';

	import { Badge } from '@/components/ui/badge/index.js';
	import { m } from '@/i18n/index.js';

	import type { WalletActionStepData } from './wallet-action-step-form.svelte.js';

	//

	type Props = {
		data: WalletActionStepData;
	};

	let { data }: Props = $props();
</script>

<WalletActionTags action={data.action}>
	{#if !data.action.published}
		<Badge variant="outline">
			{m.private()}
		</Badge>
	{/if}
</WalletActionTags>

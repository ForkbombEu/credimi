<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { page } from '$app/state';
	import type { Snippet } from 'svelte';
	import { type WalletTestParams, getWalletTestParams } from './index.js';
	import EmptyState from '@/components/ui-custom/emptyState.svelte';

	type Props = {
		ifValid: Snippet<[WalletTestParams]>;
	};

	const { ifValid }: Props = $props();

	const params = $derived(getWalletTestParams(page.url));
</script>

{#if params instanceof Error}
	<EmptyState title="Wrong or missing URL parameters"></EmptyState>
{:else}
	{@render ifValid(params)}
{/if}

<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { initI18n } from '@forkbombeu/temporal-ui';
	import { type Snippet } from 'svelte';

	type Props = {
		children: Snippet;
	};

	const { children }: Props = $props();
</script>

{#await initI18n() then _}
	{@render children()}
{/await}

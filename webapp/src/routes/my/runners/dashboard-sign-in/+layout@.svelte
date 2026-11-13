<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import AuthLayout from '@/auth/authLayout.svelte';
	import LanguageSelect from '@/i18n/languageSelect.svelte';

	//

	let { children }: { children?: Snippet } = $props();
</script>

<!-- A sign-in step, laid out like the Credimi login page rather than the dashboard. -->
<AuthLayout>
	{#snippet topbarRight()}
		<LanguageSelect />
	{/snippet}

	{@render children?.()}
</AuthLayout>

<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import AuthLayout from '@/auth/authLayout.svelte';
	import LanguageSelect from '@/i18n/languageSelect.svelte';
	import type { Snippet } from 'svelte';

	//

	interface Props {
		children?: Snippet;
	}

	let { children }: Props = $props();
</script>

<AuthLayout>
	{#snippet topbarRight()}
		<LanguageSelect />
	{/snippet}

	{@render children?.()}
</AuthLayout>

<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { String } from 'effect';

	import type { CredentialsResponse } from '@/pocketbase/types';

	import Avatar from '@/components/ui-custom/avatar.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { localizeHref, m } from '@/i18n';

	type Props = {
		credential: CredentialsResponse;
		class?: string;
	};

	const { credential, class: className = '' }: Props = $props();

	const properties: Record<string, string> = {};
	if (isValid(credential.format)) properties[m.Format()] = credential.format;
	if (isValid(credential.locale)) properties[m.Locale()] = credential.locale.toUpperCase();

	function isValid(value: string) {
		return String.isNonEmpty(value.trim());
	}
</script>

<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
<a
	href={localizeHref(`/marketplace/credentials/${credential.id}`)}
	class="border-primary bg-card text-card-foreground ring-primary flex flex-col gap-6 rounded-xl border p-6 shadow-sm transition-transform hover:-translate-y-2 hover:ring-2 {className}"
>
	<div class="flex items-center gap-2">
		{#if credential.logo}
			<Avatar src={credential.logo} class="!rounded-sm" hideIfLoadingError />
		{/if}
		<T class="font-semibold">{credential.display_name}</T>
	</div>

	<div class="space-y-1">
		{#each Object.entries(properties) as [key, value]}
			<T class="text-sm text-slate-400">{key}: <span class="text-primary">{value}</span></T>
		{/each}
	</div>
</a>

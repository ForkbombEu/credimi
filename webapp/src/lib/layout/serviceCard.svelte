<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { String } from 'effect';

	import Avatar from '@/components/ui-custom/avatar.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { cn } from '@/components/ui/utils';
	import { m, localizeHref } from '@/i18n';
	import { type CredentialIssuersResponse } from '@/pocketbase/types';

	//

	type Props = {
		service: CredentialIssuersResponse;
		class?: string;
	};

	const { service, class: className = '' }: Props = $props();

	const title = $derived(String.isNonEmpty(service.name) ? service.name : service.url);
</script>

<a
	href={localizeHref(`/services/${service.id}`)}
	class={cn(
		'flex flex-col gap-4 rounded-lg border border-primary bg-card p-6 text-card-foreground shadow-sm ring-primary transition-all hover:-translate-y-2 hover:ring-2',
		{ className }
	)}
>
	<div class="space-y-4">
		<div class="flex items-center gap-2">
			{#if service.logo_url && String.isNonEmpty(service.logo_url)}
				<Avatar src={service.logo_url} class="!rounded-sm" hideIfLoadingError />
			{/if}
			<T class="overflow-hidden text-ellipsis font-semibold">{title}</T>
		</div>
		{#if String.isNonEmpty(service.description)}
			<T tag="p" class="block font-normal leading-snug">{service.description}</T>
		{/if}
	</div>
	<div class="flex flex-col items-start gap-2 overflow-hidden text-muted-foreground">
		{#if String.isNonEmpty(service.url)}
			<T tag="small">{service.url}</T>
		{/if}
		{#if String.isNonEmpty(service.homepage_url)}
			<T tag="small">{service.homepage_url}</T>
		{/if}
		{#if String.isNonEmpty(service.repo_url)}
			<T tag="small">{service.repo_url}</T>
		{/if}
	</div>
</a>

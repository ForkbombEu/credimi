<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { localizeHref, m } from '@/i18n';
	import T from '@/components/ui-custom/t.svelte';
	import { Badge } from '@/components/ui/badge';
	import type { WalletsResponse } from '@/pocketbase/types';

	import { Separator } from '@/components/ui/separator';
	import { type ConformanceCheck } from '$services-and-products/_wallets/wallet-form-checks-table.svelte';
	import { cn } from '@/components/ui/utils';
	import Avatar from '@/components/ui-custom/avatar.svelte';
	import { pb } from '@/pocketbase';
	import RenderHtml from '@/components/ui-custom/renderHTML.svelte';

	type Props = {
		app: WalletsResponse;
		class?: string;
	};

	const { app, class: className = '' }: Props = $props();

	// Prioritize logo file over logo_url
	const logo = $derived.by(() => {
		if (app.logo) {
			return pb.files.getURL(app, app.logo);
		} else if (app.logo_url) {
			return app.logo_url;
		}
		return null;
	});
	const conformanceChecks = $derived(app.conformance_checks) as ConformanceCheck[];
</script>

<a
	href={localizeHref(`/apps/${app.id}`)}
	class={cn(
		'border-primary bg-card text-card-foreground ring-primary block overflow-auto rounded-lg border p-6 shadow-sm transition-all hover:-translate-y-2 hover:ring-2 ',
		className
	)}
>
	<div class="space-y-2 overflow-x-auto sm:overflow-hidden">
		<div class="flex flex-row justify-between">
			<div class="flex flex-col gap-2">
				<div class="flex items-center gap-2">
					{#if logo}
						<Avatar src={logo} class="!rounded-sm" hideIfLoadingError />
					{/if}
					<T class="font-semibold">{app.name}</T>
				</div>
				<T class="text-xs text-gray-400">
					<RenderHtml content={app.description} />
				</T>
			</div>
		</div>

		<Separator />

		<div class="flex flex-wrap gap-2">
			{#if conformanceChecks && conformanceChecks.length > 0}
				{#each conformanceChecks as check}
					<Badge variant={check.status === 'success' ? 'secondary' : 'destructive'}>
						{check.test}
					</Badge>
				{/each}
			{:else}
				<T class="text-gray-300">{m.No_conformance_checks_available()}</T>
			{/if}
		</div>
	</div>
</a>

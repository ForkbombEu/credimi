<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { RecordDelete, RecordEdit } from '@/collections-components/manager';
	import Avatar from '@/components/ui-custom/avatar.svelte';
	import Card from '@/components/ui-custom/card.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Separator } from '@/components/ui/separator';
	import { m } from '@/i18n';
	import type { CredentialsResponse, VerifiersResponse } from '@/pocketbase/types';
	import { pb } from '@/pocketbase';

	//

	type Props = {
		verifier: VerifiersResponse;
		credentials?: CredentialsResponse[];
	};

	let { verifier, credentials }: Props = $props();
	const avatarSrc = $derived(pb.files.getURL(verifier, verifier.logo));
</script>

<Card class="bg-card" contentClass="space-y-2 p-4">
	<div class="flex items-center justify-between">
		<div class="flex items-center gap-4">
			<Avatar src={avatarSrc} fallback={verifier.name} class="rounded-sm border" />
			<div>
				<T class="font-bold">{verifier.name}</T>
				<T class="text-xs text-gray-400">{verifier.url}</T>
			</div>
		</div>
		<div>
			<RecordEdit record={verifier} />
			<RecordDelete record={verifier} />
		</div>
	</div>

	<Separator />

	<div class="space-y-0.5 text-sm">
		<T class="font-semibold">{m.Linked_credentials()}</T>
		{#if credentials?.length === 0}
			<T class="text-gray-300">{m.No_credentials_available()}</T>
		{:else if credentials}
			<ul class="list-disc space-y-0.5 pl-4">
				{#each credentials as credential}
					<li>
						<T>{credential.key}</T>
					</li>
				{/each}
			</ul>
		{/if}
	</div>
</Card>

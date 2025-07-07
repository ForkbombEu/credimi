<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { CollectionForm } from '@/collections-components';
	import * as Dialog from '@/components/ui/dialog';
	import type { CredentialsFormData, CredentialsRecord } from '@/pocketbase/types';
	import Button from '@/components/ui/button/button.svelte';
	import { Pencil } from 'lucide-svelte';
	import { m } from '@/i18n';

	type Props = {
		credential: CredentialsRecord;
		onSuccess: () => void;
	};

	let { credential, onSuccess }: Props = $props();

	let open = $state(false);
</script>

<Dialog.Root bind:open>
	<Dialog.Trigger>
		{#snippet child({ props })}
			<Button {...props} variant="outline" size="sm" class="h-fit py-1 text-xs">
				<Pencil size={10} />
			</Button>
		{/snippet}
	</Dialog.Trigger>

	<Dialog.Content class=" sm:max-w-[425px]">
		<Dialog.Header>
			<Dialog.Title>Credential {credential.key}</Dialog.Title>
		</Dialog.Header>

		<div class="pt-8">
			<CollectionForm
				collection="credentials"
				recordId={credential.id}
				initialData={credential}
				fieldsOptions={{
					exclude: [
						'format',
						'issuer_name',
						'type',
						'name',
						'locale',
						'logo',
						'description',
						'credential_issuer',
						'json',
						'key',
						'owner'
					],
					order: ['deeplink'],
					labels: {
						published: m.Publish_to_marketplace()
					}
				}}
				onSuccess={() => {
					open = false;
					onSuccess();
				}}
			/>
		</div>
	</Dialog.Content>
</Dialog.Root>

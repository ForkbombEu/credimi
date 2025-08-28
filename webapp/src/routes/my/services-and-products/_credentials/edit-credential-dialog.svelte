<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { CredentialIssuersResponse, CredentialsResponse } from '@/pocketbase/types';
	import { Pencil } from 'lucide-svelte';
	import { m } from '@/i18n';
	import Sheet from '@/components/ui-custom/sheet.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import EditCredentialForm from './edit-credential-form.svelte';

	type Props = {
		credential: CredentialsResponse;
		credentialIssuer: CredentialIssuersResponse;
		onSuccess?: () => void;
	};

	let { credential, credentialIssuer, onSuccess }: Props = $props();
</script>

<Sheet title="{m.Edit_credential()}: {credential.name || credential.key}">
	{#snippet trigger({ sheetTriggerAttributes, openSheet })}
		<IconButton size="sm" variant="outline" icon={Pencil} {...sheetTriggerAttributes} />
	{/snippet}

	{#snippet content({ closeSheet })}
		<EditCredentialForm
			{credential}
			{credentialIssuer}
			onSuccess={() => {
				closeSheet();
				onSuccess?.();
			}}
		/>
	{/snippet}
</Sheet>

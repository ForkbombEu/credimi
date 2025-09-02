<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Pencil } from 'lucide-svelte';

	import type { CredentialIssuersResponse, CredentialsResponse } from '@/pocketbase/types';

	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import Sheet from '@/components/ui-custom/sheet.svelte';
	import { m } from '@/i18n';

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

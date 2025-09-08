<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { credentialissuer, form, pocketbase as pb } from '#';

	import Sheet from '@/components/ui-custom/sheet.svelte';
	import * as AlertDialog from '@/components/ui/alert-dialog/index.js';

	//

	type Props = {
		manager: credentialissuer.Manager;
	};

	let { manager }: Props = $props();
	const { sheet, discardAlert } = manager;
</script>

<Sheet bind:open={sheet.isOpen}>
	{#snippet content()}
		{#if manager.currentState === 'init'}
			<form.Component form={manager.importForm} />
		{/if}
		{#if manager.importedIssuer}
			<div>
				<h2>Imported Issuer</h2>
				<p>{manager.importedIssuer.name}</p>
			</div>
		{/if}
		<pb.recordform.Component form={manager.recordForm} />
	{/snippet}
</Sheet>

<AlertDialog.Root open={discardAlert.window.isOpen}>
	<AlertDialog.Content>
		<AlertDialog.Header>
			<AlertDialog.Title>Are you absolutely sure?</AlertDialog.Title>
			<AlertDialog.Description>
				This action cannot be undone. This will permanently delete your account and remove
				your data from our servers.
			</AlertDialog.Description>
		</AlertDialog.Header>

		<AlertDialog.Footer>
			<AlertDialog.Action onclick={() => discardAlert.confirm()}>Continue</AlertDialog.Action>
			<AlertDialog.Cancel onclick={() => discardAlert.dismiss()}>Cancel</AlertDialog.Cancel>
		</AlertDialog.Footer>
	</AlertDialog.Content>
</AlertDialog.Root>

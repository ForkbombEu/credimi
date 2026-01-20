<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import Checkbox from '@/components/ui/checkbox/checkbox.svelte';
	import * as Table from '@/components/ui/table';
	import { m } from '@/i18n/index.js';

	//

	type CredentialFormat = 'iso_mdl' | 'sd_jwt_vc';
	type Props = {
		suiteUid: string;
		suiteFiles: string[];
	};
	let { suiteUid, suiteFiles }: Props = $props();

	//

	const getCredentialFormatStyle = (format: CredentialFormat): string => {
		switch (format) {
			case 'iso_mdl':
				return 'bg-blue-100 text-blue-800 border border-blue-200';
			case 'sd_jwt_vc':
				return 'bg-red-100 text-red-800 border border-red-200';
			default:
				return 'bg-neutral-100 text-neutral-800 border border-neutral-200';
		}
	};
</script>

<Table.Root class="rounded-lg bg-white">
	<Table.Header>
		<Table.Row>
			<Table.Head></Table.Head>
			<Table.Head>{m.Credential_Format()}</Table.Head>
			<Table.Head>{m.Client_id_Scheme()}</Table.Head>
			<Table.Head>{m.Request_Method()}</Table.Head>
			<Table.Head>{m.Response_Mode()}</Table.Head>
		</Table.Row>
	</Table.Header>
	<Table.Body>
		{#each suiteFiles as fileId (fileId)}
			{@const value = `${suiteUid}/${fileId}`}
			{@const label = fileId.split('.').slice(0, -1).join('.')}
			{@const [format, scheme, method, mode] = label.split('+')}
			{console.log(label, format, scheme, method, mode)}
			<Table.Row class="even:bg-muted border-0 align-middle">
				<Table.Cell>
					<Checkbox {value} />
				</Table.Cell>
				<Table.Cell>
					<span
						class="{getCredentialFormatStyle(
							format as CredentialFormat
						)} rounded border-[0.5px] px-2 py-1 font-medium leading-[14px]"
						>{format}</span
					>
				</Table.Cell>
				<Table.Cell>
					{scheme}
				</Table.Cell>
				<Table.Cell>
					{method}
				</Table.Cell>
				<Table.Cell>
					{mode}
				</Table.Cell>
			</Table.Row>
		{/each}
	</Table.Body>
</Table.Root>

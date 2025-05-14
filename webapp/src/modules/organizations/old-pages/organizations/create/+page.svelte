<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { CollectionForm } from '@/collections-components';
	import { m } from '@/i18n';
	import { PageContent, PageCard, PageTop } from '@/components/layout';
	import { ArrowLeft } from 'lucide-svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import T from '@/components/ui-custom/t.svelte';
</script>

<PageTop>
	<Button href="/my/organizations" variant="outline" size="sm">
		<Icon src={ArrowLeft} mr></Icon>
		{m.Back_to_my_organizations()}
	</Button>
	<T tag="h4">{m.Create_an_organization()}</T>
</PageTop>

<PageContent>
	<PageCard>
		<!-- If needed, use organization_info collection for info -->
		<CollectionForm
			collection="organizations"
			fieldsOptions={{
				labels: {
					name: m.Organization_name()
				}
			}}
			onSuccess={(record) => {
				window.location.assign(`/my/organizations/${record.id}`);
			}}
		>
			{#snippet submitButtonContent()}
				{m.Create_organization()}
			{/snippet}
		</CollectionForm>
	</PageCard>
</PageContent>

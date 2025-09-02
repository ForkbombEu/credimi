<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ArrowLeft } from 'lucide-svelte';

	import { CollectionForm } from '@/collections-components';
	import { PageContent, PageCard, PageTop } from '@/components/layout';
	import Button from '@/components/ui-custom/button.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';
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
		<CollectionForm
			collection="organizations"
			fieldsOptions={{
				labels: {
					name: m.Organization_name(),
					description: m.Short_description(),
					logo: m.Avatar()
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

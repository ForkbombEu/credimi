<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ArrowUpRight, CheckIcon, CopyIcon, InfoIcon, XIcon } from '@lucide/svelte';

	import type { FieldSnippetOptions } from '@/collections-components/form/collectionFormTypes';

	import { CollectionForm } from '@/collections-components/index.js';
	import Alert from '@/components/ui-custom/alert.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import CopyButton from '@/components/ui-custom/copyButton.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import Tooltip from '@/components/ui-custom/tooltip.svelte';
	import { SubmitButton } from '@/forms';
	import { SwitchField } from '@/forms/fields';
	import { m } from '@/i18n/index.js';
	import { pb } from '@/pocketbase';

	import { setDashboardNavbar } from '../+layout@.svelte';

	//

	let { data } = $props();
	let { organization, isOrganizationNotEdited } = $derived(data);

	const hasPublishedRequest = $derived(
		pb.send<{ has_published: boolean }>(
			`/api/organizations/${organization.canonified_name}/has-published`,
			{}
		)
	);

	setDashboardNavbar({
		title: m.Organization(),
		right: navbarRight
	});

	//

	let updateLocked = $state(true);
</script>

{#snippet navbarRight()}
	<div class="flex items-center gap-2">
		<CopyButton textToCopy={organization.canonified_name} icon={CopyIcon}></CopyButton>
		<Button variant="outline" href="/organizations/{organization.canonified_name}">
			{m.Page_preview()}
			<ArrowUpRight />
		</Button>
	</div>
{/snippet}

{#snippet publishedField({ form }: FieldSnippetOptions<'organizations'>)}
	<div class="flex items-center justify-end">
		<div
			class="flex items-center justify-center gap-2 rounded-lg border border-slate-200 bg-slate-50/50 p-2 px-4 pt-4"
		>
			{#await hasPublishedRequest}
				<SwitchField
					{form}
					name="published"
					options={{ label: m.Publish_to_hub(), disabled: true }}
				/>
			{:then result}
				<SwitchField
					{form}
					name="published"
					options={{ label: m.Publish_to_hub(), disabled: result.has_published }}
				/>
				{#if result.has_published}
					<Tooltip>
						{#snippet child({ props })}
							<IconButton icon={InfoIcon} {...props} variant="ghost" size="sm" />
						{/snippet}
						{#snippet content()}
							<T class="text-xs text-muted-foreground">
								{m.organization_published_locked_info()}
							</T>
						{/snippet}
					</Tooltip>
				{/if}
			{:catch}
				<SwitchField {form} name="published" options={{ label: m.Publish_to_hub() }} />
			{/await}
		</div>
	</div>
{/snippet}

{#if isOrganizationNotEdited}
	<Alert variant="info" icon={InfoIcon}>
		<T>
			{m.Edit_your_organization_information_to_better_represent_your_services_and_products_on_the_hub()}
		</T>
	</Alert>
{/if}

{#key organization}
	<CollectionForm
		collection="organizations"
		initialData={organization}
		recordId={organization.id}
		onSuccess={(org) => {
			organization = org;
		}}
		fieldsOptions={{
			exclude: ['canonified_name', 'max_pipelines_in_queue'],
			order: ['published'],
			snippets: { published: publishedField }
		}}
	>
		{#snippet submitButton()}
			<!-- Hack to hide default button -->
		{/snippet}

		{#snippet children({ formState, form })}
			{@const isNameChanged = formState.current.name !== organization?.name}
			{@const isSubmitDisabled = isNameChanged && updateLocked}

			{#if isNameChanged}
				<Alert variant="warning" class="space-y-2">
					<T class="font-bold">{m.Warning()}:</T>
					<T>{m.Change_Organization_Name()}</T>
					<T>
						{m.Rename_organization_warning()}
					</T>
					<div class="flex justify-end gap-2">
						<Button variant="outline" onclick={() => (updateLocked = false)}>
							<CheckIcon />
							{m.Continue()}
						</Button>
						<Button variant="outline" onclick={() => form.reset()}>
							<XIcon />
							{m.Cancel()}
						</Button>
					</div>
				</Alert>
			{/if}

			<div class="flex justify-end">
				<SubmitButton disabled={isSubmitDisabled}>
					{m.Update_organization_page()}
				</SubmitButton>
			</div>
		{/snippet}
	</CollectionForm>
{/key}

<!-- 
{#if !showOrganizationForm}
	<div class="mb-6 flex items-center justify-between">
		<T tag="h3">{m.Page_preview()}</T>
		<Button onclick={() => (showOrganizationForm = true)}>
			<Pencil />
			{m.Edit_organization_info()}
		</Button>
	</div>
	<PageCard contentClass="!p-2">
		<div class="overflow-hidden rounded-lg border">
			<OrganizationPageDemo organization={organization!} {hubItems} isPreview />
		</div>
	</PageCard>
{:else}
	<div class="mb-6 flex items-center justify-between">
		<T tag="h3">{m.Update_your_organization_page()}</T>

		<Button onclick={() => (showOrganizationForm = false)}>
			<Undo />
			{m.Back()}
		</Button>
	</div>

	<PageCard>
		<CollectionForm
			collection="organizations"
			beforeSubmit={handleBeforeSubmit}
			onSuccess={() => {
				invalidateAll();
				showOrganizationForm = false;
			}}
			initialData={organization}
			recordId={organization?.id}
			fieldsOptions={{
				exclude: ['canonified_name']
			}}
		>
			{#snippet submitButtonContent()}
				{m.Update_organization_page()}
			{/snippet}
		</CollectionForm>
	</PageCard>
{/if}
-->

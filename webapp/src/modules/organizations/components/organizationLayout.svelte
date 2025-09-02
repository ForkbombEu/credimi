<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { OrganizationsResponse } from '@/pocketbase/types';

	import { PageTop, PageContent } from '@/components/layout';
	import T from '@/components/ui-custom/t.svelte';

	import OrganizationBreadcrumbs from './organizationBreadcrumbs.svelte';
	import OrganizationTabs from './organizationTabs.svelte';

	interface Props {
		org: OrganizationsResponse;
		children?: import('svelte').Snippet;
	}

	let { org, children }: Props = $props();
</script>

<PageTop>
	<OrganizationBreadcrumbs />
	<T tag="h4">{org.name}</T>
	{#snippet bottom()}
		<OrganizationTabs organizationId={org.id} />
	{/snippet}
</PageTop>

<PageContent>
	{@render children?.()}
</PageContent>

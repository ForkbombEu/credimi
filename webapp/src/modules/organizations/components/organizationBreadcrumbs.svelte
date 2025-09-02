<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { BreadcrumbsOptions } from '@/components/ui-custom/breadcrumbs.svelte';

	import Breadcrumbs from '@/components/ui-custom/breadcrumbs.svelte';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	//

	const breadcrumbsOptions: BreadcrumbsOptions = {
		renamers: {
			'[id]': getOrganizationNameById,
			organizations: () => m.organizations(),
			my: () => m.My()
		},
		exclude: ['[[lang]]']
	};

	async function getOrganizationNameById(id: string): Promise<string> {
		const organization = await pb.collection('organizations').getOne(id);
		return organization.name;
	}
</script>

<Breadcrumbs options={breadcrumbsOptions} />

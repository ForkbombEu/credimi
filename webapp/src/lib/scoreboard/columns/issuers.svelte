<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import { entities } from '$lib/global';

	import { renderComponent } from '@/components/ui/data-table';
	import { m } from '@/i18n';

	import * as Column from '../column';
	import * as EntityDisplay from '../entity-display';
	import EntityHeader from './headers/entity-header.svelte';

	export const column = Column.define({
		fn: (row) => {
			const issuers = row.expand.issuers ?? [];
			const credentials = row.expand.credentials ?? [];

			return issuers.map((issuer) => {
				const children = credentials
					.filter((credential) => credential.credential_issuer === issuer.id)
					.map((credential) => {
						const entityItem = EntityDisplay.fromPocketbaseEntity(credential);
						return {
							label: entityItem.name,
							href: entityItem.href,
							avatar: entityItem.avatar
						};
					});

				return {
					...EntityDisplay.fromPocketbaseEntity(issuer, entities.credential_issuers),
					children: children.length > 0 ? children : undefined
				};
			});
		},
		id: 'issuers',
		header: renderComponent(EntityHeader, {
			label: m.Issuance(),
			align: 'right'
		}),
		sortField: 'issuers.name',
		manualPillPositioning: true
	});
</script>

<script lang="ts">
	let { value }: Column.Props<typeof column> = $props();
</script>

<EntityDisplay.List items={value} layout="logos" />

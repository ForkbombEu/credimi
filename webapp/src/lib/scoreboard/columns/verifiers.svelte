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
			const verifiers = row.expand.verifiers ?? [];
			const useCaseVerifications = row.expand.use_case_verifications ?? [];

			return verifiers.map((verifier) => {
				const children = useCaseVerifications
					.filter((verification) => verification.verifier === verifier.id)
					.map((verification) => {
						const entityItem = EntityDisplay.fromPocketbaseEntity(verification);
						return {
							label: entityItem.name,
							href: entityItem.href,
							avatar: entityItem.avatar
						};
					});

				return {
					...EntityDisplay.fromPocketbaseEntity(verifier, entities.verifiers),
					children: children.length > 0 ? children : undefined
				};
			});
		},
		id: 'verifiers',
		header: renderComponent(EntityHeader, {
			label: m.Presentations()
		}),
		sortField: 'verifiers.name',
		manualPillPositioning: true
	});
</script>

<script lang="ts">
	let { value }: Column.Props<typeof column> = $props();
</script>

<EntityDisplay.List items={value} layout="logos" />

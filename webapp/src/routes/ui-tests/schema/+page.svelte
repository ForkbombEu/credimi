<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import CodeDisplay from '$lib/layout/codeDisplay.svelte';
	import z from 'zod';

	import type { CollectionFormData, Data } from '@/pocketbase/types';

	import { pb } from '@/pocketbase';
	import { createCollectionZodSchema } from '@/pocketbase/zod-schema';
	import { createDummyFile } from '@/utils/other';

	const x = z
		.object({
			x: z.string(),
			y: z.record(z.unknown())
		})
		.extend({
			y: z.object({
				u: z.number()
			})
		});

	const res = pb.collection('z_test_collection').getFullList();

	async function routine() {
		const data: Data<CollectionFormData['z_test_collection']> = {
			file_field: createDummyFile(),
			richtext_field: 'AO',
			text_field: 'Miao',
			relation_multi_field: ['sqynj66ubxkl32s', '2yqpdt4p0h3o7s9'],
			relation_field: 'sqynj66ubxkl32s',
			number_field: 21,
			json_field: { ao: 4 }
		};
		const schema = createCollectionZodSchema('z_test_collection');
		const parsedData = schema.safeParse(data);
		if (parsedData.success == true)
			await pb.collection('z_test_collection').create(parsedData.data);
	}

	routine();
</script>

{#await res then x}
	<CodeDisplay content={JSON.stringify(x, null, 2)} language="json" />
{/await}

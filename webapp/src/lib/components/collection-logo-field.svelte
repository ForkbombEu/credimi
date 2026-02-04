<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { z } from 'zod/v3';

	import { getCollectionFormContext } from '@/collections-components/form/context';
	import { LogoField } from '@/forms/fields';
	import { pb } from '@/pocketbase';

	//

	const context = getCollectionFormContext();
	const { form, initialData, collectionName, recordId } = $derived(context());

	const preview = $derived.by(() => {
		if (!initialData) return undefined;
		try {
			const { logo } = z.object({ logo: z.string().optional() }).parse(initialData);
			if (!logo) return undefined;
			return pb.files.getURL({ collectionName, id: recordId }, logo);
		} catch (e) {
			console.error(e);
			return undefined;
		}
	});
</script>

<LogoField {form} name="logo" initialPreviewUrl={preview} />

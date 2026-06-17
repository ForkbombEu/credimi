<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { fetchRecordDeeplink, type DeeplinkRecord } from '$lib/utils';

	import { m } from '@/i18n';
	import { QrCode } from '@/qr';
	import { getExceptionMessage } from '@/utils/errors';

	import PageSection from './_utils/page-section.svelte';
	import { sections } from './_utils/sections';

	//

	type Props = {
		record: DeeplinkRecord;
	};

	let { record }: Props = $props();

	//

	let isLoading = $state(true);
	let deeplink = $state<string>();
	let error = $state<string>();

	$effect(() => {
		loadDeeplink(record);
	});

	async function loadDeeplink(rec: DeeplinkRecord) {
		isLoading = true;
		error = undefined;
		deeplink = undefined;

		try {
			const result = await fetchRecordDeeplink(rec);
			deeplink = result.deeplink;
		} catch (err) {
			console.error('Failed to fetch record deeplink:', err);
			error = getExceptionMessage(err);
		} finally {
			isLoading = false;
		}
	}
</script>

<PageSection indexItem={sections.qr_code} class="flex flex-col items-stretch space-y-0">
	<QrCode
		src={deeplink}
		{isLoading}
		{error}
		loadingText={m.Processing_YAML_configuration()}
		placeholder={m.No_deeplink_available()}
		showLink
		clickable
	/>
</PageSection>

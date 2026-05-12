<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { generateDeeplinkFromYaml } from '$lib/utils';

	import { m } from '@/i18n';
	import { QrCode } from '@/qr';
	import { getExceptionMessage } from '@/utils/errors';

	import PageSection from './_utils/page-section.svelte';
	import { sections } from './_utils/sections';

	//

	type Props = {
		yaml?: string;
		deeplink?: string;
	};

	let { yaml, deeplink }: Props = $props();

	//

	let isLoading = $derived.by(() => {
		if (deeplink && !yaml) return false;
		else if (yaml) return true;
		else return false;
	});

	let yamlDeeplink = $state<string>();
	let yamlError = $state<string>();

	$effect(() => {
		generateYamlDeeplink();
	});

	async function generateYamlDeeplink() {
		if (!yaml) return;
		try {
			const result = await generateDeeplinkFromYaml(yaml);
			yamlDeeplink = result.deeplink;
		} catch (error) {
			console.error('Failed to process YAML for credential offer:', error);
			yamlError = getExceptionMessage(error);
		} finally {
			isLoading = false;
		}
	}

	const qrLink = $derived.by(() => {
		if (yaml) {
			if (isLoading) return undefined;
			else if (yamlDeeplink) return yamlDeeplink;
			else return undefined;
		} else if (deeplink) {
			return deeplink;
		} else {
			return undefined;
		}
	});
</script>

<PageSection indexItem={sections.qr_code} class="flex flex-col items-stretch space-y-0">
	<QrCode
		src={qrLink}
		{isLoading}
		error={yamlError}
		loadingText={m.Processing_YAML_configuration()}
		placeholder={m.No_deeplink_available()}
		showLink={true}
	/>
</PageSection>

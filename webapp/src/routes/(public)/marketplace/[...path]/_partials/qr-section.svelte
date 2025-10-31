<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { generateDeeplinkFromYaml } from '$lib/utils';

	import { m } from '@/i18n';
	import QrStateful from '@/qr/qr-stateful.svelte';
	import { getExceptionMessage } from '@/utils/errors';

	import PageSection from './_utils/page-section.svelte';
	import { sections } from './_utils/sections';

	//

	type Props = {
		yaml?: string;
		deeplink?: string;
	};

	let { yaml, deeplink }: Props = $props();
	console.log('yaml', yaml, 'deeplink', deeplink);

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
	<div class="space-y-4">
		<QrStateful
			src={qrLink}
			{isLoading}
			error={yamlError}
			loadingText={m.Processing_YAML_configuration()}
			placeholder={m.No_deeplink_available()}
		/>
		{#if qrLink}
			<div class="w-60 break-all text-xs">
				<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
				<a href={qrLink} target="_self">{qrLink}</a>
			</div>
		{/if}
	</div>
</PageSection>

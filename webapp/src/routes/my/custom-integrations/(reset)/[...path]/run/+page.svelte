<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import FocusPageLayout from '$lib/layout/focus-page-layout.svelte';
	import PageCardSection from '$lib/layout/page-card-section.svelte';

	import { m } from '@/i18n';

	import { CustomCheckConfigEditor, CustomCheckConfigEditorComponent } from './_components';

	//

	let { data } = $props();

	const editor = new CustomCheckConfigEditor({ customCheck: data.record });
</script>

<FocusPageLayout
	title={data.record.name}
	description={data.record.description || m.Run_custom_integration()}
	backButton={{ title: m.Back(), href: '/my/custom-integrations' }}
>
	<PageCardSection
		title={m.Configure_integration()}
		description={m.Configure_integration_description()}
	>
		<CustomCheckConfigEditorComponent {editor} />
	</PageCardSection>
</FocusPageLayout>

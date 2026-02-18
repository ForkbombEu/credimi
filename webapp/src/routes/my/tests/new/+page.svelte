<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import FocusPageLayout from '$lib/layout/focus-page-layout.svelte';
	import { StartChecksFormComponent } from '$start-checks-form';

	import { m } from '@/i18n';

	//

	let { data } = $props();
</script>

<!--  -->

<FocusPageLayout
	title={m.Start_a_new_conformance_check()}
	description={m.Start_a_new_conformance_check_description()}
	backButton={{ href: '/my', title: m.Back_to_dashboard() }}
>
	<StartChecksFormComponent
		standardsWithTestSuites={data.standardsAndTestSuites}
	/>
</FocusPageLayout>

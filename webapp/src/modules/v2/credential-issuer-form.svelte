<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import ImportIssuerForm from './import-issuer-form.svelte';

	type Props = {
		organizationId: string;
		onSuccess?: () => void;
	};

	let { onSuccess, organizationId }: Props = $props();
</script>

<ImportIssuerForm {onSuccess} {organizationId} />

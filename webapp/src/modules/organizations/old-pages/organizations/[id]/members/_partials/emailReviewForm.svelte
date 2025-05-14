<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { m } from '@/i18n';
	import { Input } from '@/components/ui/input';
	import { Label } from '@/components/ui/label';
	import { Array as A } from 'effect';
	import IconButton from '@/components/ui-custom/iconButton.svelte';

	interface Props {
		emails?: string[];
	}

	let { emails = $bindable([]) }: Props = $props();

	let filterText: string | undefined = $state(undefined);

	function filterEmails(emails: string[], text: string | undefined) {
		if (!Boolean(text)) return emails;
		return emails.filter((e) => e.includes(text ?? ''));
	}

	function removeEmail(email: string) {
		emails = A.remove(emails, emails.indexOf(email));
	}
</script>

<div class="space-y-4">
	<p>{m.Review_the_email_list_before_sending()}</p>

	<div class="space-y-2">
		<Label>{m.Search_emails()} ({emails.length})</Label>
		<Input placeholder="mail@example.org" bind:value={filterText} />
	</div>

	<div class="h-[300px] divide-y overflow-scroll rounded-lg border">
		{#each filterEmails(emails, filterText) as email}
			<div class="flex items-center justify-between py-1 pl-4 pr-2">
				<p class="text-sm">{email}</p>
				<IconButton onclick={() => removeEmail(email)} size="sm" variant="ghost" />
			</div>
		{/each}
	</div>
</div>

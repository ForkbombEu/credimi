<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ArrowLeft, Mail, X } from '@lucide/svelte';

	import Button from '@/components/ui-custom/button.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	import EmailInputForm from './emailInputForm.svelte';
	import EmailReviewForm from './emailReviewForm.svelte';

	interface Props {
		//
		organizationId: string;
		onSuccess?: (emails: string[]) => void;
		onCancel?: any;
	}

	let { organizationId, onSuccess = () => {}, onCancel = () => {} }: Props = $props();

	let formState = $state<'input' | 'review'>('input');
	let emails: string[] = $state([]);

	function handleInputFormSuccess(inputEmails: string[]) {
		emails = inputEmails;
		formState = 'review';
	}

	function handleSuccess(emails: string[]) {
		pb.send('/organizations/invite', {
			method: 'POST',
			body: {
				organizationId,
				emails
			}
		});
		onSuccess(emails);
	}
</script>

{#if formState == 'input'}
	<EmailInputForm onSuccess={handleInputFormSuccess} />
{:else if formState == 'review'}
	<div>
		<EmailReviewForm bind:emails />

		<div class="flex items-center justify-between gap-4 pt-6">
			<Button variant="outline" onclick={() => (formState = 'input')}>
				<Icon src={ArrowLeft} size={16} mr />{m.Back()}
			</Button>
			<div class="flex items-center gap-2">
				<Button variant="outline" onclick={onCancel}>
					<Icon src={X} mr />
					{m.Cancel()}
				</Button>
				<Button onclick={() => handleSuccess(emails)}>
					<Icon src={Mail} mr />
					{m.Send_invites()}
				</Button>
			</div>
		</div>
	</div>
{/if}

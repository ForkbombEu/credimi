<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { EntityTag, type EntityData } from '$lib/global';
	import BackButton from '$lib/layout/back-button.svelte';
	import PageTop from '$lib/layout/pageTop.svelte';

	import type { Link } from '@/components/types';

	import A from '@/components/ui-custom/a.svelte';
	import Avatar from '@/components/ui-custom/avatar.svelte';
	import CopyButtonSmall from '@/components/ui-custom/copy-button-small.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';

	//

	type Props = {
		hideTopBorder: boolean;
		logo?: string;
		linkAboveTitle?: Link;
		title: string;
		textToCopy?: string;
		badge?: EntityData;
	};

	let { hideTopBorder, logo, linkAboveTitle, title, textToCopy, badge }: Props = $props();
</script>

<PageTop {hideTopBorder} contentClass="space-y-4!">
	<BackButton href="/marketplace" class="px-0!">
		{m.Back_to_marketplace()}
	</BackButton>

	<div class="flex items-center gap-6">
		{#if logo}
			<Avatar src={logo} class="size-32 rounded-md border" hideIfLoadingError />
		{/if}

		<div class="space-y-3">
			<div>
				<div class="space-y-1">
					{#if linkAboveTitle}
						<A class="block" {...linkAboveTitle} href={linkAboveTitle.href}>
							{linkAboveTitle.title}
						</A>
					{/if}
					<div class="flex items-center gap-2">
						<T tag="h1">{title}</T>
						{#if textToCopy}
							<CopyButtonSmall {textToCopy} />
						{/if}
					</div>
				</div>

				{#if badge}
					<div class="pt-4">
						<EntityTag data={badge} />
					</div>
				{/if}
			</div>
		</div>
	</div>
</PageTop>

<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script module lang="ts">
	type BaseRecord = {
		id: string;
		collectionId: string;
	};
</script>

<script lang="ts" generics="R extends BaseRecord">
	import type { ComponentProps } from 'svelte';

	import { Eye, EyeOff } from '@lucide/svelte';

	import SwitchWithIcons from '@/components/ui-custom/switch-with-icons.svelte';
	import { pb } from '@/pocketbase';

	//

	type SwitchProps = Omit<
		ComponentProps<typeof SwitchWithIcons>,
		'offIcon' | 'onIcon' | 'checked' | 'onCheckedChange'
	>;

	type BooleanKey = {
		[K in keyof R]: R[K] extends boolean ? K : never;
	}[keyof R];

	type Props = {
		record: R;
		onPublishedChange?: (props: { record: R; published: boolean }) => void;
		field: BooleanKey;
	} & SwitchProps;

	let { record, onPublishedChange, field: fieldKey, ...rest }: Props = $props();

	function handler(published: boolean) {
		if (onPublishedChange) {
			onPublishedChange({ record, published });
		} else {
			pb.collection(record.collectionId).update(record.id, { [fieldKey]: published });
		}
	}
</script>

<SwitchWithIcons
	offIcon={EyeOff}
	onIcon={Eye}
	checked={record[fieldKey] as boolean}
	onCheckedChange={handler}
	{...rest}
/>

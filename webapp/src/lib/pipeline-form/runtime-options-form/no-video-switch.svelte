<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { SuperForm } from 'sveltekit-superforms';

	import { Label } from '@/components/ui/label';
	import { Switch } from '@/components/ui/switch';
	import { m } from '@/i18n/index.js';

	import { readNoVideo, writeNoVideo } from './runtime-options-form.svelte.js';

	//

	type Props = { form: SuperForm<{ code: string }> };
	const { form }: Props = $props();
	const { form: formData } = form;

	const noVideo = $derived(readNoVideo($formData.code));
</script>

<div class="space-y-1">
	<div class="flex items-center gap-2">
		<Switch
			id="pipeline-no-video"
			checked={noVideo === true}
			disabled={noVideo === undefined}
			onCheckedChange={(enabled) => ($formData.code = writeNoVideo($formData.code, enabled))}
		/>
		<Label for="pipeline-no-video">{m.pipeline_no_video()}</Label>
	</div>
	<p class="text-sm text-muted-foreground">{m.pipeline_no_video_description()}</p>
</div>

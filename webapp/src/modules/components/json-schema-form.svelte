<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { BasicForm, createForm, ON_CHANGE, ON_INPUT, ON_BLUR } from '@sjsf/form';
	import { resolver } from '@sjsf/form/resolvers/basic';
	import { translation } from '@sjsf/form/translations/en';
	import { createFormValidator } from '@sjsf/ajv8-validator';
	import { preventPageReload } from '@sjsf/form/prevent-page-reload.svelte';
	import { setThemeContext, theme } from '@sjsf/shadcn-theme';
	import * as components from '@sjsf/shadcn-theme/default';
	import { nanoid } from 'nanoid';

	type OnUpdateData = {
		valid: boolean;
		data: unknown;
	};

	type Props = {
		schema: object;
		onUpdate?: (data: OnUpdateData) => void;
		options?: {
			hideSubmitButton?: boolean;
			hideTitle?: boolean;
		};
	};

	const { schema, options, onUpdate }: Props = $props();

	const form = createForm({
		idPrefix: nanoid(5),
		resolver,
		theme,
		validator: createFormValidator({}),
		schema,
		translation,
		fieldsValidationMode: ON_CHANGE | ON_INPUT | ON_BLUR,
		uiSchema: {
			'ui:options': {
				hideTitle: options?.hideTitle ?? false
			}
		}
	});
	preventPageReload(form);

	setThemeContext({ components });

	$effect(() => {
		onUpdate?.({
			valid: form.validate().size === 0,
			data: form.value
		});
	});
</script>

<BasicForm {form} class={{ "[&>button[type='submit']]:hidden": options?.hideSubmitButton }} />

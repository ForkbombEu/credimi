// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { createForm, ON_CHANGE, ON_INPUT, ON_BLUR } from '@sjsf/form';
import { resolver } from '@sjsf/form/resolvers/basic';
import { translation } from '@sjsf/form/translations/en';
import { createFormValidator } from '@sjsf/ajv8-validator';
import { preventPageReload } from '@sjsf/form/prevent-page-reload.svelte';
import { theme } from '@sjsf/shadcn-theme';
import { nanoid } from 'nanoid';

type OnUpdateData = {
	valid: boolean;
	data: unknown;
};

type CreateJsonSchemaFormOptions = {
	hideTitle?: boolean;
	preventPageReload?: boolean;
	onUpdate?: (data: OnUpdateData) => void;
	initialValue?: unknown;
};

export function createJsonSchemaForm(schema: object, options?: CreateJsonSchemaFormOptions) {
	const initialValue = options?.initialValue;
	const form = createForm({
		idPrefix: nanoid(5),
		resolver,
		initialValue,
		theme,
		validator: createFormValidator({}),
		schema,
		translation,
		fieldsValidationMode: ON_CHANGE | ON_INPUT | ON_BLUR,
		uiSchema: {
			'ui:options': {
				hideTitle: options?.hideTitle ?? false,
				form: { ['data-hide-submit']: true }
			}
		}
	});

	if (options?.preventPageReload) {
		preventPageReload(form);
	}

	const onUpdate = options?.onUpdate;
	if (onUpdate) {
		$effect(() => {
			onUpdate({
				valid: form.validate().size === 0,
				data: form.value
			});
		});
	}

	return form;
}

export type JsonSchemaForm = ReturnType<typeof createJsonSchemaForm>;

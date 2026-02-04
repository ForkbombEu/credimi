// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { createFormValidator } from '@sjsf/ajv8-validator';
import { createForm, ON_BLUR, ON_CHANGE, ON_INPUT } from '@sjsf/form';
import { createFormIdBuilder } from '@sjsf/form/id-builders/modern';
import { createFormMerger } from '@sjsf/form/mergers/modern';
import { preventPageReload } from '@sjsf/form/prevent-page-reload.svelte';
import { resolver } from '@sjsf/form/resolvers/basic';
import { translation } from '@sjsf/form/translations/en';
import { theme } from '@sjsf/shadcn-theme';

//

type CreateJsonSchemaFormOptions = {
	hideTitle?: boolean;
	preventPageReload?: boolean;
	onSubmit?: (data: unknown) => void;
	initialValue?: unknown;
};

export function createJsonSchemaForm(schema: object, options?: CreateJsonSchemaFormOptions) {
	const initialValue = options?.initialValue;
	const form = createForm({
		idBuilder: createFormIdBuilder(),
		merger: createFormMerger,
		resolver,
		onSubmit: options?.onSubmit,
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

	return form;
}

export type JsonSchemaForm = ReturnType<typeof createJsonSchemaForm>;

// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { BaseForm, type InitFormOptions } from '$pipeline-form/steps/types';

import Component from './json-parse-step-form.svelte';

//

export type JsonParseFormData = {
	rawJSON: string;
	struct_type: string;
};

export const JSON_PARSE_STRUCT_TYPES = ['map'] as const;

export class JsonParseStepForm extends BaseForm<JsonParseFormData, JsonParseStepForm> {
	readonly Component = Component;

	data = $state<JsonParseFormData>({
		rawJSON: '{}',
		struct_type: 'map'
	});

	constructor(opts?: InitFormOptions<JsonParseFormData>) {
		super(opts);
		if (opts?.initial) {
			this.data = { ...this.data, ...opts.initial };
		}
	}

	get isValid(): boolean {
		return this.data.rawJSON.trim() !== '' && this.data.struct_type.trim() !== '';
	}

	canSave() {
		return this.isValid;
	}

	getSubmitData() {
		return this.isValid ? this.data : undefined;
	}

	submit() {
		this.commit();
	}
}

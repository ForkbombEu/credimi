// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { Pipeline } from '$lib';
import { Debounced } from 'runed';

export class InlineManualEditor {
	yaml = $state('');
	readonly baselineYaml: string;

	private debouncedValidation: Debounced<ReturnType<typeof Pipeline.validateYaml>>;

	constructor(initialYaml: string) {
		this.yaml = initialYaml;
		this.baselineYaml = initialYaml;
		this.debouncedValidation = new Debounced(() => Pipeline.validateYaml(this.yaml), 400);
	}

	get validation() {
		return this.debouncedValidation.current;
	}

	get isDirty() {
		return this.yaml !== this.baselineYaml;
	}

	get isValid() {
		return this.debouncedValidation.current.ok;
	}

	async validateNow() {
		await this.debouncedValidation.updateImmediately();
		return this.debouncedValidation.current;
	}

	dispose() {
		this.debouncedValidation.cancel();
	}
}

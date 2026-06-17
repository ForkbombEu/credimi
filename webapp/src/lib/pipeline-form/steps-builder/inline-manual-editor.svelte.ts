// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { Pipeline } from '$lib';

import type { PipelineYamlValidation } from '$lib/pipeline/validate-yaml';

const VALIDATION_DEBOUNCE_MS = 400;

export class InlineManualEditor {
	readonly baselineYaml: string;

	#yaml = $state('');
	#validation = $state<PipelineYamlValidation>({ ok: true, value: '' });
	#debounceTimer: ReturnType<typeof setTimeout> | null = null;

	constructor(initialYaml: string) {
		this.baselineYaml = initialYaml;
		this.#yaml = initialYaml;
		this.#validation = Pipeline.validateYaml(initialYaml);
	}

	get yaml() {
		return this.#yaml;
	}

	set yaml(value: string) {
		if (this.#yaml === value) return;
		this.#yaml = value;
		this.scheduleValidation();
	}

	get validation() {
		return this.#validation;
	}

	get isDirty() {
		return this.#yaml !== this.baselineYaml;
	}

	get isValid() {
		return this.#validation.ok;
	}

	private scheduleValidation() {
		if (this.#debounceTimer) clearTimeout(this.#debounceTimer);
		this.#debounceTimer = setTimeout(() => {
			this.#debounceTimer = null;
			this.#validation = Pipeline.validateYaml(this.#yaml);
		}, VALIDATION_DEBOUNCE_MS);
	}

	async validateNow() {
		if (this.#debounceTimer) {
			clearTimeout(this.#debounceTimer);
			this.#debounceTimer = null;
		}
		this.#validation = Pipeline.validateYaml(this.#yaml);
		return this.#validation;
	}

	dispose() {
		if (this.#debounceTimer) {
			clearTimeout(this.#debounceTimer);
			this.#debounceTimer = null;
		}
	}
}

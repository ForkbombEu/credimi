// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { PipelineFinallyCondition } from '$lib/pipeline/types';

import { pipe, String } from 'effect';

function addNewlineBefore(token: string, all = true) {
	return replaceWith(token, (token) => `\n${token}`, all);
}

/** Insert a blank line before any `  - ` item at the steps list indent only. */
function addNewlineBeforeTopLevelStepItem() {
	const pattern = /(?<=\n)( {2}- )/g;
	return (yaml: string) => yaml.replace(pattern, '\n$1');
}

/** Insert a blank line before `  always:` / `  on_success:` / `  on_failure:`. */
function addNewlineBeforeFinallyCondition(condition: PipelineFinallyCondition) {
	const pattern = new RegExp(`(?<=\\n)( {2}${condition}:)`, 'g');
	return (yaml: string) => yaml.replace(pattern, '\n$1');
}

/** Insert a blank line before any `    - ` item under finally condition lists. */
function addNewlineBeforeFinallyStepItem() {
	const pattern = /(?<=\n)( {4}- )/g;
	return (yaml: string) => yaml.replace(pattern, '\n$1');
}

function replaceWith(token: string, transform: (token: string) => string, all = true) {
	if (all) {
		return String.replaceAll(token, transform(token));
	} else {
		return String.replace(token, transform(token));
	}
}

/** Pretty-print spacing for stringified pipeline YAML (sections, steps, finally). */
export function formatPipelineYamlDocument(yaml: string): string {
	return pipe(
		yaml,
		// Adding spaces between top-level sections
		addNewlineBefore('runtime:'),
		addNewlineBefore('steps:'),
		addNewlineBefore('finally:'),
		// Blank line between finally condition buckets (always / on_success / on_failure)
		addNewlineBeforeFinallyCondition('always'),
		addNewlineBeforeFinallyCondition('on_success'),
		addNewlineBeforeFinallyCondition('on_failure'),
		// Blank line before any top-level step list item (exactly two spaces).
		// Matches `- id:` / `- use:` / `- continue_on_error:` alike; four-space
		// finally items are handled separately below.
		addNewlineBeforeTopLevelStepItem(),
		// Blank line before nested finally step items (exactly four spaces).
		addNewlineBeforeFinallyStepItem(),
		// Correcting first item newline under steps / finally / condition buckets
		replaceWith('steps:\n\n', (t) => t.replace('\n\n', '\n'), false),
		replaceWith('finally:\n\n', (t) => t.replace('\n\n', '\n'), false),
		replaceWith('  always:\n\n', (t) => t.replace('\n\n', '\n'), false),
		replaceWith('  on_success:\n\n', (t) => t.replace('\n\n', '\n'), false),
		replaceWith('  on_failure:\n\n', (t) => t.replace('\n\n', '\n'), false)
	);
}

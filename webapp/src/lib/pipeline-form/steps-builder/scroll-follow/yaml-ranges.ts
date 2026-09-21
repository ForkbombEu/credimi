// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

export type CardSection = 'steps' | 'follow-ups';

export type YamlCardRange = {
	section: CardSection;
	index: number;
	/** Inclusive 0-based line index */
	startLine: number;
	/** Inclusive 0-based line index */
	endLine: number;
};

const TOP_LEVEL_STEP = /^ {2}- (?:id|use):/;
const FINALLY_STEP = /^ {4}- (?:id|use):/;
const STEPS_HEADER = /^steps:\s*$/;
const FINALLY_HEADER = /^finally:\s*$/;
const TOP_LEVEL_KEY = /^[A-Za-z_][\w-]*:/;

/**
 * Map Pipeline Composer YAML preview text to card ranges by section + index.
 * Top-level steps use two-space list items; Follow-ups under `finally` use four-space items
 * (document order across condition buckets).
 */
export function mapYamlCardRanges(yaml: string): YamlCardRange[] {
	const lines = yaml.split('\n');
	const ranges: YamlCardRange[] = [];

	const stepsStart = lines.findIndex((line) => STEPS_HEADER.test(line));
	if (stepsStart === -1) return ranges;

	let i = stepsStart + 1;
	let stepIndex = 0;
	let currentStart: number | null = null;

	const flushStep = (endExclusive: number) => {
		if (currentStart === null) return;
		let end = endExclusive - 1;
		while (end > currentStart && lines[end].trim() === '') end -= 1;
		ranges.push({
			section: 'steps',
			index: stepIndex,
			startLine: currentStart,
			endLine: end
		});
		stepIndex += 1;
		currentStart = null;
	};

	while (i < lines.length) {
		const line = lines[i] ?? '';
		if (FINALLY_HEADER.test(line) || (TOP_LEVEL_KEY.test(line) && !line.startsWith(' '))) {
			flushStep(i);
			break;
		}
		if (TOP_LEVEL_STEP.test(line)) {
			flushStep(i);
			currentStart = i;
		}
		i += 1;
	}
	flushStep(i);

	const finallyStart = lines.findIndex((line) => FINALLY_HEADER.test(line));
	if (finallyStart === -1) return ranges;

	i = finallyStart + 1;
	let followUpIndex = 0;
	currentStart = null;

	const flushFollowUp = (endExclusive: number) => {
		if (currentStart === null) return;
		let end = endExclusive - 1;
		while (end > currentStart && lines[end].trim() === '') end -= 1;
		ranges.push({
			section: 'follow-ups',
			index: followUpIndex,
			startLine: currentStart,
			endLine: end
		});
		followUpIndex += 1;
		currentStart = null;
	};

	while (i < lines.length) {
		const line = lines[i] ?? '';
		if (TOP_LEVEL_KEY.test(line) && !line.startsWith(' ')) {
			flushFollowUp(i);
			break;
		}
		if (FINALLY_STEP.test(line)) {
			flushFollowUp(i);
			currentStart = i;
		}
		i += 1;
	}
	flushFollowUp(i);

	return ranges;
}

export function findRangeForUnit(
	ranges: YamlCardRange[],
	section: CardSection,
	index: number
): YamlCardRange | undefined {
	return ranges.find((r) => r.section === section && r.index === index);
}

export function findUnitAtLine(ranges: YamlCardRange[], line: number): YamlCardRange | undefined {
	return ranges.find((r) => line >= r.startLine && line <= r.endLine);
}

/** First step start line, or null if none — used to clear wash in name/runtime header. */
export function firstStepStartLine(ranges: YamlCardRange[]): number | null {
	const first = ranges.find((r) => r.section === 'steps');
	return first ? first.startLine : null;
}

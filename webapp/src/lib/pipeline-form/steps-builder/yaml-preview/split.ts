// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { mapYamlCardRanges, type YamlCardRange } from './ranges.js';

export type YamlPreviewStepBlock = {
	index: number;
	text: string;
};

export type YamlPreviewFollowUpBlock = {
	index: number;
	text: string;
};

/**
 * Document fragments for the virtual YAML preview pane.
 * Full `createPipelineYaml` remains SoT for copy / manual edit — these slices are display-only.
 */
export type YamlPreviewParts = {
	/** Everything before the first top-level step item (`name`, `runtime`, `steps:`). */
	header: string;
	/** One YAML snippet per top-level step (index-aligned with cards). */
	steps: YamlPreviewStepBlock[];
	/**
	 * Trailing blank lines after each step body (exact suffix, e.g. `'\n'`).
	 * Kept outside the interactive/selected block so the orange ring does not
	 * include the inter-step YAML gap. `betweenSteps[i]` sits after `steps[i]`.
	 * Length is `steps.length` (trailing may be '').
	 */
	betweenSteps: string[];
	/**
	 * Text between the last step and the first follow-up item (typically `finally:` /
	 * condition headers). Empty when there is no `finally` section.
	 */
	followUpsPreamble: string;
	/** One YAML snippet per finally step (document order; index-aligned with follow-up cards). */
	followUps: YamlPreviewFollowUpBlock[];
	/**
	 * Intervening text between follow-up items (e.g. blank lines / next condition header).
	 * `betweenFollowUps[i]` sits after `followUps[i]`. Length is `followUps.length` (trailing may be '').
	 */
	betweenFollowUps: string[];
};

const EMPTY: YamlPreviewParts = {
	header: '',
	steps: [],
	betweenSteps: [],
	followUpsPreamble: '',
	followUps: [],
	betweenFollowUps: []
};

/** `  always:` / `  on_success:` style finally condition headers. */
const FINALLY_CONDITION = /^ {2}[A-Za-z_][\w-]*:\s*$/;

function sliceExclusive(lines: string[], start: number, endExclusive: number): string {
	if (start >= endExclusive || start < 0) return '';
	return lines.slice(start, endExclusive).join('\n');
}

/**
 * Pull trailing blank lines off a step body so they can sit in `betweenSteps`
 * (outside the selected Shiki block). Returns the exact original suffix so rejoin
 * stays lossless when concatenated onto `body` before chunk-joining.
 */
function peelTrailingBlankLines(text: string): { body: string; trailing: string } {
	if (!text) return { body: '', trailing: '' };
	const lines = text.split('\n');
	let end = lines.length - 1;
	while (end >= 0 && lines[end]!.trim() === '') {
		end -= 1;
	}
	if (end === lines.length - 1) return { body: text, trailing: '' };
	const body = lines.slice(0, end + 1).join('\n');
	return { body, trailing: text.slice(body.length) };
}

/**
 * Pull trailing blank lines + finally-condition headers off a follow-up body so they
 * can sit in `betweenFollowUps` (mapYamlCardRanges folds them into the previous unit).
 */
function peelTrailingFinallyChrome(text: string): { body: string; trailing: string } {
	if (!text) return { body: '', trailing: '' };
	const lines = text.split('\n');
	let end = lines.length - 1;
	while (end >= 0 && (lines[end]!.trim() === '' || FINALLY_CONDITION.test(lines[end]!))) {
		end -= 1;
	}
	if (end === lines.length - 1) return { body: text, trailing: '' };
	return {
		body: lines.slice(0, end + 1).join('\n'),
		trailing: lines.slice(end + 1).join('\n')
	};
}

/** Concatenate preview fragments back into a full document (for tests / sanity checks). */
export function joinPipelineYamlPreview(parts: YamlPreviewParts): string {
	const chunks: string[] = [];
	if (parts.header) chunks.push(parts.header);
	for (let i = 0; i < parts.steps.length; i++) {
		const step = parts.steps[i]!;
		const between = parts.betweenSteps[i] ?? '';
		// `between` is an exact suffix (often `'\n'`); keep it on the same chunk so
		// the later `join('\n')` does not insert an extra separator.
		const piece = step.text + between;
		if (piece) chunks.push(piece);
	}
	if (parts.followUpsPreamble) chunks.push(parts.followUpsPreamble);
	for (let i = 0; i < parts.followUps.length; i++) {
		const fu = parts.followUps[i]!;
		if (fu.text) chunks.push(fu.text);
		const between = parts.betweenFollowUps[i] ?? '';
		if (between) chunks.push(between);
	}
	return chunks.join('\n');
}

/**
 * Split a full pipeline YAML preview string into header / per-step / follow-up blocks
 * using the same range map as the former line-based peer-follow.
 */
export function splitPipelineYamlPreview(yaml: string): YamlPreviewParts {
	if (!yaml) return EMPTY;

	const lines = yaml.split('\n');
	const ranges = mapYamlCardRanges(yaml);
	const stepRanges = ranges.filter((r): r is YamlCardRange => r.section === 'steps');
	const followUpRanges = ranges.filter((r): r is YamlCardRange => r.section === 'follow-ups');

	if (stepRanges.length === 0 && followUpRanges.length === 0) {
		return { ...EMPTY, header: yaml };
	}

	const firstStep = stepRanges[0];
	const header = firstStep
		? sliceExclusive(lines, 0, firstStep.startLine)
		: followUpRanges[0]
			? sliceExclusive(lines, 0, followUpRanges[0].startLine)
			: yaml;

	const lastStep = stepRanges[stepRanges.length - 1];
	const afterSteps = lastStep ? lastStep.endLine + 1 : firstStep ? firstStep.startLine : 0;

	// Each step slice runs until the next step starts. Trailing blank lines are
	// peeled into `betweenSteps` so selection/hover rings wrap only the step body
	// while the virtual row still keeps the true YAML gap.
	const steps: YamlPreviewStepBlock[] = [];
	const betweenSteps: string[] = [];
	for (let i = 0; i < stepRanges.length; i++) {
		const r = stepRanges[i]!;
		const endExclusive = stepRanges[i + 1]?.startLine ?? afterSteps;
		const raw = sliceExclusive(lines, r.startLine, endExclusive);
		const { body, trailing } = peelTrailingBlankLines(raw);
		steps.push({ index: r.index, text: body });
		betweenSteps.push(trailing);
	}

	if (followUpRanges.length === 0) {
		return {
			header,
			steps,
			betweenSteps,
			followUpsPreamble: sliceExclusive(lines, afterSteps, lines.length),
			followUps: [],
			betweenFollowUps: []
		};
	}

	const firstFollowUp = followUpRanges[0]!;
	const followUpsPreamble = sliceExclusive(lines, afterSteps, firstFollowUp.startLine);

	const followUps: YamlPreviewFollowUpBlock[] = [];
	const betweenFollowUps: string[] = [];

	for (let i = 0; i < followUpRanges.length; i++) {
		const current = followUpRanges[i]!;
		const next = followUpRanges[i + 1];
		const endExclusive = next ? next.startLine : lines.length;
		const raw = sliceExclusive(lines, current.startLine, endExclusive);
		const { body, trailing } = peelTrailingFinallyChrome(raw);
		followUps.push({ index: current.index, text: body });
		betweenFollowUps.push(trailing);
	}

	return { header, steps, betweenSteps, followUpsPreamble, followUps, betweenFollowUps };
}

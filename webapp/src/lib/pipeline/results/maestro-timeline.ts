// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/** One line of the `maestro-timeline.jsonl` file stored with a pipeline result. */
export type MaestroTimelineEntry = {
	/** Unix milliseconds. */
	t: number;
	kind: 'recording_started' | 'command' | 'flow' | 'notice' | 'screenshot';
	flow_id?: string;
	step_id?: string;
	index: number;
	parent: number;
	depth: number;
	description?: string;
	status?: string;
	message?: string;
};

export type MaestroTimeline = {
	/** Unix milliseconds of video time 0. */
	recordingStartedAt?: number;
	/** The row whose Maestro screenshot is the stored result screenshot. */
	screenshotRow?: { flowId: string; index: number };
	/** `command`, `flow` and `notice` state updates, sorted by `t`. */
	entries: MaestroTimelineEntry[];
};

export type MaestroRowView = {
	index: number;
	parent: number;
	depth: number;
	kind: MaestroTimelineEntry['kind'];
	description: string;
	status: string;
	message?: string;
	/** Unix milliseconds of the row's first state. */
	startedAt: number;
};

export type MaestroFlowView = {
	flowId: string;
	title: string;
	rows: MaestroRowView[];
};

const ENTRY_KINDS: Record<string, true> = {
	recording_started: true,
	command: true,
	flow: true,
	notice: true,
	screenshot: true
};

function optionalString(value: unknown): string | undefined {
	return typeof value === 'string' ? value : undefined;
}

function finiteNumber(value: unknown, fallback: number): number {
	return typeof value === 'number' && Number.isFinite(value) ? value : fallback;
}

function toEntry(value: unknown): MaestroTimelineEntry | undefined {
	if (typeof value !== 'object' || value === null) return undefined;
	const raw = value as Record<string, unknown>;
	if (typeof raw.t !== 'number' || !Number.isFinite(raw.t)) return undefined;
	if (typeof raw.kind !== 'string' || ENTRY_KINDS[raw.kind] !== true) return undefined;
	return {
		t: raw.t,
		kind: raw.kind as MaestroTimelineEntry['kind'],
		flow_id: optionalString(raw.flow_id),
		step_id: optionalString(raw.step_id),
		index: finiteNumber(raw.index, 0),
		parent: finiteNumber(raw.parent, 0),
		depth: finiteNumber(raw.depth, 0),
		description: optionalString(raw.description),
		status: optionalString(raw.status),
		message: optionalString(raw.message)
	};
}

export function parseMaestroTimeline(text: string): MaestroTimeline {
	const timeline: MaestroTimeline = { entries: [] };
	for (const line of text.split('\n')) {
		if (!line.trim()) continue;
		let entry: MaestroTimelineEntry | undefined;
		try {
			entry = toEntry(JSON.parse(line));
		} catch {
			continue;
		}
		if (!entry) continue;
		if (entry.kind === 'recording_started') {
			timeline.recordingStartedAt ??= entry.t;
		} else if (entry.kind === 'screenshot') {
			timeline.screenshotRow = { flowId: entry.flow_id ?? '', index: entry.index };
		} else {
			timeline.entries.push(entry);
		}
	}
	// Array.prototype.sort is stable, so updates with the same `t` keep their order.
	timeline.entries.sort((a, b) => a.t - b.t);
	return timeline;
}

//

type FlowState = {
	flowId: string;
	stepId?: string;
	rows: Map<number, MaestroRowView>;
};

const UNFINISHED_STATUSES: Record<string, true> = { RUNNING: true, PENDING: true };
const UNSUCCESSFUL_STATUSES: Record<string, true> = {
	FAILED: true,
	INTERRUPTED: true,
	RUNNING: true,
	PENDING: true
};

/** Replays the entries with `t <= at`; flows are returned in start order. */
function replay(timeline: MaestroTimeline, at: number): FlowState[] {
	const flows = new Map<string, FlowState>();
	for (const entry of timeline.entries) {
		if (entry.t > at) break;
		const flowId = entry.flow_id ?? '';
		let flow = flows.get(flowId);
		if (!flow) {
			flow = { flowId, rows: new Map() };
			flows.set(flowId, flow);
		}
		if (!flow.stepId && entry.step_id) flow.stepId = entry.step_id;
		const previous = flow.rows.get(entry.index);
		flow.rows.set(entry.index, {
			index: entry.index,
			parent: entry.parent,
			depth: entry.depth,
			kind: entry.kind,
			description: entry.description ?? '',
			status: entry.status ?? '',
			message: entry.message || undefined,
			startedAt: previous?.startedAt ?? entry.t
		});
	}
	const ordered = [...flows.values()];
	// A device runs one flow at a time: rows of earlier flows that still look
	// running were cut short. In the final state every such row was.
	const lastRunning = at === Infinity ? ordered.length : ordered.length - 1;
	for (const flow of ordered.slice(0, lastRunning)) {
		for (const row of flow.rows.values()) {
			if (UNFINISHED_STATUSES[row.status] === true) row.status = 'INTERRUPTED';
		}
	}
	return ordered;
}

function flowSucceeded(flow: FlowState): boolean {
	for (const row of flow.rows.values()) {
		if (row.kind === 'flow' || UNSUCCESSFUL_STATUSES[row.status] === true) return false;
	}
	return true;
}

const titlesCache = new WeakMap<MaestroTimeline, Map<string, string>>();

/**
 * Flow titles by flow id, as on the runner live page: a step that runs again
 * after its previous flow did not succeed is a numbered retry; a repeat after
 * a success is not.
 */
export function maestroFlowTitles(timeline: MaestroTimeline): Map<string, string> {
	const cached = titlesCache.get(timeline);
	if (cached) return cached;
	const titles = new Map<string, string>();
	const lastByStep = new Map<string, { flow: FlowState; retry: number }>();
	replay(timeline, Infinity).forEach((flow, i) => {
		if (!flow.stepId) {
			titles.set(flow.flowId, `Flow ${i + 1}:`);
			return;
		}
		const previous = lastByStep.get(flow.stepId);
		const retry = previous && !flowSucceeded(previous.flow) ? previous.retry + 1 : 0;
		lastByStep.set(flow.stepId, { flow, retry });
		titles.set(flow.flowId, `${flow.stepId}${retry ? ` (retry ${retry})` : ''}:`);
	});
	titlesCache.set(timeline, titles);
	return titles;
}

/** The box state at unix milliseconds `at`; `Infinity` gives the final state. */
export function maestroFlowsAt(timeline: MaestroTimeline, at: number): MaestroFlowView[] {
	const titles = maestroFlowTitles(timeline);
	return replay(timeline, at).map((flow) => ({
		flowId: flow.flowId,
		title: titles.get(flow.flowId) ?? '',
		rows: [...flow.rows.values()].sort((a, b) => a.index - b.index)
	}));
}

//

function vttTimestamp(seconds: number): string {
	const totalMs = Math.round(seconds * 1000);
	const ms = totalMs % 1000;
	const totalSeconds = Math.floor(totalMs / 1000);
	const s = totalSeconds % 60;
	const m = Math.floor(totalSeconds / 60) % 60;
	const h = Math.floor(totalSeconds / 3600);
	const pad = (n: number, width = 2) => String(n).padStart(width, '0');
	return `${pad(h)}:${pad(m)}:${pad(s)}.${pad(ms, 3)}`;
}

/** WebVTT subtitles with one cue per state update, or undefined without a recording start. */
export function buildMaestroVtt(timeline: MaestroTimeline): string | undefined {
	const startedAt = timeline.recordingStartedAt;
	if (startedAt === undefined) return undefined;
	const titles = maestroFlowTitles(timeline);
	const starts = timeline.entries.map((entry) => Math.max(0, (entry.t - startedAt) / 1000));
	const cues = timeline.entries.map((entry, i) => {
		const start = starts[i];
		let end = i + 1 < starts.length ? starts[i + 1] : start + 3;
		if (end <= start) end = start + 0.5;
		const title = titles.get(entry.flow_id ?? '') ?? '';
		let text = `${title} ${entry.description ?? ''}`;
		if (entry.kind === 'command' && entry.status) text += ` (${entry.status})`;
		// A blank line would end the cue early; `<`, `&` and `-->` are cue syntax.
		const cueText = text
			.replace(/\s*\r?\n\s*/g, ' ')
			.replaceAll('&', '&amp;')
			.replaceAll('<', '&lt;')
			.replaceAll('-->', '->');
		return `${vttTimestamp(start)} --> ${vttTimestamp(end)}\n${cueText}\n`;
	});
	return `WEBVTT\n\n${cues.join('\n')}`;
}

//

type ViewerResult = { video: string; screenshot: string; timeline?: string };

/** The result viewer page, or the raw file when the result has no timeline. */
export function buildPipelineResultViewerHref(
	mode: 'video' | 'screenshot',
	result: ViewerResult
): string {
	if (!result.timeline) return mode === 'video' ? result.video : result.screenshot;
	const params = [
		`mode=${mode}`,
		`video=${encodeURIComponent(result.video)}`,
		`screenshot=${encodeURIComponent(result.screenshot)}`,
		`timeline=${encodeURIComponent(result.timeline)}`
	];
	return `/pipeline-result?${params.join('&')}`;
}

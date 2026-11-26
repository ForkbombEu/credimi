// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { WorkflowStatus } from '$lib/workflows/types';

import { ClientResponseError } from 'pocketbase';
import { err, ok, Result } from 'true-myth/result';

import { pb } from '@/pocketbase';
import { getExceptionMessage } from '@/utils/errors';

//

/** A runner device that can host an authoring session (GET /api/action-author/devices). */
export type AuthorDevice = {
	path: string;
	name: string;
	runner_name: string;
	runner_owner: string;
	model: string;
	type: string;
	platform: string;
	is_online: boolean;
	agent_available: boolean;
	runs_ahead?: number;
};

/** A build the agent can install (GET /api/action-author/wallet-versions). */
export type BuildOption = {
	identifier: string;
	wallet_name: string;
	organization_name: string;
	version_tag: string;
	platforms: string[];
	source: string;
};

/** An action of the pending review, ordered by its candidate key. */
export type ReviewAction = {
	/** Candidate key of the action in its session: a1, a2, … */
	key: string;
	name: string;
	build: string;
	code: string;
	candidate_sha256: string;
	plan: Record<string, unknown> | null;
	/** Path of the action this one replaces in place, empty when none. */
	replaces: string;
	/** Why the previous Accept of this action did not save, empty when none. */
	error: string;
	/** The agent's note on the action (wallet findings, why the wallet cannot do the role). */
	note?: string;
	/** The agent's verdict on an FCAF action: passed, unsupported or finding; absent otherwise. */
	fcaf_status?: 'passed' | 'unsupported' | 'finding';
	/** An FCAF action's replays on the fixtures of its role's authoring set; absent otherwise. */
	fcaf_replays?: FcafReplay[];
	/** Candidate keys among the prerequisites of the action's plan. */
	starts_from: string[];
	/** Name of each key of starts_from: the review's action, else the one the session saved, else the key. */
	starts_from_names: Record<string, string>;
	/** Pipelines of the caller's organization or published that use the replaced action. */
	used_by: number;
};

/** The review waiting for the user's decisions, one action per tab. */
export type PendingReview = {
	tool_id: string;
	/** FCAF wallet profile saved with every FCAF action of the review; an accept binds it. */
	profile?: Record<string, unknown>;
	/** The agent's FCAF findings about the wallet itself, in English; empty when none. */
	warnings: string[];
	actions: ReviewAction[];
};

export type AuthoringMode = 'chat' | 'fcaf';

/** Where a role of an FCAF campaign stands. */
export type FcafRoleStatus =
	| 'pending'
	| 'working'
	| 'passed'
	| 'unsupported'
	| 'finding'
	| 'blocked'
	| 'skipped';

/** A role of an FCAF campaign, with the agent's note. */
export type FcafRoleProgress = { role: string; status: FcafRoleStatus; key: string; note: string };

/** The campaign of an FCAF session while its chat runs. */
export type FcafProgress = { current_role: string; roles: FcafRoleProgress[] };

/** The state of the caller organization's action for an FCAF role on a wallet. */
export type FcafCoverageStatus = 'missing' | 'current' | 'incomplete' | 'stale' | 'unmanaged';

export type FcafRoleCoverage = {
	role: string;
	summary: string;
	outcome: string;
	category: string;
	starts_from: string[];
	status: FcafCoverageStatus;
	action: string;
	role_sha256: string;
	/** The agent's verdict on the saved FCAF action, empty when not recorded or no action. */
	result: '' | 'passed' | 'unsupported' | 'finding';
	/** The agent's note on the saved action. */
	note: string;
	/** The fixtures of the role's authoring set an incomplete action was not replayed on. */
	missing_fixtures: string[];
	/** The saved action's replays on fixtures of its role's authoring set. */
	fcaf_replays: FcafReplay[];
	/** How the role's action did in the wallet's last FCAF run; null when that run did not include it, or no run. */
	run: FcafRoleRun | null;
	/**
	 * The role's verdict: the pipeline's when the last run covered the role's current or
	 * incomplete action, else the agent's recorded verdict (not verified yet); empty otherwise.
	 */
	verdict: '' | FcafPipelineVerdict | 'unsupported';
	/** Where the verdict comes from; empty without one. */
	verdict_source: '' | 'pipeline' | 'authoring';
};

/** The verdict the pipeline gives a role it covered. */
export type FcafPipelineVerdict = 'passed' | 'finding' | 'failing' | 'blocked';

/** How a role's action did in the wallet's last FCAF run. */
export type FcafRoleRun = {
	status: 'ok' | 'failing' | 'not_run';
	/** The role's verdict from the run: the action's failure, else its tests' results. */
	verdict: FcafPipelineVerdict | 'not_run';
	failed_steps: number;
	/** The Maestro command the action failed on, empty when it did not fail. */
	failed_command: string;
	/** The fixture (scenario#request) the action failed on, empty when it did not fail. */
	fixture_id: string;
	/** The role's failed tests (at most 20). */
	failed_tests: string[];
	/** How many of the role's tests passed, failed or were blocked. */
	tests_run: number;
	run_url: string;
};

/** The test counts of the wallet's last FCAF run; total is the sum of every counter. */
export type FcafRunSummary = {
	pass: number;
	fail: number;
	blocked: number;
	skipped: number;
	inconclusive: number;
	not_applicable: number;
	error: number;
	total: number;
	pipeline: string;
	created: string;
	/** How a run that ended without a report ended (empty otherwise): the lowercase Temporal status. */
	status: '' | FcafRunStatus;
};

/** The lowercase Temporal status of a finished FCAF run. */
export type FcafRunStatus =
	| 'completed'
	| 'failed'
	| 'canceled'
	| 'terminated'
	| 'timed_out'
	| 'continued_as_new';

/** FCAF relying-party coverage of a wallet (GET /api/action-author/fcaf-coverage). */
export type FcafCoverage = {
	suite: string;
	roles: FcafRoleCoverage[];
	warnings: string[];
	/** The counts of the wallet's last finished FCAF run, null when none. */
	run_summary: FcafRunSummary | null;
	/** The link of the wallet's last finished FCAF run, empty when none. */
	run_url: string;
};

/** A scenario an FCAF run left out, with the roles the wallet has no action for. */
export type FcafSkippedScenario = { scenario: string; missing_roles: string[] };

/** The FCAF validation pipeline queued on a device (POST /api/action-author/fcaf-pipeline). */
export type FcafPipelineRun = {
	pipeline: string;
	pipeline_url: string;
	ticket_id: string;
	device_id: string;
	/** Queue status of the ticket: queued, starting, running, … */
	status: string;
	/** Set only when the run already started. */
	workflow_id?: string;
	run_id?: string;
	run_url?: string;
	skipped: FcafSkippedScenario[];
	test_count: number;
};

/** An authoring session and its state (GET /api/action-author/sessions/{id}). */
export type Session = {
	session_id: string;
	run_id: string;
	chat_workflow_id: string;
	status: WorkflowStatus;
	start_time: string;
	close_time?: string;
	wallet_version_identifier: string;
	platform: string;
	device_id: string;
	device_name: string;
	runner_name: string;
	runner_owner: string;
	model: string;
	live_view: boolean;
	chat_ready: boolean;
	/** Whether the chat workflow still runs; false once it ended, even while the session releases the device. */
	chat_running: boolean;
	closes_at: string;
	pending_review: PendingReview | null;
	mode: AuthoringMode;
	/** The campaign of an FCAF session while its chat runs, null otherwise. */
	fcaf_progress: FcafProgress | null;
};

/** A row of GET /api/action-author/sessions?wallet=… */
export type SessionListItem = {
	session_id: string;
	status: WorkflowStatus;
	start_time: string;
	wallet_version_identifier: string;
	device_id: string;
	device_name: string;
	mode: AuthoringMode;
};

/** An item of a chat's transcript (GET /api/action-author/sessions/{id}/transcript). */
export type TranscriptEntry = {
	kind: 'user' | 'assistant' | 'tool_call' | 'review' | 'saved' | 'unavailable';
	/** RFC 3339 time of the history event. */
	time: string;
	text?: string;
	tool?: string;
	/** Pretty JSON arguments of a tool call, cut at 4000 characters. */
	arguments?: string;
	/** The user's decision on an action, for a review entry. */
	decision?: Decision;
	/** What the user asked to change, for a change decision. */
	comment?: string;
	name?: string;
	code?: string;
	replaced?: boolean;
};

export type StartedSession = { session_id: string; ticket_id: string; device_id: string };

export type ApiFailure = { status: number; reason: string; message: string };

/** The agent's verdict on an FCAF action, as the review showed it. */
export type FcafResult = { status: 'passed' | 'unsupported' | 'finding'; note: string };

/** How an FCAF action did on one fixture of its role's authoring set. */
export type FcafReplay = {
	fixture: string;
	class: string;
	status: 'passed' | 'finding' | 'action_failed';
};

export type Decision = 'accept' | 'refuse' | 'change';

/** The user's decision on one action of a review. */
export type ReviewDecisionItem = {
	key: string;
	name: string;
	candidate_sha256: string;
	decision: Decision;
	/** What to change, only for a change decision. */
	comment: string;
	plan: Record<string, unknown> | null;
	/** The FCAF profile the review showed, only for an accepted FCAF action. */
	profile?: Record<string, unknown>;
	/** The agent's verdict the review showed, only for an accepted FCAF action. */
	fcaf_result?: FcafResult;
	/** The replays the review showed, only for an accepted FCAF action. */
	fcaf_replays?: FcafReplay[];
};

/** POST /api/action-author/reviews: the decisions on every action of a review. */
export type ReviewRequest = {
	author_workflow_id: string;
	tool_id: string;
	decisions: ReviewDecisionItem[];
};

export type ReviewResponse = {
	tool_id: string;
	decisions: { key: string; decision: Decision }[];
};

//

const BASE = '/api/action-author';

function toFailure(e: unknown): ApiFailure {
	if (e instanceof ClientResponseError) {
		const response = e.response as
			| { message?: string; error?: { reason?: string } }
			| undefined;
		return {
			status: e.status,
			reason: response?.error?.reason ?? '',
			message: response?.message || e.message
		};
	}
	return { status: 0, reason: '', message: getExceptionMessage(e) };
}

async function call<T>(
	path: string,
	options: { method?: string; body?: unknown; keepalive?: boolean } = {}
): Promise<Result<T, ApiFailure>> {
	try {
		const res = await pb.send<T>(`${BASE}${path}`, {
			method: options.method ?? 'GET',
			body: options.body,
			keepalive: options.keepalive,
			requestKey: null
		});
		return ok(res);
	} catch (e) {
		return err(toFailure(e));
	}
}

export async function listDevices(): Promise<Result<AuthorDevice[], ApiFailure>> {
	const res = await call<{ devices: AuthorDevice[] | null }>('/devices');
	return res.map((r) => r.devices ?? []);
}

export async function listBuilds(): Promise<Result<BuildOption[], ApiFailure>> {
	const res = await call<{ wallet_versions: BuildOption[] | null }>('/wallet-versions');
	return res.map((r) => r.wallet_versions ?? []);
}

export function startSession(
	deviceId: string,
	walletVersionIdentifier: string,
	mode: AuthoringMode
): Promise<Result<StartedSession, ApiFailure>> {
	return call<StartedSession>('/sessions', {
		method: 'POST',
		body: { device_id: deviceId, wallet_version_identifier: walletVersionIdentifier, mode }
	});
}

export function getSession(id: string): Promise<Result<Session, ApiFailure>> {
	return call<Session>(`/sessions/${encodeURIComponent(id)}`);
}

export async function listSessions(
	walletPath: string
): Promise<Result<SessionListItem[], ApiFailure>> {
	const params = new URLSearchParams({ wallet: walletPath });
	const res = await call<{ sessions: SessionListItem[] | null }>(
		`/sessions?${params.toString()}`
	);
	return res.map((r) => r.sessions ?? []);
}

export function getFcafCoverage(walletPath: string): Promise<Result<FcafCoverage, ApiFailure>> {
	const params = new URLSearchParams({ wallet: walletPath });
	return call<FcafCoverage>(`/fcaf-coverage?${params.toString()}`);
}

/** Queues the wallet version's FCAF validation pipeline on an Android device; no scenarios runs every scenario. */
export function runFcafPipeline(
	walletVersionIdentifier: string,
	deviceId: string,
	scenarios?: string[]
): Promise<Result<FcafPipelineRun, ApiFailure>> {
	return call<FcafPipelineRun>('/fcaf-pipeline', {
		method: 'POST',
		body: {
			wallet_version_identifier: walletVersionIdentifier,
			device_id: deviceId,
			...(scenarios?.length ? { scenarios } : {})
		}
	});
}

export async function getTranscript(
	sessionId: string
): Promise<Result<TranscriptEntry[], ApiFailure>> {
	const res = await call<{ entries: TranscriptEntry[] }>(
		`/sessions/${encodeURIComponent(sessionId)}/transcript`
	);
	return res.map((r) => r.entries);
}

/** Whether the chat UI can replay the session's ended chat through its runner's agent. */
export async function getReplayable(sessionId: string): Promise<Result<boolean, ApiFailure>> {
	const res = await call<{ replayable: boolean }>(
		`/sessions/${encodeURIComponent(sessionId)}/replay`
	);
	return res.map((r) => r.replayable);
}

export function closeSession(
	id: string,
	options: { keepalive?: boolean } = {}
): Promise<Result<{ session_id: string }, ApiFailure>> {
	return call<{ session_id: string }>(`/sessions/${encodeURIComponent(id)}/close`, {
		method: 'POST',
		keepalive: options.keepalive
	});
}

/** An action author session ID: ActionAuthorSession-<runner id>-<ticket uuid>. */
export const SESSION_ID_PATTERN =
	/^ActionAuthorSession-[a-z0-9]{15}-([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})$/;

/** The ticket of a session ID (ActionAuthorSession-<runner id>-<ticket>). */
export function ticketOf(sessionId: string): string {
	const ticket = SESSION_ID_PATTERN.exec(sessionId)?.[1];
	if (ticket === undefined) throw new Error('not an action author session id');
	return ticket;
}

/** Ends the session or its queued ticket while the page unloads; fire and forget. */
export function endOnExit(sessionId: string, deviceId: string, queued: boolean): void {
	if (!queued) {
		void closeSession(sessionId, { keepalive: true });
		return;
	}
	const ticket = ticketOf(sessionId);
	pb.send(
		`/api/pipeline/queue/${encodeURIComponent(ticket)}?device_ids=${encodeURIComponent(deviceId)}`,
		{ method: 'DELETE', keepalive: true, requestKey: null }
	).catch(() => {});
}

export function sendReview(request: ReviewRequest): Promise<Result<ReviewResponse, ApiFailure>> {
	return call<ReviewResponse>('/reviews', { method: 'POST', body: request });
}

/**
 * Sends the chat's first message through the session's chat proxy. request_id makes the
 * harness's update ID msg-<session id>, so a repeated send never starts a second turn.
 */
export async function sendFirstMessage(
	sessionId: string,
	chatId: string,
	text: string
): Promise<Result<void, ApiFailure>> {
	try {
		const res = await fetch(`/action-author-ui/${encodeURIComponent(sessionId)}/api/messages`, {
			method: 'POST',
			credentials: 'same-origin',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ session_id: chatId, message: text, request_id: sessionId })
		});
		if (res.ok) return ok(undefined);
		let body: { error?: unknown; message?: unknown } | undefined;
		try {
			body = await res.json();
		} catch {
			body = undefined;
		}
		return err({
			status: res.status,
			reason: typeof body?.error === 'string' ? body.error : '',
			message:
				typeof body?.message === 'string' && body.message ? body.message : res.statusText
		});
	} catch (e) {
		return err({ status: 0, reason: '', message: getExceptionMessage(e) });
	}
}

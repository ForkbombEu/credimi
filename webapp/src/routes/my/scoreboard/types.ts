// SPDX-FileCopyrightText: 2025 Forkbomb BV
// SPDX-License-Identifier: AGPL-3.0-or-later

export interface ScoreboardEntry {
	id: string;
	name: string;
	type: 'wallet' | 'issuer' | 'verifier' | 'pipeline';
	totalRuns: number;
	successCount: number;
	failureCount: number;
	successRate: number;
	lastRun: string;
}

export interface OTelAttribute {
	key: string;
	value: any;
}

export interface OTelStatus {
	code: string;
	message: string;
}

export interface OTelSpan {
	traceId: string;
	spanId: string;
	name: string;
	kind: string;
	startTimeUnixNano: number;
	endTimeUnixNano: number;
	attributes: OTelAttribute[];
	status: OTelStatus;
}

export interface OTelScopeSpan {
	scope: {
		name: string;
		version: string;
	};
	spans: OTelSpan[];
}

export interface OTelResourceSpan {
	resource: {
		attributes: OTelAttribute[];
	};
	scopeSpans: OTelScopeSpan[];
}

export interface OTelTracesData {
	resourceSpans: OTelResourceSpan[];
}

export interface ScoreboardData {
	summary: {
		wallets: ScoreboardEntry[];
		issuers: ScoreboardEntry[];
		verifiers: ScoreboardEntry[];
		pipelines: ScoreboardEntry[];
	};
	otelData: OTelTracesData;
}

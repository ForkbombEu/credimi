// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

//

export interface PipelineProgress {
	expected_duration_seconds: number;
	elapsed_seconds: number;
	percent: number;
	eta_seconds: number;
	sample_size: number;
}

/** Formats seconds as a compact human duration, e.g. "2m 13s". */
export function formatDurationParts(totalSeconds: number): string {
	const seconds = Math.max(0, Math.round(totalSeconds));
	const h = Math.floor(seconds / 3600);
	const min = Math.floor((seconds % 3600) / 60);
	const s = seconds % 60;
	if (h > 0) return `${h}h ${min}m`;
	if (min > 0) return `${min}m ${s}s`;
	return `${s}s`;
}

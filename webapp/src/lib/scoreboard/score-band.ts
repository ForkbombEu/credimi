// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { m } from '@/i18n';

//

export type ScoreBandKey = 'stable' | 'flaky' | 'failing' | 'broken' | 'none';

export type ScoreBand = {
	key: ScoreBandKey;
	label: string;
	/** Pill container classes (border, background, text) */
	pill: string;
	/** Status dot classes */
	dot: string;
	/** Mini progress bar fill classes */
	bar: string;
};

const BANDS: Record<ScoreBandKey, Omit<ScoreBand, 'label'>> = {
	stable: {
		key: 'stable',
		pill: 'border-emerald-200 bg-emerald-50 text-emerald-800',
		dot: 'bg-emerald-500',
		bar: 'bg-emerald-500'
	},
	flaky: {
		key: 'flaky',
		pill: 'border-amber-200 bg-amber-50 text-amber-800',
		dot: 'bg-amber-500',
		bar: 'bg-amber-500'
	},
	failing: {
		key: 'failing',
		pill: 'border-orange-200 bg-orange-50 text-orange-800',
		dot: 'bg-orange-500',
		bar: 'bg-orange-500'
	},
	broken: {
		key: 'broken',
		pill: 'border-red-200 bg-red-50 text-red-800',
		dot: 'bg-red-500',
		bar: 'bg-red-500'
	},
	none: {
		key: 'none',
		pill: 'border-border bg-muted text-muted-foreground',
		dot: 'bg-muted-foreground',
		bar: 'bg-muted-foreground'
	}
};

const LABELS: Record<Exclude<ScoreBandKey, 'none'>, () => string> = {
	stable: m.scoreboard_band_stable,
	flaky: m.scoreboard_band_flaky,
	failing: m.scoreboard_band_failing,
	broken: m.scoreboard_band_broken
};

/** Score bands: >=80 Stable, 60-79 Flaky, 30-59 Failing, <30 Broken, no data grey. */
export function scoreBand(percent: number | null | undefined, total: number): ScoreBand {
	if (!total || percent === null || percent === undefined || Number.isNaN(percent)) {
		return { ...BANDS.none, label: '' };
	}
	if (percent >= 80) return { ...BANDS.stable, label: LABELS.stable() };
	if (percent >= 60) return { ...BANDS.flaky, label: LABELS.flaky() };
	if (percent >= 30) return { ...BANDS.failing, label: LABELS.failing() };
	return { ...BANDS.broken, label: LABELS.broken() };
}

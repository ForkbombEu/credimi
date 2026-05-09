// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { loadScoreboardData } from './functions';
import Component from './table.svelte';
import { ScoreboardTable as Instance } from './table.svelte.js';

//

export { Component, Instance, loadScoreboardData as loadData };

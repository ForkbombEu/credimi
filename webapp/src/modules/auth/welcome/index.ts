// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { createSessionStorageHandlers } from '@/utils/storage';
import WelcomeBanner from './welcomeBanner.svelte';

const WelcomeSession = createSessionStorageHandlers('welcome');

export { WelcomeBanner, WelcomeSession };

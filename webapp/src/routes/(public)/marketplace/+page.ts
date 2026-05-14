// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { redirect } from '@/i18n';

export const load = async () => {
	redirect('/hub');
};

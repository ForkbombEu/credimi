// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { redirect } from '@/i18n/index.js';

//

export const load = async ({ parent }) => {
	const { pipeline } = await parent();
	if (pipeline.published) {
		redirect('/my/pipelines');
	}
};

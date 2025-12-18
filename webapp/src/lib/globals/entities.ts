// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { IconComponent } from '@/components/types';

//

export type EntityUIData = {
	id: string;
	icon: IconComponent;
	label: {
		singular: string;
		plural: string;
	};
	class: {
		bg: string;
		text: string;
		border: string;
	};
};

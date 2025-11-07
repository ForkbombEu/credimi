// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { Debounced } from 'runed';

type Props = {
	initialValue?: string;
	delay?: number;
	onSearch: (value: string) => void;
};

export class Search {
	text = $state('');
	private debouncedText: Debounced<string>;

	constructor(props: Props) {
		const { initialValue, delay = 300, onSearch } = props;

		this.debouncedText = new Debounced(() => this.text, delay);
		if (initialValue) this.text = initialValue;

		$effect(() => {
			onSearch(this.debouncedText.current);
		});
	}

	clear() {
		this.text = '';
	}
}

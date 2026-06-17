// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

export type PaneHandle = {
	getSize: () => number;
	resize: (size: number) => void;
};

export const STEPS_BUILDER_PANE_LAYOUT = {
	blocks: { addStep: 22, stepsSequence: 38, right: 40 },
	manual: { addStep: 14, stepsSequence: 26, editor: 60 }
} as const;

export function applyStepsBuilderPaneLayout(
	panes: {
		addStep: PaneHandle | null;
		stepsSequence: PaneHandle | null;
		right: PaneHandle | null;
	},
	isManual: boolean
) {
	if (isManual) {
		const layout = STEPS_BUILDER_PANE_LAYOUT.manual;
		panes.addStep?.resize(layout.addStep);
		panes.stepsSequence?.resize(layout.stepsSequence);
		panes.right?.resize(layout.editor);
	} else {
		const layout = STEPS_BUILDER_PANE_LAYOUT.blocks;
		panes.addStep?.resize(layout.addStep);
		panes.stepsSequence?.resize(layout.stepsSequence);
		panes.right?.resize(layout.right);
	}
}

// SPDX-FileCopyrightText: 2025 The Forkbomb Company
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import plantumlEncoder from 'plantuml-encoder';

const PLANTUML_SERVER = 'https://www.plantuml.com/plantuml';

export default function remarkPlantuml() {
	return (tree) => {
		const newChildren = [];

		for (const node of tree.children) {
			if (node.type === 'code' && node.lang === 'puml') {
				const encoded = plantumlEncoder.encode(node.value);
				newChildren.push({
					type: 'html',
					value: `<img src="${PLANTUML_SERVER}/svg/${encoded}" alt="PlantUML diagram" loading="lazy" style="max-width:100%" />`,
				});
			} else {
				newChildren.push(node);
			}
		}

		tree.children = newChildren;
	};
}

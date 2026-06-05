// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

const WIDE_TABLE_WIDTH_PX = 800;
const WIDE_TABLE_COLUMN_COUNT = 6;

const PRINT_TABLE_STYLES = `
body {
	margin: 0;
	padding: 0;
}

.overflow-x-auto {
	overflow: visible !important;
	max-width: none !important;
}

table {
	width: 100% !important;
	table-layout: fixed !important;
	border-collapse: collapse !important;
}

th {
	background: #f4f4f5 !important;
	border-bottom: 1px solid #d4d4d8 !important;
	padding: 0.5rem 0.75rem !important;
	text-align: left !important;
	font-size: 0.72rem !important;
	font-weight: 600 !important;
	letter-spacing: 0.08em !important;
	text-transform: uppercase !important;
	word-break: break-word !important;
	overflow-wrap: anywhere !important;
	white-space: normal !important;
}

td {
	border-bottom: 1px solid #e4e4e7 !important;
	padding: 0.5rem 0.75rem !important;
	font-size: 0.92em !important;
	vertical-align: top !important;
	word-break: break-word !important;
	overflow-wrap: anywhere !important;
	white-space: normal !important;
}

tr,
th,
td {
	break-inside: avoid;
	page-break-inside: avoid;
}

thead {
	display: table-header-group;
}
`;

function prepareElementForPrint(element: HTMLElement): HTMLElement {
	const clone = element.cloneNode(true) as HTMLElement;

	clone.querySelectorAll('.overflow-x-auto').forEach((el) => {
		el.classList.remove('overflow-x-auto', 'overflow-y-clip', 'max-w-full');
	});

	return clone;
}

function hasWideTables(element: HTMLElement): boolean {
	return [...element.querySelectorAll('table')].some((table) => {
		const columnCount = table.rows[0]?.cells.length ?? 0;
		return table.scrollWidth > WIDE_TABLE_WIDTH_PX || columnCount >= WIDE_TABLE_COLUMN_COUNT;
	});
}

function injectPrintStyles(doc: Document, landscape: boolean) {
	const style = doc.createElement('style');
	style.textContent = `
		@page {
			margin: 1cm;
			${landscape ? 'size: landscape;' : ''}
		}

		${PRINT_TABLE_STYLES}
	`;
	doc.head.appendChild(style);
}

export function printElement(element: HTMLElement) {
	const iframe = document.createElement('iframe');
	iframe.style.cssText = 'position:fixed;width:0;height:0;border:0';
	document.body.appendChild(iframe);

	const doc = iframe.contentDocument;
	if (!doc) {
		iframe.remove();
		return;
	}

	doc.open();
	doc.write('<!DOCTYPE html><html><head></head><body></body></html>');
	doc.close();

	for (const node of document.querySelectorAll('link[rel="stylesheet"], style')) {
		doc.head.appendChild(node.cloneNode(true));
	}

	const landscape = hasWideTables(element);
	injectPrintStyles(doc, landscape);
	doc.body.appendChild(prepareElementForPrint(element));

	const contentWindow = iframe.contentWindow;
	if (!contentWindow) {
		iframe.remove();
		return;
	}

	const cleanup = () => iframe.remove();

	contentWindow.addEventListener('afterprint', cleanup, { once: true });
	contentWindow.focus();
	contentWindow.print();
	setTimeout(cleanup, 1000);
}

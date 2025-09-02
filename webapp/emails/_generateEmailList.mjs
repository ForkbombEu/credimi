// SPDX-FileCopyrightText: 2024 The Forkbomb Company
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/// <reference types="@types/node" />

// @ts-check

import fs from 'node:fs';
import { fileURLToPath } from 'node:url';
import path from 'path';

/* - Logic - */

const emailsFolderPath = path.join(dirname(), '..', 'static', 'emails');
const emailsPaths = getAllFilesInFolder(emailsFolderPath).filter((p) => p.includes('.html'));

/** @type {[string, string[]][]} */
const emailDataEntries = emailsPaths.map((emailPath) => {
	const name = path.parse(path.basename(emailPath)).name;
	const attributes = getEmailAttributes(emailPath);
	return [name, attributes];
});

const emailData = Object.fromEntries(emailDataEntries);

const fileContent = `type EmailData = ${JSON.stringify(emailData, null, 4)}`;
const filePath = path.join(emailsFolderPath, '_emailData.d.ts');
fs.writeFileSync(filePath, fileContent);

/* - Utils - */

function dirname() {
	const __filename = fileURLToPath(import.meta.url);
	const __dirname = path.dirname(__filename);
	return __dirname;
}

/**
 * Helper function to recursively find all files in a directory
 * @param {string} dirPath
 * @param {string[]} [exclude=[]]
 * @param {string[]} arrayOfFiles
 * @returns
 */
function getAllFilesInFolder(dirPath, exclude = [], arrayOfFiles = []) {
	const files = fs.readdirSync(dirPath);

	files.forEach(function (file) {
		const fullPath = path.join(dirPath, file);
		if (exclude.includes(fullPath)) return;
		if (fs.statSync(fullPath).isDirectory()) {
			arrayOfFiles = getAllFilesInFolder(fullPath, exclude, arrayOfFiles);
		} else {
			arrayOfFiles.push(fullPath);
		}
	});

	return arrayOfFiles;
}

/**
 * @param {string} path
 * @returns {string[]}
 */
function getEmailAttributes(path) {
	const fileContent = fs.readFileSync(path, 'utf-8');
	let noComments = fileContent;
	let previous;
	do {
		previous = noComments;
		noComments = noComments.replace(/<!--[\s\S]*?-->/g, '');
	} while (noComments !== previous);
	const pattern = /{{\.(\w+)}}/g;
	const matches = [...noComments.matchAll(pattern)].map((match) => match[1]);
	return matches;
}

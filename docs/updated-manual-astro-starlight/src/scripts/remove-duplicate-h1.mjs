import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const ROOT = path.resolve(__dirname, '../content/docs');

async function walk(dir) {
  const entries = await fs.readdir(dir, { withFileTypes: true });
  const out = [];
  for (const entry of entries) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) out.push(...await walk(full));
    else if (entry.name.endsWith('.md')) out.push(full);
  }
  return out;
}

function extractFrontmatter(content) {
  const m = content.match(/^---\n([\s\S]*?)\n---\n*/);
  if (!m) return null;
  const raw = m[1];
  const titleMatch = raw.match(/^title:\s*(.+)$/m);
  if (!titleMatch) return null;

  let title = titleMatch[1].trim();
  title = title.replace(/^['"]|['"]$/g, '');

  return {
    title,
    header: m[0],
    body: content.slice(m[0].length),
  };
}

function normalize(str) {
  return str
    .normalize('NFKD')
    .toLowerCase()
    .replace(/[#*_`~]/g, '')
    .replace(/[\p{Extended_Pictographic}\p{Symbol}\p{Punctuation}]/gu, '')
    .replace(/\s+/g, ' ')
    .trim();
}

async function main() {
  const files = await walk(ROOT);

  for (const file of files) {
    const content = await fs.readFile(file, 'utf8');
    const fm = extractFrontmatter(content);
    if (!fm) continue;

    const lines = fm.body.split('\n');
    let i = 0;
    while (i < lines.length && lines[i].trim() === '') i++;

    if (i >= lines.length) continue;
    const line = lines[i].trim();
    if (!line.startsWith('# ')) continue;

    const h1 = line.slice(2).trim();

    if (normalize(h1) === normalize(fm.title)) {
      lines.splice(i, 1);
      if (i < lines.length && lines[i].trim() === '') lines.splice(i, 1);

      const updated = fm.header + lines.join('\n');
      await fs.writeFile(file, updated, 'utf8');
      console.log(`FIX  ${file}`);
    }
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});

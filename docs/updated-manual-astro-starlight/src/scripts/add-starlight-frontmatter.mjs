// scripts/add-starlight-frontmatter.mjs
import fs from 'node:fs/promises';
import path from 'node:path';

const ROOT = path.resolve('src/content/docs');

async function walk(dir) {
  const entries = await fs.readdir(dir, { withFileTypes: true });
  const files = await Promise.all(
    entries.map(async (entry) => {
      const full = path.join(dir, entry.name);
      if (entry.isDirectory()) return walk(full);
      return full;
    })
  );
  return files.flat();
}

function hasFrontmatter(content) {
  return content.startsWith('---\n') || content.startsWith('---\r\n');
}

function titleFromContent(content, filePath) {
  const h1 = content.match(/^#\s+(.+)$/m);
  if (h1?.[1]) return h1[1].trim();

  const base = path.basename(filePath, path.extname(filePath));
  if (base.toLowerCase() === 'index') {
    return path.basename(path.dirname(filePath));
  }

  return base
    .replace(/[_-]+/g, ' ')
    .replace(/\b\w/g, (c) => c.toUpperCase());
}

function yamlEscape(str) {
  return JSON.stringify(str);
}

async function main() {
  const files = (await walk(ROOT)).filter((f) => f.endsWith('.md'));

  for (const file of files) {
    const content = await fs.readFile(file, 'utf8');

    if (hasFrontmatter(content)) {
      console.log(`SKIP  ${file} (already has frontmatter)`);
      continue;
    }

    const title = titleFromContent(content, file);
    const frontmatter =
      `---\n` +
      `title: ${yamlEscape(title)}\n` +
      `description: ""\n` +
      `---\n\n`;

    await fs.writeFile(file, frontmatter + content, 'utf8');
    console.log(`DONE  ${file}`);
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
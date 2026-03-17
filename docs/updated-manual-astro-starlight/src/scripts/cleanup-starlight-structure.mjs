import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const ROOT = path.resolve(__dirname, '..'); // src/
const DOCS = path.join(ROOT, 'content', 'docs');
const ASTRO_CONFIG = path.resolve(ROOT, '..', 'astro.config.mjs');
const PUBLIC_DIR = path.resolve(ROOT, '..', 'public');

const DIR_RENAMES = {
  Manual: 'manual',
  Legal: 'legal',
  Software_Architecture: 'software-architecture',
};

const FILE_RENAMES = {
  'software-architecture/1_start.md': 'software-architecture/intro.md',
  'software-architecture/2_functionalities.md': 'software-architecture/functionalities.md',
  'software-architecture/3_user_stories.md': 'software-architecture/user-stories.md',
  'software-architecture/4_building_blocks.md': 'software-architecture/building-blocks.md',
  'software-architecture/5_implementation.md': 'software-architecture/implementation-deployment.md',
  'software-architecture/6_testing_demo.md': 'software-architecture/testing-demo-strategy.md',
  'software-architecture/7_dev_setup.md': 'software-architecture/developer-setup.md',
  'software-architecture/8_docker.md': 'software-architecture/docker-deployment.md',
  'software-architecture/Workflow_YAML.md': 'software-architecture/workflow-yaml.md',
};

const ORDER_MAP = {
  'software-architecture/index.md': 0,
  'software-architecture/intro.md': 1,
  'software-architecture/functionalities.md': 2,
  'software-architecture/user-stories.md': 3,
  'software-architecture/building-blocks.md': 4,
  'software-architecture/implementation-deployment.md': 5,
  'software-architecture/testing-demo-strategy.md': 6,
  'software-architecture/developer-setup.md': 7,
  'software-architecture/docker-deployment.md': 8,
  'software-architecture/workflow-yaml.md': 9,
};

const DESCRIPTION_MAP = {
  'manual/index.md': 'Overview and entry point for Credimi documentation.',
  'legal/privacy-policy.md': 'Privacy policy for Credimi.',
  'legal/terms-and-conditions.md': 'Terms and conditions for Credimi.',
  'software-architecture/index.md': 'Overview of the Credimi software architecture.',
  'software-architecture/intro.md': 'Introduction and goals of the Credimi architecture.',
  'software-architecture/functionalities.md': 'Planned functionalities of Credimi.',
  'software-architecture/user-stories.md': 'User stories covered by Credimi.',
  'software-architecture/building-blocks.md': 'Core building blocks of the Credimi platform.',
  'software-architecture/implementation-deployment.md': 'Implementation and deployment details for Credimi.',
  'software-architecture/testing-demo-strategy.md': 'Testing and demo strategy for Credimi.',
  'software-architecture/developer-setup.md': 'Developer setup instructions for Credimi.',
  'software-architecture/docker-deployment.md': 'Docker deployment instructions for Credimi.',
  'software-architecture/workflow-yaml.md': 'Workflow YAML examples for Credimi.',
};

const LINK_REPLACEMENTS = [
  ['(/Manual/', '(/manual/'],
  ['(/Legal/', '(/legal/'],
  ['(/Software_Architecture/', '(/software-architecture/'],
  ['](/Manual/', '](/manual/'],
  ['](/Legal/', '](/legal/'],
  ['](/Software_Architecture/', '](/software-architecture/'],
  ['(../Manual/', '(../manual/'],
  ['(../Legal/', '(../legal/'],
  ['(../Software_Architecture/', '(../software-architecture/'],
  ['(./Manual/', '(./manual/'],
  ['(./Legal/', '(./legal/'],
  ['(./Software_Architecture/', '(./software-architecture/'],
  ['1_start', 'intro'],
  ['2_functionalities', 'functionalities'],
  ['3_user_stories', 'user-stories'],
  ['4_building_blocks', 'building-blocks'],
  ['5_implementation', 'implementation-deployment'],
  ['6_testing_demo', 'testing-demo-strategy'],
  ['7_dev_setup', 'developer-setup'],
  ['8_docker', 'docker-deployment'],
  ['Workflow_YAML', 'workflow-yaml'],
];

async function exists(p) {
  try {
    await fs.access(p);
    return true;
  } catch {
    return false;
  }
}

async function walk(dir) {
  const entries = await fs.readdir(dir, { withFileTypes: true });
  const out = [];
  for (const entry of entries) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) out.push(...await walk(full));
    else out.push(full);
  }
  return out;
}

function relFromDocs(absPath) {
  return path.relative(DOCS, absPath).replace(/\\/g, '/');
}

function parseFrontmatter(content) {
  const m = content.match(/^---\n([\s\S]*?)\n---\n?/);
  if (!m) return null;
  return { raw: m[1], full: m[0], body: content.slice(m[0].length) };
}

function titleFromFrontmatter(raw) {
  const m = raw.match(/^title:\s*(.+)$/m);
  if (!m) return null;
  return m[1].trim().replace(/^['"]|['"]$/g, '');
}

function withFrontmatter(content, updater) {
  const fm = parseFrontmatter(content);
  if (!fm) return content;
  const updatedRaw = updater(fm.raw);
  return `---\n${updatedRaw}\n---\n${fm.body.replace(/^\n?/, '')}`;
}

function ensureDescription(raw, description) {
  if (!description || /^description:\s*/m.test(raw)) return raw;
  const lines = raw.split('\n');
  let inserted = false;
  const out = [];
  for (const line of lines) {
    out.push(line);
    if (!inserted && /^title:\s*/.test(line)) {
      out.push(`description: "${description.replace(/"/g, '\\"')}"`);
      inserted = true;
    }
  }
  if (!inserted) out.push(`description: "${description.replace(/"/g, '\\"')}"`);
  return out.join('\n');
}

function ensureSidebarOrder(raw, order) {
  if (order === undefined) return raw;
  if (/^sidebar:\s*$/m.test(raw)) {
    if (/^\s+order:\s*/m.test(raw)) {
      return raw.replace(/^(\s+order:\s*).+$/m, `$1${order}`);
    }
    return raw.replace(/^sidebar:\s*$/m, `sidebar:\n  order: ${order}`);
  }
  return `${raw}\nsidebar:\n  order: ${order}`;
}

async function renameDirs() {
  for (const [from, to] of Object.entries(DIR_RENAMES)) {
    const src = path.join(DOCS, from);
    const dst = path.join(DOCS, to);
    if (await exists(src) && !(await exists(dst))) {
      await fs.rename(src, dst);
      console.log(`DIR  ${from} -> ${to}`);
    }
  }
}

async function renameFiles() {
  for (const [fromRel, toRel] of Object.entries(FILE_RENAMES)) {
    const src = path.join(DOCS, fromRel);
    const dst = path.join(DOCS, toRel);
    if (await exists(src) && !(await exists(dst))) {
      await fs.rename(src, dst);
      console.log(`FILE ${fromRel} -> ${toRel}`);
    }
  }
}

async function rewriteMarkdownFiles() {
  const files = (await walk(DOCS)).filter((f) => f.endsWith('.md') || f.endsWith('.mdx'));
  for (const file of files) {
    let content = await fs.readFile(file, 'utf8');

    for (const [from, to] of LINK_REPLACEMENTS) {
      content = content.split(from).join(to);
    }

    const rel = relFromDocs(file);
    content = withFrontmatter(content, (raw) => {
      let out = raw;
      out = ensureDescription(out, DESCRIPTION_MAP[rel]);
      out = ensureSidebarOrder(out, ORDER_MAP[rel]);
      return out;
    });

    await fs.writeFile(file, content, 'utf8');
    console.log(`EDIT ${rel}`);
  }
}

async function patchAstroConfig() {
  if (!(await exists(ASTRO_CONFIG))) return;

  let content = await fs.readFile(ASTRO_CONFIG, 'utf8');

  content = content.replace(/directory:\s*['"]Manual['"]/g, `directory: 'manual'`);
  content = content.replace(/directory:\s*['"]Legal['"]/g, `directory: 'legal'`);
  content = content.replace(/directory:\s*['"]Software_Architecture['"]/g, `directory: 'software-architecture'`);

  if (!/logo:\s*\{/.test(content)) {
    content = content.replace(
      /title:\s*['"][^'"]+['"],?/,
      (m) => `${m}\n      logo: { src: '/credimi-logo.png' },`
    );
  } else {
    content = content.replace(
      /logo:\s*\{[\s\S]*?\}/m,
      `logo: { src: '/credimi-logo.png' }`
    );
  }

  await fs.writeFile(ASTRO_CONFIG, content, 'utf8');
  console.log(`EDIT astro.config.mjs`);
}

async function copyLogo() {
  const src = path.join(DOCS, 'images', 'logo', 'credimi_logo-transp_emblem.png');
  const dst = path.join(PUBLIC_DIR, 'credimi-logo.png');
  if (!(await exists(src))) {
    console.log(`SKIP logo source missing`);
    return;
  }
  await fs.mkdir(PUBLIC_DIR, { recursive: true });
  await fs.copyFile(src, dst);
  console.log(`COPY public/credimi-logo.png`);
}

async function main() {
  await renameDirs();
  await renameFiles();
  await rewriteMarkdownFiles();
  await copyLogo();
  await patchAstroConfig();
  console.log('Done.');
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
/*
 * Build a TOOLS.md from generated MCP tools JSON
 * Usage: node examples/build-tools-readme.js [pathToJson] [outPath]
 * Defaults: examples/generated/n8n-openapi-tools.json -> examples/generated/TOOLS.md
 */

const fs = require('fs');
const path = require('path');

function loadTools(jsonPath) {
  const raw = fs.readFileSync(jsonPath, 'utf8');
  const data = JSON.parse(raw);
  return Array.isArray(data.tools) ? data.tools : [];
}

function groupByTag(tools) {
  const groups = new Map();
  for (const t of tools) {
    const name = t.name || '';
    let tag = 'api'; // Default tag

    // Try to extract tag from n8n-style names
    const n8nParts = String(name).split('.');
    if (n8nParts.length >= 3 && n8nParts[0] === 'n8n') {
      tag = n8nParts[2];
    } else {
      // Try to extract tag from Hostinger/Docker-style names
      const hostingerParts = String(name).split('_');
      if (hostingerParts.length >= 2) {
        // Take the first part and capitalize it
        tag = hostingerParts[0].charAt(0).toUpperCase() + hostingerParts[0].slice(1);
      }
    }

    if (!groups.has(tag)) groups.set(tag, []);
    groups.get(tag).push(t);
  }
  for (const [, list] of groups) {
    list.sort((a, b) => (a.name || '').localeCompare(b.name || ''));
  }
  return groups;
}

function buildMarkdown(tools) {
  const total = tools.length;
  const groups = groupByTag(tools);
  const lines = [];
  lines.push('# n8n MCP Tools');
  lines.push('');
  lines.push(`Total tools: ${total}`);
  lines.push('');
  const tagNames = Array.from(groups.keys()).sort();
  for (const tag of tagNames) {
    lines.push(`## ${tag}`);
    const list = groups.get(tag);
    for (const t of list) {
      const desc = t.description ? ` — ${t.description}` : '';
      lines.push(`- ${t.name}${desc}`);
    }
    lines.push('');
  }
  return lines.join('\n');
}

function main() {
  const n8nToolsPath = path.resolve(__dirname, 'generated', 'n8n-openapi-tools.json');
  const hostingerToolsPath = path.resolve(__dirname, 'generated', 'hostinger-openapi-tools.json');
  const outPath = process.argv[3] || path.resolve(__dirname, 'generated', 'TOOLS.md');

  let allTools = [];
  if (fs.existsSync(n8nToolsPath)) {
    allTools = allTools.concat(loadTools(n8nToolsPath));
  }
  if (fs.existsSync(hostingerToolsPath)) {
    allTools = allTools.concat(loadTools(hostingerToolsPath));
  }

  const md = buildMarkdown(allTools);
  fs.mkdirSync(path.dirname(outPath), { recursive: true });
  fs.writeFileSync(outPath, md);
  console.log(`Wrote ${outPath} (${allTools.length} tools)`);
}

main();


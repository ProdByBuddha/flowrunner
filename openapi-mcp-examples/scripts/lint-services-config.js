// Lint a services config file (e.g., examples/services.merged.json)
// Checks for duplicate names and basic shape { services: [...] }.
// Usage:
//   node examples/scripts/lint-services-config.js --file examples/services.merged.json

const fs = require('fs');
const path = require('path');

function parseArgs(argv) {
  const out = { _: [] };
  for (let i = 0; i < argv.length; i++) {
    const t = argv[i];
    if (t.startsWith('--')) {
      const k = t.slice(2);
      const v = argv[i + 1] && !argv[i + 1].startsWith('--') ? argv[++i] : 'true';
      out[k] = v;
    } else out._.push(t);
  }
  return out;
}

function main() {
  const args = parseArgs(process.argv.slice(2));
  const file = args.file || '';
  if (!file) {
    console.error('Usage: node examples/scripts/lint-services-config.js --file <path>');
    process.exit(1);
  }
  const raw = fs.readFileSync(path.resolve(process.cwd(), file), 'utf8');
  let cfg;
  try { cfg = JSON.parse(raw); } catch (e) {
    console.error('Invalid JSON:', e.message);
    process.exit(2);
  }
  const list = Array.isArray(cfg.services) ? cfg.services : [];
  if (list.length === 0) {
    console.error('No services found (expected { services: [...] }).');
    process.exit(3);
  }
  const seen = new Map();
  const dups = [];
  for (const s of list) {
    if (!s || typeof s !== 'object') continue;
    const name = String(s.name || '');
    if (!name) continue;
    if (seen.has(name)) dups.push(name);
    else seen.set(name, true);
  }
  if (dups.length) {
    console.error('Duplicate service names found:', Array.from(new Set(dups)).join(', '));
    process.exit(4);
  }
  console.log(`OK: ${list.length} services; no duplicates.`);
}

main();

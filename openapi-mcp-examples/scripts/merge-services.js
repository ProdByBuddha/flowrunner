// Merge two services JSON files (both shaped as { services: [...] })
// Strategies:
// - replace (default): replace entries from --base with entries from --add when names match; otherwise append new.
// - skip: keep base entries on name conflicts; append only new names.
// - append: alias for skip; append only new services and never modify existing.
// - report: keep base entries on conflicts (like skip), append only new; also emit a conflict report JSON to stdout or --report-out.
// Usage:
//   node examples/scripts/merge-services.js --base examples/services.example.json --add examples/services.generated.json --out examples/services.merged.json [--strategy replace|skip|report] [--report-out path]

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

function readJson(p) {
  const s = fs.readFileSync(path.resolve(process.cwd(), p), 'utf8');
  return JSON.parse(s);
}

function main() {
  const args = parseArgs(process.argv.slice(2));
  const basePath = args.base || '';
  const addPath = args.add || '';
  const outPath = args.out || '';
  const strategy = String(args.strategy || 'replace').toLowerCase(); // replace|skip|append|report
  const reportOut = args['report-out'] || '';
  if (!basePath || !addPath || !outPath) {
    console.error('Usage: node examples/scripts/merge-services.js --base <file> --add <file> --out <file> [--strategy replace|skip]');
    process.exit(1);
  }
  const base = readJson(basePath);
  const add = readJson(addPath);
  const baseList = Array.isArray(base.services) ? base.services : [];
  const addList = Array.isArray(add.services) ? add.services : [];
  const byName = new Map();
  for (const s of baseList) {
    if (!s || !s.name) continue; byName.set(String(s.name), s);
  }
  const conflicts = [];
  for (const s of addList) {
    if (!s || !s.name) continue;
    const name = String(s.name);
    if (byName.has(name)) {
      if (strategy === 'replace') byName.set(name, s);
      else if (strategy === 'report') conflicts.push({ name, base: byName.get(name), add: s });
      // skip existing for skip/report
    } else {
      byName.set(name, s);
    }
  }
  const merged = { services: Array.from(byName.values()) };
  const outJson = JSON.stringify(merged, null, 2);
  fs.mkdirSync(path.dirname(path.resolve(process.cwd(), outPath)), { recursive: true });
  fs.writeFileSync(path.resolve(process.cwd(), outPath), outJson);
  console.log(`Merged ${baseList.length} base + ${addList.length} add -> ${merged.services.length} services at ${outPath}`);
  if (strategy === 'report') {
    const rep = { conflicts: conflicts.map(c => ({ name: c.name })) };
    const repStr = JSON.stringify(rep, null, 2);
    if (reportOut) {
      fs.mkdirSync(path.dirname(path.resolve(process.cwd(), reportOut)), { recursive: true });
      fs.writeFileSync(path.resolve(process.cwd(), reportOut), repStr);
      console.log(`Conflict report written to ${reportOut} (${conflicts.length} conflict(s))`);
    } else {
      console.log('Conflict report (stdout):');
      console.log(repStr);
    }
  }
}

main();

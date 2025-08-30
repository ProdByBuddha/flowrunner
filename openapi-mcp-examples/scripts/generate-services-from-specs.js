// Generate a services.json from all specs in ./specs (json|yaml|yml)
// Usage:
//   node examples/scripts/generate-services-from-specs.js --out examples/services.generated.json

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
  const specsDir = path.resolve(process.cwd(), 'specs');
  if (!fs.existsSync(specsDir)) {
    console.error('No specs/ directory found. Create it and add *.json|*.yaml|*.yml');
    process.exit(1);
  }
  const services = [];
  for (const f of fs.readdirSync(specsDir)) {
    const lower = f.toLowerCase();
    if (!/\.(json|ya?ml)$/.test(lower)) continue;
    const name = path.basename(f).replace(/\.(json|ya?ml)$/i, '');
    services.push({
      name,
      type: 'openapi',
      specFile: `./specs/${f}`,
      baseUrl: '',
      auth: { kind: 'bearer', env: `${name.toUpperCase()}_TOKEN` },
      filters: {}
    });
  }
  const obj = { services };
  const json = JSON.stringify(obj, null, 2);
  const outPath = args.out || '';
  if (outPath) {
    fs.mkdirSync(path.dirname(path.resolve(process.cwd(), outPath)), { recursive: true });
    fs.writeFileSync(path.resolve(process.cwd(), outPath), json);
    console.log(`Wrote ${outPath} (${services.length} services)`);
  } else {
    process.stdout.write(json + '
');
  }
}

main();

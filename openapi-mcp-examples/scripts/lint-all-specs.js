// Lint all OpenAPI specs under ./specs (recursively) using the bundled linter.
// Usage: npm run openapi:lint:all

const fs = require('fs');
const path = require('path');
const { spawnSync } = require('child_process');

function* walk(dir) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const p = path.join(dir, entry.name);
    if (entry.isDirectory()) yield* walk(p);
    else yield p;
  }
}

function isSpec(file) {
  const lower = file.toLowerCase();
  return /\.(json|ya?ml)$/.test(lower);
}

function main() {
  const root = path.resolve(process.cwd(), 'specs');
  if (!fs.existsSync(root)) {
    console.error('No specs/ directory found.');
    process.exit(1);
  }
  const files = Array.from(walk(root)).filter(isSpec);
  if (files.length === 0) {
    console.log('No spec files found in specs/.');
    process.exit(0);
  }
  let failed = 0;
  for (const file of files) {
    process.stdout.write(`Linting: ${file}\n`);
    const res = spawnSync(process.execPath, [path.resolve('examples/lint-openapi.js'), '--file', file], { stdio: 'inherit' });
    if (res.status !== 0) failed++;
  }
  if (failed) {
    console.error(`\n${failed} spec(s) failed lint.`);
    process.exit(1);
  } else {
    console.log(`\nAll ${files.length} spec(s) passed.`);
  }
}

main();


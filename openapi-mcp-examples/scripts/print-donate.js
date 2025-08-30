const fs = require('fs');
const path = require('path');
try {
  const p = path.resolve(process.cwd(), 'donate.json');
  const raw = fs.readFileSync(p, 'utf8');
  const obj = JSON.parse(raw);
  for (const [k, v] of Object.entries(obj)) {
    process.stdout.write(`${k}: ${v}
`);
  }
} catch (e) {
  console.error('Failed to read donate.json:', e.message);
  process.exit(1);
}

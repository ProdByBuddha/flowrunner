/*
 * n8n Workflows CLI
 * - List, export, update, activate/deactivate workflows via REST API
 * - Auth: N8N_API_KEY (X-N8N-API-KEY) or N8N_BEARER_TOKEN or Basic Auth
 * - Base URL: N8N_API_URL (default http://localhost:5678/rest)
 *
 * Usage examples:
 *   N8N_API_URL=https://your-n8n/rest N8N_API_KEY=... npm run n8n:list
 *   npm run n8n:export -- --id <workflowId> --out data/workflows/<name>.json
 *   npm run n8n:update -- --id <workflowId> --file data/workflows/<name>.json
 *   npm run n8n:dump -- --out-dir data/workflows
 */

import fs from 'fs';
import path from 'path';
import https from 'https';
import http from 'http';

const BASE_URL = process.env.N8N_API_URL || 'http://localhost:5678/rest';

function buildHeaders(extra) {
  const headers = Object.assign(
    { 'Accept': 'application/json', 'Content-Type': 'application/json' },
    extra || {}
  );
  if (process.env.N8N_API_KEY) headers['X-N8N-API-KEY'] = process.env.N8N_API_KEY;
  if (process.env.N8N_BEARER_TOKEN) headers['Authorization'] = `Bearer ${process.env.N8N_BEARER_TOKEN}`;
  if (process.env.N8N_BASIC_AUTH_USER && process.env.N8N_BASIC_AUTH_PASS) {
    const token = Buffer.from(`${process.env.N8N_BASIC_AUTH_USER}:${process.env.N8N_BASIC_AUTH_PASS}`).toString('base64');
    if (!headers['Authorization']) headers['Authorization'] = `Basic ${token}`;
  }
  return headers;
}

function httpRequest(method, urlString, { headers, body, timeoutMs } = {}) {
  return new Promise((resolve, reject) => {
    try {
      const url = new URL(urlString);
      const lib = url.protocol === 'https:' ? https : http;
      const data = body == null ? null : Buffer.from(typeof body === 'string' ? body : JSON.stringify(body));
      const req = lib.request(
        {
          protocol: url.protocol,
          hostname: url.hostname,
          port: url.port || (url.protocol === 'https:' ? 443 : 80),
          path: url.pathname + url.search,
          method,
          headers: buildHeaders(Object.assign({}, headers, data ? { 'Content-Length': String(data.length) } : {}))
        },
        (res) => {
          let resBody = '';
          res.setEncoding('utf8');
          res.on('data', (chunk) => (resBody += chunk));
          res.on('end', () => resolve({ status: res.statusCode, statusText: res.statusMessage, headers: res.headers, body: resBody }));
        }
      );
      req.on('error', reject);
      if (timeoutMs) req.setTimeout(timeoutMs, () => req.destroy(new Error('Request timed out')));
      if (data) req.write(data);
      req.end();
    } catch (err) {
      reject(err);
    }
  });
}

function makeUrl(pathname, query) {
  const base = new URL(BASE_URL);
  const joinedPath = (base.pathname.replace(/\/$/, '') + '/' + String(pathname || '').replace(/^\//, '')) || '/';
  const dest = new URL(base.origin);
  dest.pathname = joinedPath;
  if (query && typeof query === 'object') {
    for (const [k, v] of Object.entries(query)) {
      if (v === undefined || v === null) continue;
      dest.searchParams.set(k, String(v));
    }
  }
  return dest.toString();
}

async function api(method, path, { query, body } = {}) {
  const url = makeUrl(path, query);
  const res = await httpRequest(method, url, { body, timeoutMs: 30000 });
  let json;
  try {
    json = res.body ? JSON.parse(res.body) : null;
  } catch (_) {
    json = res.body;
  }
  if (res.status < 200 || res.status >= 300) {
    const err = new Error(`HTTP ${res.status} ${res.statusText}`);
    err.response = { status: res.status, body: json };
    throw err;
  }
  return json;
}

function parseArgs(argv) {
  const out = { _: [] };
  for (let i = 0; i < argv.length; i++) {
    const t = argv[i];
    if (t.startsWith('--')) {
      const key = t.slice(2);
      const val = argv[i + 1] && !argv[i + 1].startsWith('--') ? argv[++i] : 'true';
      out[key] = val;
    } else {
      out._.push(t);
    }
  }
  return out;
}

async function listWorkflows() {
  const res = await api('GET', '/workflows');
  const items = Array.isArray(res) ? res : res?.data || [];
  items.forEach((w) => {
    console.log(`${w.id}\t${w.name}\tactive=${w.active}`);
  });
}

async function getWorkflow(id) {
  const w = await api('GET', `/workflows/${id}`);
  console.log(JSON.stringify(w, null, 2));
}

async function exportWorkflow(id, outPath) {
  const w = await api('GET', `/workflows/${id}`);
  const dir = path.dirname(outPath);
  fs.mkdirSync(dir, { recursive: true });
  fs.writeFileSync(outPath, JSON.stringify(w, null, 2));
  console.log(`Exported ${id} -> ${outPath}`);
}

async function dumpAll(outDir) {
  const res = await api('GET', '/workflows');
  const items = Array.isArray(res) ? res : res?.data || [];
  fs.mkdirSync(outDir, { recursive: true });
  for (const w of items) {
    const file = path.join(outDir, `${sanitize(w.name)}-${w.id}.json`);
    const full = await api('GET', `/workflows/${w.id}`);
    fs.writeFileSync(file, JSON.stringify(full, null, 2));
    console.log(`Dumped ${w.id} -> ${file}`);
  }
}

function sanitize(name) {
  return String(name).replace(/[^a-z0-9._-]+/gi, '_');
}

async function updateWorkflow(id, filePath) {
  const body = JSON.parse(fs.readFileSync(filePath, 'utf8'));
  // Ensure id matches or is set
  body.id = id;
  const updated = await api('PUT', `/workflows/${id}`, { body });
  console.log(`Updated workflow ${id}: ${updated.name}`);
}

async function activateWorkflow(id) {
  await api('POST', `/workflows/${id}/activate`);
  console.log(`Activated workflow ${id}`);
}

async function deactivateWorkflow(id) {
  await api('POST', `/workflows/${id}/deactivate`);
  console.log(`Deactivated workflow ${id}`);
}

async function main() {
  const args = parseArgs(process.argv.slice(2));
  const cmd = args._[0];

  if (!cmd || args['help']) {
    console.log('n8n Workflows CLI');
    console.log('Commands:');
    console.log('  list');
    console.log('  get --id <id>');
    console.log('  export --id <id> --out <file>');
    console.log('  dump-all --out-dir <dir>');
    console.log('  update --id <id> --file <file>');
    console.log('  activate --id <id>');
    console.log('  deactivate --id <id>');
    console.log('\nEnv: N8N_API_URL, N8N_API_KEY | N8N_BEARER_TOKEN | N8N_BASIC_AUTH_USER/PASS');
    process.exit(cmd ? 1 : 0);
  }

  try {
    switch (cmd) {
      case 'list':
        await listWorkflows();
        break;
      case 'get':
        if (!args.id) throw new Error('--id is required');
        await getWorkflow(args.id);
        break;
      case 'export':
        if (!args.id || !args.out) throw new Error('--id and --out are required');
        await exportWorkflow(args.id, args.out);
        break;
      case 'dump-all':
        if (!args['out-dir']) throw new Error('--out-dir is required');
        await dumpAll(args['out-dir']);
        break;
      case 'update':
        if (!args.id || !args.file) throw new Error('--id and --file are required');
        await updateWorkflow(args.id, args.file);
        break;
      case 'activate':
        if (!args.id) throw new Error('--id is required');
        await activateWorkflow(args.id);
        break;
      case 'deactivate':
        if (!args.id) throw new Error('--id is required');
        await deactivateWorkflow(args.id);
        break;
      default:
        throw new Error(`Unknown command: ${cmd}`);
    }
  } catch (err) {
    console.error('Error:', err.message);
    if (err.response) console.error(JSON.stringify(err.response, null, 2));
    process.exit(1);
  }
}

main();

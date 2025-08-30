/*
 * Hostinger MCP Server (stdio JSON-RPC)
 * - Dynamically generates tools from Hostinger OpenAPI (examples/specs/hostinger-api.json or URL)
 * - Injects bearer token from env for all calls (Authorization: Bearer ...)
 *
 * Env:
 * - HOSTINGER_API_URL (default: from spec servers[])
 * - HOSTINGER_API_TOKEN (Bearer token; preferred)
 * - HOSTINGER_SPEC_FILE (optional path to OpenAPI JSON)
 * - HOSTINGER_SPEC_URL (optional URL to OpenAPI JSON)
 * - DEBUG_HTTP=1 to log outbound request summaries (templates and generator honor this)
 * - Hardening (applies to generated tools via templates):
 *   - OPENAPI_MCP_ALLOWED_METHODS, OPENAPI_MCP_ALLOWED_PATHS, OPENAPI_MCP_RATE_LIMIT, OPENAPI_MCP_RATE_WINDOW_MS
 */

import fs from 'fs';
import path from 'path';
import https from 'https';
import http from 'http';
import { fileURLToPath } from 'url';

// Get __dirname equivalent for ESM
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Load environment variables (dotenvx recommended)
try { 
  const { config } = await import('@dotenvx/dotenvx');
  config({ quiet: true }); 
} catch (_) {
  // Fallback to basic dotenv
  try { 
    const { config } = await import('dotenv');
    config(); 
  } catch (_) {}
}

const tools = [];
let hostingerSdk = null;
try { 
  const hostingerModule = await import('hostinger-api-sdk'); 
  hostingerSdk = hostingerModule.default || hostingerModule; 
} catch (_) { 
  hostingerSdk = null; 
}

async function httpGetJson(urlString, headers = {}) {
  return new Promise((resolve, reject) => {
    try {
      const u = new URL(urlString);
      const lib = u.protocol === 'https:' ? https : http;
      const req = lib.request({
        protocol: u.protocol,
        hostname: u.hostname,
        port: u.port || (u.protocol === 'https:' ? 443 : 80),
        path: u.pathname + u.search,
        method: 'GET',
        headers: Object.assign({ Accept: 'application/json' }, headers)
      }, (res) => {
        let body = '';
        res.setEncoding('utf8');
        res.on('data', (c) => (body += c));
        res.on('end', () => {
          try { resolve(JSON.parse(body)); } catch (e) { reject(new Error('Non-JSON response')); }
        });
      });
      req.on('error', reject); req.end();
    } catch (err) { reject(err); }
  });
}

async function loadOpenApiSpec() {
  const SPEC_FILE = process.env.HOSTINGER_SPEC_FILE || '';
  const SPEC_URL = process.env.HOSTINGER_SPEC_URL || '';
  if (SPEC_FILE && fs.existsSync(SPEC_FILE)) {
    return JSON.parse(fs.readFileSync(path.resolve(process.cwd(), SPEC_FILE), 'utf8'));
  }
  if (SPEC_URL) {
    return await httpGetJson(SPEC_URL);
  }
  // Fallback to local packaged spec
  const fallback = path.resolve(process.cwd(), 'examples/specs/hostinger-api.json');
  if (fs.existsSync(fallback)) return JSON.parse(fs.readFileSync(fallback, 'utf8'));
  throw new Error('No Hostinger OpenAPI spec found. Set HOSTINGER_SPEC_FILE or HOSTINGER_SPEC_URL, or include examples/specs/hostinger-api.json.');
}

async function loadTools() {
  const { generateMcpTools } = await import('../lib/openapi-generator/index.js');
  const spec = await loadOpenApiSpec();
  const apiBase = process.env.HOSTINGER_API_URL || (spec.servers && spec.servers[0] && spec.servers[0].url) || '';
  const token = process.env.HOSTINGER_API_TOKEN || '';
  const securityHandlers = {
    apiToken: ({ def, headers, args }) => {
      const t = token || args.bearerToken || '';
      if (!t) throw new Error('Missing Hostinger API token (set HOSTINGER_API_TOKEN or pass bearerToken).');
      headers['Authorization'] = `Bearer ${t}`;
    }
  };
  const gen = await generateMcpTools(spec, { baseUrl: apiBase, securityHandlers });
  const rawTools = gen.map((t) => ({ name: t.name, description: t.description || '', inputSchema: t.inputSchema || { type: 'object' }, handler: t.handler }));

  // Optional: curated high-level tools via official SDK (if installed and enabled)
  const curated = [];
  if (hostingerSdk && /^(1|true|yes)$/i.test(String(process.env.HOSTINGER_USE_SDK || ''))) {
    try {
      const client = new hostingerSdk.Client({
        baseUrl: apiBase || undefined,
        token: token || undefined
      });
      // Add curated helpers prefixed to avoid name collisions
      curated.push(
        {
          name: 'hostinger.catalog.list',
          description: 'List catalog items (curated via SDK) — optional filters: category, name',
          inputSchema: { type: 'object', properties: { category: { type: 'string' }, name: { type: 'string' } } },
          handler: async ({ category, name } = {}) => {
            const res = await client.billing.catalog.list({ category, name });
            return res;
          }
        },
        {
          name: 'hostinger.domains.list',
          description: 'List domains for the authenticated account (curated via SDK)',
          inputSchema: { type: 'object' },
          handler: async () => {
            const res = await client.domains.list();
            return res;
          }
        },
        {
          name: 'hostinger.domains.checkAvailability',
          description: 'Check domain availability (curated via SDK)',
          inputSchema: { type: 'object', properties: { domain: { type: 'string' } }, required: ['domain'] },
          handler: async ({ domain }) => {
            const res = await client.domains.checkAvailability({ domain });
            return res;
          }
        }
      );
    } catch (e) {
      // eslint-disable-next-line no-console
      console.warn('[hostinger] SDK present but initialization failed:', e.message);
    }
  }
  const profile = String(process.env.HOSTINGER_PROFILE || '').toLowerCase();
  const includeRaw = /^(1|true|yes)$/i.test(String(process.env.HOSTINGER_INCLUDE_RAW || '1'));
  if (profile === 'curated') {
    const allowRaw = new Set((process.env.HOSTINGER_ALLOW_RAW || '').split(',').map((s) => s.trim()).filter(Boolean));
    for (const t of curated) tools.push(t);
    if (includeRaw) {
      for (const rt of rawTools) {
        if (allowRaw.size === 0 || allowRaw.has(rt.name)) tools.push(rt);
      }
    }
  } else {
    for (const t of curated) tools.push(t);
    for (const rt of rawTools) tools.push(rt);
  }
}

function listToolsResponse() {
  return { tools: tools.map((t) => ({ name: t.name, description: t.description || '', inputSchema: t.inputSchema || { type: 'object' } })) };
}

async function callToolByName(name, args) {
  const tool = tools.find((t) => t.name === name);
  if (!tool) throw new Error(`Unknown tool: ${name}`);
  return tool.handler(args || {});
}

function writeResponse(id, result, error) {
  const msg = { jsonrpc: '2.0', id };
  if (error) msg.error = error; else msg.result = result;
  process.stdout.write(JSON.stringify(msg) + '\n');
}

function toRpcError(err) { return { code: -32000, message: err.message || 'Request failed', data: err.response || null }; }

async function maybeOnce() {
  const args = process.argv.slice(2);
  if (args[0] !== '--once') return false;
  const method = args[1];
  const params = args[2] ? JSON.parse(args[2]) : {};
  try {
    if (method === 'tools/list') { console.log(JSON.stringify(listToolsResponse(), null, 2)); return true; }
    const tool = tools.find((t) => t.name === method);
    if (!tool) throw new Error(`Unknown method: ${method}`);
    const result = await tool.handler(params || {});
    console.log(JSON.stringify(result, null, 2));
    return true;
  } catch (err) {
    console.error('Error:', err.message);
    if (err.response) console.error(JSON.stringify(err.response, null, 2));
    process.exitCode = 1;
    return true;
  }
}

async function main() {
  await loadTools();
  const ranOnce = await maybeOnce();
  if (ranOnce) return;
  let buffer = '';
  process.stdin.setEncoding('utf8');
  process.stdin.on('data', async (chunk) => {
    buffer += chunk; let idx;
    while ((idx = buffer.indexOf('\n')) !== -1) {
      const line = buffer.slice(0, idx).trim(); buffer = buffer.slice(idx + 1);
      if (!line) continue;
      let msg; try { msg = JSON.parse(line); } catch (_) { writeResponse(null, null, { code: -32700, message: 'Parse error' }); continue; }
      const { id, method, params } = msg || {};
      if (method === 'initialize') { writeResponse(id, { protocolVersion: '0.1.0', serverInfo: { name: 'mcp-hostinger', version: '0.1.0' }, capabilities: { tools: {} } }); continue; }
      if (method === 'tools/list') { writeResponse(id, listToolsResponse()); continue; }
      if (method === 'tools/call') {
        const name = params?.name; const args = params?.arguments || {};
        try { const result = await callToolByName(name, args); writeResponse(id, { content: [{ type: 'json', json: result }] }); }
        catch (err) { writeResponse(id, null, toRpcError(err)); }
        continue;
      }
      writeResponse(id, null, { code: -32601, message: `Unknown method: ${method}` });
    }
  });
}

main().catch((e) => { console.error('Fatal:', e.message); process.exit(1); });

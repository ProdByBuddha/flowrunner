#!/usr/bin/env node
/*
 * Minimal Node-based MCP-like JSON-RPC server for n8n
 * - Exposes a generic n8n.request tool for any REST call
 * - Adds convenience methods for common operations
 *
 * Transport:
 * - STDIN/STDOUT, newline-delimited JSON-RPC 2.0 messages
 *   One JSON object per line. Example request:
 *   {"jsonrpc":"2.0","id":"1","method":"n8n.workflows.list","params":{}}
 *
 * Config via env (.env supported):
 * - N8N_API_URL (default: http://localhost:5678/rest)
 * - N8N_API_KEY (sent as X-N8N-API-KEY)
 * - N8N_BEARER_TOKEN (sent as Authorization: Bearer <token>)
 * - N8N_BASIC_AUTH_USER / N8N_BASIC_AUTH_PASS (sent as Basic auth)
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
  } catch (_) { }
}

const BASE_URL = process.env.N8N_API_URL || 'http://localhost:5678/rest';
const DRY_RUN = /^(1|true|yes)$/i.test(String(process.env.N8N_MCP_DRY_RUN || ''));

// Hardening configuration
const ALLOWED_METHODS = new Set(
  String(process.env.N8N_MCP_ALLOWED_METHODS || 'GET,POST,PUT,PATCH,DELETE')
    .split(',')
    .map((s) => s.trim().toUpperCase())
    .filter(Boolean)
);
const ALLOWED_PATH_PATTERNS = String(process.env.N8N_MCP_ALLOWED_PATHS || '.*')
  .split(',')
  .map((s) => s.trim())
  .filter(Boolean);
const RATE_LIMIT = Number(process.env.N8N_MCP_RATE_LIMIT || 60); // calls per window
const RATE_WINDOW_MS = Number(process.env.N8N_MCP_RATE_WINDOW_MS || 60_000);
let rateWindowStart = Date.now();
let rateCount = 0;

function wildcardToRegExp(pattern) {
  // Escape regex special chars, then replace * with .*
  const esc = pattern
    .replace(/[.+^${}()|[\]\\]/g, '\\$&')
    .replace(/\*/g, '.*');
  return new RegExp('^' + esc + '$');
}

const ALLOWED_PATH_REGEXES = ALLOWED_PATH_PATTERNS.map(wildcardToRegExp);

function checkRateLimit() {
  const now = Date.now();
  if (now - rateWindowStart >= RATE_WINDOW_MS) {
    rateWindowStart = now;
    rateCount = 0;
  }
  rateCount += 1;
  if (RATE_LIMIT > 0 && rateCount > RATE_LIMIT) {
    const err = new Error('Rate limit exceeded');
    err.code = 'RATE_LIMITED';
    throw err;
  }
}

function enforcePolicy(method, path) {
  const m = String(method || '').toUpperCase();
  if (!ALLOWED_METHODS.has(m)) {
    const err = new Error(`Method not allowed: ${m}`);
    err.code = 'METHOD_NOT_ALLOWED';
    throw err;
  }
  const p = String(path || '');
  const ok = ALLOWED_PATH_REGEXES.length === 0 || ALLOWED_PATH_REGEXES.some((re) => re.test(p));
  if (!ok) {
    const err = new Error(`Path not allowed: ${p}`);
    err.code = 'PATH_NOT_ALLOWED';
    throw err;
  }
}

// Minimal audit logging (disabled by default)
const LOG_FILE = process.env.N8N_MCP_LOG_FILE || '';
const LOG_MAX_SIZE = Number(process.env.N8N_MCP_LOG_MAX_SIZE || 1_048_576); // 1MB
const LOG_MAX_FILES = Number(process.env.N8N_MCP_LOG_MAX_FILES || 5);
const LOG_FORMAT = (process.env.N8N_MCP_LOG_FORMAT || 'json').toLowerCase();

function rotateLogsIfNeeded(filePath) {
  if (!LOG_FILE) return;
  try {
    if (!fs.existsSync(filePath)) return;
    const { size } = fs.statSync(filePath);
    if (LOG_MAX_SIZE > 0 && size > LOG_MAX_SIZE) {
      const dir = path.dirname(filePath);
      const base = path.basename(filePath);
      const stamp = new Date().toISOString().replace(/[:.]/g, '-');
      const rolled = path.join(dir, `${base}.${stamp}.log`);
      fs.renameSync(filePath, rolled);
      // Prune oldest
      const candidates = fs
        .readdirSync(dir)
        .filter((f) => f.startsWith(base + '.'))
        .map((f) => ({ f, t: fs.statSync(path.join(dir, f)).mtimeMs }))
        .sort((a, b) => b.t - a.t);
      for (let i = LOG_MAX_FILES; i < candidates.length; i++) {
        try { fs.unlinkSync(path.join(dir, candidates[i].f)); } catch (_) { }
      }
    }
  } catch (_) { }
}

function logEventSafe(evt) {
  if (!LOG_FILE) return;
  try {
    const rec = Object.assign({}, evt);
    const line = LOG_FORMAT === 'tsv'
      ? `${rec.time}\t${rec.method}\t${rec.path}\t${rec.hasQuery ? '1' : '0'}\t${rec.status}\t${rec.ok ? '1' : '0'}\t${rec.ms}\n`
      : JSON.stringify(rec) + '\n';
    rotateLogsIfNeeded(LOG_FILE);
    fs.mkdirSync(path.dirname(LOG_FILE), { recursive: true });
    fs.appendFileSync(LOG_FILE, line, 'utf8');
  } catch (_) { }
}

function buildHeaders(extra) {
  const headers = Object.assign(
    {
      'Accept': 'application/json',
      'Content-Type': 'application/json'
    },
    extra || {}
  );

  if (process.env.N8N_API_KEY) headers['X-N8N-API-KEY'] = process.env.N8N_API_KEY;
  if (process.env.N8N_BEARER_TOKEN) headers['Authorization'] = `Bearer ${process.env.N8N_BEARER_TOKEN}`;
  if (process.env.N8N_BASIC_AUTH_USER && process.env.N8N_BASIC_AUTH_PASS) {
    const token = Buffer.from(`${process.env.N8N_BASIC_AUTH_USER}:${process.env.N8N_BASIC_AUTH_PASS}`).toString('base64');
    headers['Authorization'] = headers['Authorization'] || `Basic ${token}`;
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
          res.on('end', () => {
            const result = {
              statusCode: res.statusCode,
              statusMessage: res.statusMessage,
              headers: res.headers,
              body: resBody
            };
            resolve(result);
          });
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
  // Respect base path (e.g., /api/v1). If pathname starts with '/', new URL() would discard it.
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

async function n8nRequest({ method, path, query, body, headers, timeoutMs }) {
  // Hardening: rate limit and allowlist checks
  enforcePolicy(method, path);
  checkRateLimit();
  const url = makeUrl(path, query);
  if (DRY_RUN) {
    return {
      dryRun: true,
      method: (method || 'GET').toUpperCase(),
      path: String(path || ''),
      url,
      hasQuery: !!(query && Object.keys(query).length),
      hasBody: !!(body && Object.keys(body || {}).length)
    };
  }
  const t0 = Date.now();
  const res = await httpRequest(method || 'GET', url, { headers, body, timeoutMs: timeoutMs || 30000 });
  const ms = Date.now() - t0;
  let parsed;
  try {
    parsed = res.body && res.body.length ? JSON.parse(res.body) : null;
  } catch (_) {
    parsed = res.body;
  }
  const ok = res.statusCode >= 200 && res.statusCode < 300;
  // Safe audit log (no headers/body, suppress query details)
  logEventSafe({
    time: new Date().toISOString(),
    method: (method || 'GET').toUpperCase(),
    path: String(path || ''),
    hasQuery: !!(query && Object.keys(query).length),
    status: res.statusCode,
    ok,
    ms
  });
  if (!ok) {
    const err = new Error(`HTTP ${res.statusCode} ${res.statusMessage}`);
    err.response = Object.assign({}, res, { body: parsed });
    throw err;
  }
  return parsed;
}

// Convenience wrappers (REST defaults for self-hosted at /rest) + Cloud (/api/v1)
const handlers = {
  'n8n.workflows.list': () => n8nRequest({ method: 'GET', path: '/workflows' }),
  'n8n.workflow.get': ({ id }) => n8nRequest({ method: 'GET', path: `/workflows/${id}` }),
  'n8n.workflow.create': ({ data }) => n8nRequest({ method: 'POST', path: '/workflows', body: data }),
  'n8n.workflow.update': ({ id, data }) => n8nRequest({ method: 'PUT', path: `/workflows/${id}`, body: data }),
  'n8n.workflow.delete': ({ id }) => n8nRequest({ method: 'DELETE', path: `/workflows/${id}` }),
  'n8n.workflow.activate': ({ id }) => n8nRequest({ method: 'POST', path: `/workflows/${id}/activate` }),
  'n8n.workflow.deactivate': ({ id }) => n8nRequest({ method: 'POST', path: `/workflows/${id}/deactivate` }),
  'n8n.executions.list': ({ workflowId, limit, lastId, finished }) => n8nRequest({ method: 'GET', path: '/executions', query: { workflowId, limit, lastId, finished } }),
  'n8n.execution.get': ({ id }) => n8nRequest({ method: 'GET', path: `/executions/${id}` }),
  'n8n.execution.stop': ({ id }) => n8nRequest({ method: 'POST', path: `/executions/${id}/stop` }),
  // Webhooks often live outside /rest; you can also call via n8n.request with a full path if BASE_URL points to /webhook
  'n8n.webhook.post': ({ path = '/webhook', body, query }) => n8nRequest({ method: 'POST', path, body, query })
};

// Minimal MCP tool registry for IDE/CLI agents
const tools = [
  { name: 'n8n.workflows.list', description: 'List workflows', inputSchema: { type: 'object' }, handler: handlers['n8n.workflows.list'] },
  { name: 'n8n.workflow.get', description: 'Get a workflow by id', inputSchema: { type: 'object', properties: { id: { type: 'string' } }, required: ['id'] }, handler: handlers['n8n.workflow.get'] },
  { name: 'n8n.workflow.create', description: 'Create workflow', inputSchema: { type: 'object', properties: { data: { type: 'object' } }, required: ['data'] }, handler: handlers['n8n.workflow.create'] },
  { name: 'n8n.workflow.update', description: 'Update workflow', inputSchema: { type: 'object', properties: { id: { type: 'string' }, data: { type: 'object' } }, required: ['id', 'data'] }, handler: handlers['n8n.workflow.update'] },
  { name: 'n8n.workflow.delete', description: 'Delete workflow', inputSchema: { type: 'object', properties: { id: { type: 'string' } }, required: ['id'] }, handler: handlers['n8n.workflow.delete'] },
  { name: 'n8n.workflow.activate', description: 'Activate workflow', inputSchema: { type: 'object', properties: { id: { type: 'string' } }, required: ['id'] }, handler: handlers['n8n.workflow.activate'] },
  { name: 'n8n.workflow.deactivate', description: 'Deactivate workflow', inputSchema: { type: 'object', properties: { id: { type: 'string' } }, required: ['id'] }, handler: handlers['n8n.workflow.deactivate'] },
  { name: 'n8n.executions.list', description: 'List executions', inputSchema: { type: 'object', properties: { workflowId: { type: 'string' }, limit: { type: 'number' }, lastId: { type: 'string' }, finished: { type: ['boolean', 'string'] } } }, handler: handlers['n8n.executions.list'] },
  { name: 'n8n.execution.get', description: 'Get execution by id', inputSchema: { type: 'object', properties: { id: { type: 'string' } }, required: ['id'] }, handler: handlers['n8n.execution.get'] },
  { name: 'n8n.execution.stop', description: 'Stop execution by id', inputSchema: { type: 'object', properties: { id: { type: 'string' } }, required: ['id'] }, handler: handlers['n8n.execution.stop'] },
  { name: 'n8n.webhook.post', description: 'POST to webhook path (relative to BASE)', inputSchema: { type: 'object', properties: { path: { type: 'string', default: '/webhook' }, body: { type: 'object' }, query: { type: 'object' } } }, handler: handlers['n8n.webhook.post'] }
];



// Option A: Dynamically generate OpenAPI-derived tools at startup via local generator
// Configure with OPENAPI_SPEC_FILE or OPENAPI_SPEC_URL. Optionally OPENAPI_BASE_URL overrides spec servers[].url
try {
  const SPEC_FILE = process.env.OPENAPI_SPEC_FILE || '';
  const SPEC_URL = process.env.OPENAPI_SPEC_URL || '';
  const OPENAPI_BASE_URL = process.env.OPENAPI_BASE_URL || process.env.N8N_API_URL || '';
  if (SPEC_FILE || SPEC_URL) {
    const { generateMcpTools } = await import('../lib/openapi-generator/index.js');

    async function loadOpenApiSpec() {
      if (SPEC_FILE) {
        const raw = fs.readFileSync(path.resolve(process.cwd(), SPEC_FILE), 'utf8');
        try {
          // Try JSON first
          return JSON.parse(raw);
        } catch (_) {
          // Try YAML
          try {
            const YAML = await import('yaml');
            return YAML.default.parse(raw);
          } catch (_) {
            throw new Error('OPENAPI_SPEC_FILE must be a valid JSON or YAML file');
          }
        }
      }
      // SPEC_URL path
      await new Promise((resolve) => setImmediate(resolve)); // yield
      return await new Promise((resolve, reject) => {
        try {
          const u = new URL(SPEC_URL);
          const lib = u.protocol === 'https:' ? https : http;
          const req = lib.request({
            protocol: u.protocol,
            hostname: u.hostname,
            port: u.port || (u.protocol === 'https:' ? 443 : 80),
            path: u.pathname + u.search,
            method: 'GET',
            headers: { 'Accept': 'application/json' }
          }, (res) => {
            let body = '';
            res.setEncoding('utf8');
            res.on('data', (c) => (body += c));
            res.on('end', async () => {
              // Try JSON first
              try { resolve(JSON.parse(body)); return; } catch (_) { }
              // Try YAML if available
              try {
                const YAML = await import('yaml');
                resolve(YAML.default.parse(body));
                return;
              } catch (_) { }
              reject(new Error('OPENAPI_SPEC_URL did not return a parseable spec (JSON/YAML)'));
            });
          });
          req.on('error', reject);
          req.end();
        } catch (err) { reject(err); }
      });
    }

    (async () => {
      try {
        const spec = await loadOpenApiSpec();
        const genTools = await generateMcpTools(spec, { baseUrl: OPENAPI_BASE_URL || undefined });
        for (const t of genTools) {
          tools.push({
            name: t.name,
            description: t.description || '',
            inputSchema: t.inputSchema || { type: 'object' },
            handler: t.handler
          });
        }
        // eslint-disable-next-line no-console
        console.log(`[mcp-n8n] Loaded ${genTools.length} OpenAPI tools`);
      } catch (err) {
        // eslint-disable-next-line no-console
        console.error('[mcp-n8n] Failed to load OpenAPI tools:', err.message);
      }
    })();
  } else {
    // Option B: Load pre-generated OpenAPI tools JSON if present
    const genPath = path.resolve(__dirname, 'generated', 'n8n-openapi-tools.json');
    if (fs.existsSync(genPath)) {
      const gen = JSON.parse(fs.readFileSync(genPath, 'utf8'));
      const generated = Array.isArray(gen.tools) ? gen.tools : [];
      for (const t of generated) {
        const { name, description, method, pathTemplate, inputSchema } = t;
        const handler = async (args = {}) => {
          let p = String(pathTemplate);
          p = p.replace(/\{([^}]+)\}/g, (_, k) => {
            if (!(k in args)) throw new Error(`Missing path param: ${k}`);
            return encodeURIComponent(String(args[k]));
          });
          const query = args.query || undefined;
          const body = args.body || undefined;
          return n8nRequest({ method: method || 'GET', path: p, query, body });
        };
        tools.push({ name, description: description || '', inputSchema: inputSchema || { type: 'object' }, handler });
      }
    }
  }
} catch (e) {
  // Ignore load errors; server will run with built-ins
}

function listToolsResponse() {
  return {
    tools: tools.map((t) => ({ name: t.name, description: t.description || '', inputSchema: t.inputSchema || { type: 'object' } }))
  };
}

async function callToolByName(name, args) {
  const tool = tools.find((t) => t.name === name);
  if (!tool) throw new Error(`Unknown tool: ${name}`);
  return tool.handler(args || {});
}

function writeResponse(id, result, error) {
  const msg = { jsonrpc: '2.0', id };
  if (error) msg.error = error;
  else msg.result = result;
  process.stdout.write(JSON.stringify(msg) + '\n');
}

function toRpcError(err) {
  return {
    code: -32000,
    message: err.message || 'Request failed',
    data: err.response || null
  };
}

// Optional single-shot mode: node examples/mcp-n8n-server.js --once method '{"param":"value"}'
async function maybeOnce() {
  const args = process.argv.slice(2);
  if (args[0] !== '--once') return false;
  const method = args[1];
  const params = args[2] ? JSON.parse(args[2]) : {};
  try {
    if (method === 'tools/list') {
      const out = listToolsResponse();
      console.log(JSON.stringify(out, null, 2));
      return true;
    }
    // Try tool registry first
    const toolMatch = (tools.find((t) => t.name === method) || null);
    if (toolMatch) {
      const result = await toolMatch.handler(params || {});
      console.log(JSON.stringify(result, null, 2));
      return true;
    }
    const handler = handlers[method] || null;
    if (!handler) throw new Error(`Unknown method: ${method}`);
    const result = await handler(params || {});
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
  const ranOnce = await maybeOnce();
  if (ranOnce) return;

  // Stream mode: newline-delimited JSON-RPC
  let buffer = '';
  process.stdin.setEncoding('utf8');
  process.stdin.on('data', (chunk) => {
    buffer += chunk;
    let idx;
    while ((idx = buffer.indexOf('\n')) !== -1) {
      const line = buffer.slice(0, idx).trim();
      buffer = buffer.slice(idx + 1);
      if (!line) continue;
      let msg;
      try {
        msg = JSON.parse(line);
      } catch (e) {
        writeResponse(null, null, { code: -32700, message: 'Parse error' });
        continue;
      }
      const { id, method, params } = msg || {};
      // Basic MCP method support
      if (method === 'initialize') {
        writeResponse(id, { protocolVersion: '2024-11-05', serverInfo: { name: 'mcp-n8n', version: '1.3.0' }, capabilities: { tools: {} } });
        continue;
      }
      if (method === 'tools/list') {
        writeResponse(id, listToolsResponse());
        continue;
      }
      if (method === 'tools/call') {
        const name = params?.name;
        const args = params?.arguments || params?.args || {};
        Promise.resolve()
          .then(() => callToolByName(name, args))
          .then((result) => writeResponse(id, { content: [{ type: 'json', json: result }] }))
          .catch((err) => writeResponse(id, null, toRpcError(err)));
        continue;
      }
      // Fallback: direct JSON-RPC to our handlers
      const handler = handlers[method] || null;
      if (!handler) {
        writeResponse(id, null, { code: -32601, message: `Method not found: ${method}` });
        continue;
      }
      Promise.resolve()
        .then(() => handler(params || {}))
        .then((result) => writeResponse(id, result))
        .catch((err) => writeResponse(id, null, toRpcError(err)));
    }
  });
}

main();

#!/usr/bin/env node
/*
 * Generic OpenAPI → MCP tool server
 * - Loads an OpenAPI 3.x spec (file or URL)
 * - Generates MCP tools via bundled generator
 * - Exposes MCP over stdio (initialize, tools/list, tools/call)
 *
 * Config via env:
 * - OPENAPI_SPEC_FILE or OPENAPI_SPEC_URL (one required)
 * - OPENAPI_BASE_URL (optional override of spec servers[0].url)
 * - Policy (optional):
 *   - OPENAPI_MCP_ALLOWED_METHODS (e.g., GET,POST)
 *   - OPENAPI_MCP_ALLOWED_PATHS (comma-separated patterns, supports *)
 *   - OPENAPI_MCP_RATE_LIMIT (calls per window), OPENAPI_MCP_RATE_WINDOW_MS
 *   - OPENAPI_MCP_LOG_FILE, OPENAPI_MCP_LOG_MAX_SIZE, OPENAPI_MCP_LOG_MAX_FILES, OPENAPI_MCP_LOG_FORMAT
 * - Auth (optional, used by generic security handlers):
 *   - OPENAPI_API_KEY (fallback for any apiKey scheme)
 *   - OPENAPI_APIKEY_<SCHEMENAME> (scheme-specific api key)
 *   - OPENAPI_BEARER_TOKEN
 *   - OPENAPI_BASIC_USER / OPENAPI_BASIC_PASS
 */

import fs from 'fs';
import path from 'path';
import https from 'https';
import http from 'http';
import { fileURLToPath } from 'url';

// Get __dirname equivalent for ESM
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const { generateMcpTools } = await import('../lib/openapi-generator/index.js');

function readEnvFileIfPresent() {
  try {
    const envPath = path.resolve(process.cwd(), '.env');
    if (!fs.existsSync(envPath)) return;
    const lines = fs.readFileSync(envPath, 'utf8').split(/\r?\n/);
    for (const line of lines) {
      const m = line.match(/^\s*([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.*)\s*$/);
      if (!m) continue;
      const key = m[1];
      let val = m[2];
      if ((val.startsWith('"') && val.endsWith('"')) || (val.startsWith("'") && val.endsWith("'"))) val = val.slice(1, -1);
      if (!process.env[key]) process.env[key] = val;
    }
  } catch (_) {}
}
readEnvFileIfPresent();

// Policy config
const ALLOWED_METHODS = new Set(
  String(process.env.OPENAPI_MCP_ALLOWED_METHODS || 'GET,POST,PUT,PATCH,DELETE')
    .split(',').map((s) => s.trim().toUpperCase()).filter(Boolean)
);
function wildcardToRegExp(pattern) {
  const esc = pattern.replace(/[.+^${}()|[\]\\]/g, '\\$&').replace(/\*/g, '.*');
  return new RegExp('^' + esc + '$');
}
const ALLOWED_PATH_REGEXES = String(process.env.OPENAPI_MCP_ALLOWED_PATHS || '.*')
  .split(',').map((s) => s.trim()).filter(Boolean).map(wildcardToRegExp);
const RATE_LIMIT = Number(process.env.OPENAPI_MCP_RATE_LIMIT || 60);
const RATE_WINDOW_MS = Number(process.env.OPENAPI_MCP_RATE_WINDOW_MS || 60_000);
let rateWindowStart = Date.now();
let rateCount = 0;
function checkRateLimit() {
  const now = Date.now();
  if (now - rateWindowStart >= RATE_WINDOW_MS) { rateWindowStart = now; rateCount = 0; }
  rateCount += 1;
  if (RATE_LIMIT > 0 && rateCount > RATE_LIMIT) { const e = new Error('Rate limit exceeded'); e.code = 'RATE_LIMITED'; throw e; }
}
function enforcePolicy(method, p) {
  const m = String(method || '').toUpperCase();
  if (!ALLOWED_METHODS.has(m)) { const e = new Error(`Method not allowed: ${m}`); e.code = 'METHOD_NOT_ALLOWED'; throw e; }
  const ok = ALLOWED_PATH_REGEXES.length === 0 || ALLOWED_PATH_REGEXES.some((re) => re.test(String(p || '')));
  if (!ok) { const e = new Error(`Path not allowed: ${p}`); e.code = 'PATH_NOT_ALLOWED'; throw e; }
}

// Minimal audit log
const LOG_FILE = process.env.OPENAPI_MCP_LOG_FILE || '';
const LOG_MAX_SIZE = Number(process.env.OPENAPI_MCP_LOG_MAX_SIZE || 1_048_576);
const LOG_MAX_FILES = Number(process.env.OPENAPI_MCP_LOG_MAX_FILES || 5);
const LOG_FORMAT = (process.env.OPENAPI_MCP_LOG_FORMAT || 'json').toLowerCase();
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
      const candidates = fs.readdirSync(dir)
        .filter((f) => f.startsWith(base + '.'))
        .map((f) => ({ f, t: fs.statSync(path.join(dir, f)).mtimeMs }))
        .sort((a, b) => b.t - a.t);
      for (let i = LOG_MAX_FILES; i < candidates.length; i++) {
        try { fs.unlinkSync(path.join(dir, candidates[i].f)); } catch (_) {}
      }
    }
  } catch (_) {}
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
  } catch (_) {}
}

// Build security handlers from env that work for common schemes
function buildSecurityHandlersFromEnv(api) {
  const handlers = {};
  const schemes = (api && api.components && api.components.securitySchemes) || {};
  for (const [name, def] of Object.entries(schemes)) {
    handlers[name] = ({ def: schemeDef, headers, query, args }) => {
      const d = schemeDef || def || {};
      if (!d || !d.type) return;
      if (d.type === 'apiKey') {
        const envKey = process.env[`OPENAPI_APIKEY_${String(name).toUpperCase()}`] || process.env.OPENAPI_API_KEY;
        const v = (args && (args[d.name] || args.apiKey)) || envKey;
        if (!v) return;
        if (d.in === 'header') headers[d.name] = v;
        else if (d.in === 'query') query[d.name] = v;
        else if (d.in === 'cookie') { if (args) args[d.name] = args[d.name] || v; }
      } else if (d.type === 'http' && d.scheme === 'bearer') {
        const v = (args && args.bearerToken) || process.env.OPENAPI_BEARER_TOKEN;
        if (v) headers['Authorization'] = `Bearer ${v}`;
      } else if (d.type === 'http' && d.scheme === 'basic') {
        const user = (args && args.username) || process.env.OPENAPI_BASIC_USER;
        const pass = (args && args.password) || process.env.OPENAPI_BASIC_PASS;
        if (user && pass) headers['Authorization'] = `Basic ${Buffer.from(`${user}:${pass}`).toString('base64')}`;
      }
    };
  }
  return handlers;
}

async function loadOpenApiSpec() {
  const SPEC_FILE = process.env.OPENAPI_SPEC_FILE || '';
  const SPEC_URL = process.env.OPENAPI_SPEC_URL || '';
  if (SPEC_FILE) {
    const raw = fs.readFileSync(path.resolve(process.cwd(), SPEC_FILE), 'utf8');
    try {
      // Try JSON first
      return JSON.parse(raw);
    } catch (_) {
      // Try YAML
      try {
        const YAML = await import('yaml');
        return YAML.parse(raw);
      } catch (_) {
        throw new Error('OPENAPI_SPEC_FILE must be a valid JSON or YAML file');
      }
    }
  }
  if (SPEC_URL) {
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
          try { resolve(JSON.parse(body)); return; } catch (_) {}
          // Try YAML if available
          try { 
            const YAML = await import('yaml'); 
            resolve(YAML.parse(body)); 
            return; 
          } catch (_) {}
          // Try to extract swaggerDoc JSON from a JS init file (e.g., Swagger UI init)
          try {
            const idx = body.indexOf('"swaggerDoc"');
            if (idx !== -1) {
              const braceStart = body.indexOf('{', idx);
              if (braceStart !== -1) {
                let i = braceStart, depth = 0, inStr = false, esc = false;
                for (; i < body.length; i++) {
                  const ch = body[i];
                  if (inStr) {
                    if (esc) { esc = false; continue; }
                    if (ch === '\\') { esc = true; continue; }
                    if (ch === '"') { inStr = false; continue; }
                    continue;
                  }
                  if (ch === '"') { inStr = true; continue; }
                  if (ch === '{') depth++;
                  if (ch === '}') { depth--; if (depth === 0) { i++; break; } }
                }
                const jsonStr = body.slice(braceStart, i);
                resolve(JSON.parse(jsonStr));
                return;
              }
            }
          } catch (_) {}
          reject(new Error('OPENAPI_SPEC_URL did not return a parseable spec (JSON/YAML/swaggerDoc)'));
        });
        });
        req.on('error', reject);
        req.end();
      } catch (err) { reject(err); }
    });
  }
  throw new Error('Set OPENAPI_SPEC_FILE or OPENAPI_SPEC_URL');
}

const tools = [];

async function initTools() {
  const spec = await loadOpenApiSpec();
  const baseUrl = process.env.OPENAPI_BASE_URL || undefined;
  const securityHandlers = buildSecurityHandlersFromEnv(spec);
  const genTools = await generateMcpTools(spec, { baseUrl, securityHandlers });
  for (const t of genTools) {
    tools.push({ name: t.name, description: t.description || '', inputSchema: t.inputSchema || { type: 'object' }, handler: t.handler });
  }
}

function listToolsResponse() {
  return { tools: tools.map((t) => ({ name: t.name, description: t.description || '', inputSchema: t.inputSchema || { type: 'object' } })) };
}

async function callToolByName(name, args) {
  const tool = tools.find((t) => t.name === name);
  if (!tool) throw new Error(`Unknown tool: ${name}`);
  // Enforce policy for generated tools based on their metadata if present
  try {
    const meta = tool;
    if (meta && meta.pathTemplate && meta.method) enforcePolicy(meta.method, meta.pathTemplate);
  } catch (_) {}
  checkRateLimit();
  const t0 = Date.now();
  try {
    const result = await tool.handler(args || {});
    const ms = Date.now() - t0;
    logEventSafe({ time: new Date().toISOString(), method: 'tools/call', path: name, hasQuery: !!args, status: 200, ok: true, ms });
    return result;
  } catch (err) {
    const ms = Date.now() - t0;
    logEventSafe({ time: new Date().toISOString(), method: 'tools/call', path: name, hasQuery: !!args, status: 500, ok: false, ms });
    throw err;
  }
}

function writeResponse(id, result, error) {
  const msg = { jsonrpc: '2.0', id };
  if (error) msg.error = error; else msg.result = result;
  process.stdout.write(JSON.stringify(msg) + '\n');
}
function toRpcError(err) {
  return { code: -32000, message: err.message || 'Request failed', data: err.response || null };
}

async function maybeOnce() {
  const args = process.argv.slice(2);
  if (args[0] !== '--once') return false;
  const method = args[1];
  const params = args[2] ? JSON.parse(args[2]) : {};
  await initTools();
  if (method === 'tools/list') { console.log(JSON.stringify(listToolsResponse(), null, 2)); return true; }
  const tool = tools.find((t) => t.name === method);
  if (tool) { const out = await tool.handler(params || {}); console.log(JSON.stringify(out, null, 2)); return true; }
  throw new Error(`Unknown method: ${method}`);
}

async function main() {
  const ranOnce = await maybeOnce();
  if (ranOnce) return;
  await initTools();
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
      try { msg = JSON.parse(line); } catch (_) { writeResponse(null, null, { code: -32700, message: 'Parse error' }); continue; }
      const { id, method, params } = msg || {};
      if (method === 'initialize') { writeResponse(id, { protocolVersion: '2024-11-05', serverInfo: { name: 'mcp-openapi', version: '1.3.0' }, capabilities: { tools: {} } }); continue; }
      if (method === 'tools/list') { writeResponse(id, listToolsResponse()); continue; }
      if (method === 'tools/call') {
        const name = params?.name; const args = params?.arguments || params?.args || {};
        Promise.resolve().then(() => callToolByName(name, args)).then((result) => writeResponse(id, { content: [{ type: 'json', json: result }] })).catch((err) => writeResponse(id, null, toRpcError(err)));
        continue;
      }
      writeResponse(id, null, { code: -32601, message: 'Method not found' });
    }
  });
}

main().catch((e) => { console.error('Fatal:', e.stack || e.message); process.exit(1); });

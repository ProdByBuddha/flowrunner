/*
 * Docker MCP Server (stdio JSON-RPC)
 * - Wraps Docker CLI (docker) for common operations with safety gates
 * - Optionally calls Docker Engine API via unix socket (DOCKER_SOCK) or TCP
 *
 * Env:
 * - DOCKER_BIN (default: docker)
 * - DOCKER_COMPOSE_BIN (default: docker compose)
 * - DOCKER_SOCK (default: /var/run/docker.sock) to use Engine API via unix socket
 * - DOCKER_API_HOST (optional, e.g., http://localhost:2375) to use TCP Engine API
 * - DOCKER_ALLOW_RUN=1 to allow container creation/start/exec/remove
 * - DOCKER_ALLOWED_IMAGES (comma-separated allowlist for run)
 * - DEBUG_DOCKER=1 to log spawned commands and Engine API requests
 * - Hardening (inherited from generated templates, used here to keep symmetry):
 *   - OPENAPI_MCP_RATE_LIMIT, OPENAPI_MCP_CONCURRENCY*
 */

import { spawn } from 'child_process';
import http from 'http';
import https from 'https';
import { URL } from 'url';

function logDebug(...a) { if (/^(1|true|yes)$/i.test(String(process.env.DEBUG_DOCKER || ''))) console.log('[docker]', ...a); }

function run(cmd, args, opts = {}) {
  return new Promise((resolve, reject) => {
    logDebug('spawn', cmd, args.join(' '));
    const child = spawn(cmd, args, { stdio: ['ignore', 'pipe', 'pipe'], ...opts });
    let out = '';
    let err = '';
    child.stdout.on('data', (d) => (out += d.toString()))
    child.stderr.on('data', (d) => (err += d.toString()))
    child.on('error', reject);
    child.on('close', (code) => {
      if (code === 0) resolve(out);
      else reject(new Error(err || out || `Command failed (${code})`));
    });
  });
}

async function engineRequest(method, path, { headers = {}, body, timeoutMs = 30000 } = {}) {
  const sock = process.env.DOCKER_SOCK || '/var/run/docker.sock';
  const host = process.env.DOCKER_API_HOST || '';
  if (!host) {
    // unix socket mode
    return new Promise((resolve, reject) => {
      const req = http.request({ socketPath: sock, path, method, headers }, (res) => {
        let b = '';
        res.setEncoding('utf8');
        res.on('data', (c) => (b += c));
        res.on('end', () => resolve({ statusCode: res.statusCode, headers: res.headers, body: b }));
      });
      req.on('error', reject);
      if (timeoutMs) req.setTimeout(timeoutMs, () => req.destroy(new Error('Request timed out')));
      if (body) req.write(typeof body === 'string' ? body : JSON.stringify(body));
      req.end();
    });
  } else {
    // TCP host
    const u = new URL(host);
    const lib = u.protocol === 'https:' ? https : http;
    return new Promise((resolve, reject) => {
      const req = lib.request({ protocol: u.protocol, hostname: u.hostname, port: u.port || (u.protocol === 'https:' ? 443 : 80), path, method, headers }, (res) => {
        let b = '';
        res.setEncoding('utf8');
        res.on('data', (c) => (b += c));
        res.on('end', () => resolve({ statusCode: res.statusCode, headers: res.headers, body: b }));
      });
      req.on('error', reject);
      if (timeoutMs) req.setTimeout(timeoutMs, () => req.destroy(new Error('Request timed out')));
      if (body) req.write(typeof body === 'string' ? body : JSON.stringify(body));
      req.end();
    });
  }
}

const tools = [];

function pushTool(name, description, inputSchema, handler) {
  tools.push({ name, description, inputSchema, handler });
}

const dockerBin = process.env.DOCKER_BIN || 'docker';
const composeBin = process.env.DOCKER_COMPOSE_BIN || 'docker'; // use 'docker compose'
const allowRun = /^(1|true|yes)$/i.test(String(process.env.DOCKER_ALLOW_RUN || ''));
const allowedImages = new Set(String(process.env.DOCKER_ALLOWED_IMAGES || '').split(',').map((s) => s.trim()).filter(Boolean));

// Non-destructive
pushTool('docker.ps', 'List containers', { type: 'object', properties: { all: { type: 'boolean', default: false } } }, async ({ all }) => {
  const args = ['ps', '--format', '{{json .}}']; if (all) args.push('-a');
  const out = await run(dockerBin, args);
  return out.trim().split(/\n+/).filter(Boolean).map((l) => { try { return JSON.parse(l); } catch { return l; } });
});

pushTool('docker.images', 'List images', { type: 'object' }, async () => {
  const out = await run(dockerBin, ['images', '--format', '{{json .}}']);
  return out.trim().split(/\n+/).filter(Boolean).map((l) => { try { return JSON.parse(l); } catch { return l; } });
});

// Destructive / gated
pushTool('docker.run', 'Run a container (gated by DOCKER_ALLOW_RUN)', {
  type: 'object', properties: {
    image: { type: 'string' }, name: { type: 'string' }, detach: { type: 'boolean', default: true },
    ports: { type: 'array', items: { type: 'string' }, description: 'Publish ports, e.g., ["8080:80"]' },
    env: { type: 'array', items: { type: 'string' }, description: 'Env vars, e.g., ["KEY=VAL"]' },
    cmd: { type: 'array', items: { type: 'string' }, description: 'Command args' }
  }, required: ['image']
}, async ({ image, name, detach = true, ports = [], env = [], cmd = [] }) => {
  if (!allowRun) throw new Error('Run not allowed; set DOCKER_ALLOW_RUN=1');
  if (allowedImages.size && !allowedImages.has(image)) throw new Error(`Image not allowed: ${image}`);
  const args = ['run']; if (detach) args.push('-d'); if (name) args.push('--name', name);
  for (const p of ports) args.push('-p', p);
  for (const e of env) args.push('-e', e);
  args.push(image, ...cmd);
  const out = await run(dockerBin, args);
  return { containerId: out.trim() };
});

pushTool('docker.stop', 'Stop a container', { type: 'object', properties: { id: { type: 'string' } }, required: ['id'] }, async ({ id }) => {
  const out = await run(dockerBin, ['stop', id]);
  return { stopped: out.trim() };
});

pushTool('docker.rm', 'Remove a container', { type: 'object', properties: { id: { type: 'string' }, force: { type: 'boolean', default: false } }, required: ['id'] }, async ({ id, force = false }) => {
  const args = ['rm']; if (force) args.push('-f'); args.push(id);
  const out = await run(dockerBin, args);
  return { removed: out.trim() };
});

pushTool('docker.logs', 'Get container logs', { type: 'object', properties: { id: { type: 'string' }, tail: { type: 'string' } }, required: ['id'] }, async ({ id, tail }) => {
  const args = ['logs']; if (tail) args.push('--tail', String(tail)); args.push(id);
  const out = await run(dockerBin, args);
  return { output: out };
});

pushTool('docker.exec', 'Exec a command in a container (gated by DOCKER_ALLOW_RUN)', {
  type: 'object', properties: { id: { type: 'string' }, cmd: { type: 'array', items: { type: 'string' } } }, required: ['id', 'cmd']
}, async ({ id, cmd }) => {
  if (!allowRun) throw new Error('Exec not allowed; set DOCKER_ALLOW_RUN=1');
  const out = await run(dockerBin, ['exec', id, ...cmd]);
  return { output: out };
});

pushTool('docker.compose.up', 'docker compose up -d (gated by DOCKER_ALLOW_RUN)', {
  type: 'object', properties: { file: { type: 'string', description: 'compose file' }, project: { type: 'string' } }
}, async ({ file, project }) => {
  if (!allowRun) throw new Error('Compose up not allowed; set DOCKER_ALLOW_RUN=1');
  const args = ['compose']; if (file) args.push('-f', file); if (project) args.push('-p', project); args.push('up', '-d');
  const out = await run(composeBin, args);
  return { output: out };
});

pushTool('docker.compose.down', 'docker compose down', {
  type: 'object', properties: { file: { type: 'string' }, project: { type: 'string' } }
}, async ({ file, project }) => {
  const args = ['compose']; if (file) args.push('-f', file); if (project) args.push('-p', project); args.push('down');
  const out = await run(composeBin, args);
  return { output: out };
});

// Generic Docker Engine API call
pushTool('docker.engine.request', 'Call Docker Engine API (unix socket or TCP)', {
  type: 'object', properties: { method: { type: 'string' }, path: { type: 'string' }, headers: { type: 'object' }, body: { type: 'object' } }, required: ['method', 'path']
}, async ({ method, path, headers, body }) => {
  const res = await engineRequest(String(method || 'GET').toUpperCase(), path, { headers, body });
  let parsed; try { parsed = JSON.parse(res.body || ''); } catch { parsed = res.body; }
  return { statusCode: res.statusCode, headers: res.headers, body: parsed };
});

// Curated Engine API helpers
function qs(params = {}) {
  const u = new URL('http://x');
  Object.entries(params).forEach(([k, v]) => {
    if (v === undefined || v === null || v === '') return;
    if (typeof v === 'object') u.searchParams.set(k, JSON.stringify(v));
    else u.searchParams.set(k, String(v));
  });
  return u.search;
}

pushTool('docker.engine.containers.list', 'List containers via Engine API', {
  type: 'object', properties: {
    all: { type: 'boolean', description: 'Show all containers (default shows just running)' },
    limit: { type: 'number' },
    size: { type: 'boolean' },
    filters: { type: 'object', description: 'JSON filters as object' }
  }
}, async ({ all, limit, size, filters }) => {
  const query = qs({ all: all ? 1 : undefined, limit, size: size ? 1 : undefined, filters });
  const res = await engineRequest('GET', `/containers/json${query}`, {});
  let parsed; try { parsed = JSON.parse(res.body || ''); } catch { parsed = res.body; }
  return parsed;
});

pushTool('docker.engine.images.list', 'List images via Engine API', {
  type: 'object', properties: {
    all: { type: 'boolean' },
    filters: { type: 'object' },
    digests: { type: 'boolean' }
  }
}, async ({ all, filters, digests }) => {
  const query = qs({ all: all ? 1 : undefined, filters, digests: digests ? 1 : undefined });
  const res = await engineRequest('GET', `/images/json${query}`, {});
  let parsed; try { parsed = JSON.parse(res.body || ''); } catch { parsed = res.body; }
  return parsed;
});

pushTool('docker.engine.version', 'Get Engine version', { type: 'object' }, async () => {
  const res = await engineRequest('GET', `/version`, {});
  let parsed; try { parsed = JSON.parse(res.body || ''); } catch { parsed = res.body; }
  return parsed;
});

pushTool('docker.engine.containers.inspect', 'Inspect a container via Engine API', {
  type: 'object', properties: {
    id: { type: 'string', description: 'Container ID or name' },
    size: { type: 'boolean', description: 'Return size information for containers' }
  }, required: ['id']
}, async ({ id, size }) => {
  const query = qs({ size: size ? 1 : undefined });
  const res = await engineRequest('GET', `/containers/${encodeURIComponent(id)}/json${query}`, {});
  let parsed; try { parsed = JSON.parse(res.body || ''); } catch { parsed = res.body; }
  return parsed;
});

pushTool('docker.engine.containers.stats', 'Container stats via Engine API', {
  type: 'object', properties: {
    id: { type: 'string', description: 'Container ID or name' },
    stream: { type: 'boolean', description: 'Stream stats (default false)', default: false }
  }, required: ['id']
}, async ({ id, stream = false }) => {
  // For non-streaming usage in MCP stdio, default to stream=false
  const query = qs({ stream: stream ? 1 : 0 });
  const res = await engineRequest('GET', `/containers/${encodeURIComponent(id)}/stats${query}`, {});
  let parsed; try { parsed = JSON.parse(res.body || ''); } catch { parsed = res.body; }
  return parsed;
});

pushTool('docker.engine.containers.logs', 'Container logs via Engine API (non-streaming)', {
  type: 'object', properties: {
    id: { type: 'string', description: 'Container ID or name' },
    stdout: { type: 'boolean', default: true },
    stderr: { type: 'boolean', default: true },
    timestamps: { type: 'boolean' },
    tail: { type: 'string', description: 'Number of lines from the end to show (e.g., "100")' },
    since: { type: 'string', description: 'Show logs since timestamp (Unix time or date string)' }
  }, required: ['id']
}, async ({ id, stdout = true, stderr = true, timestamps, tail, since }) => {
  const query = qs({ stdout: stdout ? 1 : undefined, stderr: stderr ? 1 : undefined, timestamps: timestamps ? 1 : undefined, tail, since });
  const res = await engineRequest('GET', `/containers/${encodeURIComponent(id)}/logs${query}`, {});
  // Logs are plain text (may contain ANSI). Do not JSON-parse.
  return { output: res.body, statusCode: res.statusCode };
});

pushTool('docker.engine.containers.top', 'Process list via Engine API', {
  type: 'object', properties: {
    id: { type: 'string', description: 'Container ID or name' },
    psArgs: { type: 'string', description: 'ps arguments (e.g., "aux")' }
  }, required: ['id']
}, async ({ id, psArgs }) => {
  const query = qs({ ps_args: psArgs });
  const res = await engineRequest('GET', `/containers/${encodeURIComponent(id)}/top${query}`, {});
  let parsed; try { parsed = JSON.parse(res.body || ''); } catch { parsed = res.body; }
  return parsed;
});

function listToolsResponse() { return { tools: tools.map((t) => ({ name: t.name, description: t.description || '', inputSchema: t.inputSchema || { type: 'object' } })) }; }
async function callToolByName(name, args) { const t = tools.find((x) => x.name === name); if (!t) throw new Error(`Unknown tool: ${name}`); return t.handler(args || {}); }
function writeResponse(id, result, error) { const msg = { jsonrpc: '2.0', id }; if (error) msg.error = error; else msg.result = result; process.stdout.write(JSON.stringify(msg) + '\n'); }
function toRpcError(err) { return { code: -32000, message: err.message || 'Request failed', data: err.response || null }; }

async function main() {
  let buf = '';
  process.stdin.setEncoding('utf8');
  process.stdin.on('data', async (chunk) => {
    buf += chunk; let idx;
    while ((idx = buf.indexOf('\n')) !== -1) {
      const line = buf.slice(0, idx).trim(); buf = buf.slice(idx + 1);
      if (!line) continue; let msg; try { msg = JSON.parse(line); } catch (_) { writeResponse(null, null, { code: -32700, message: 'Parse error' }); continue; }
      const { id, method, params } = msg || {};
      if (method === 'initialize') { writeResponse(id, { protocolVersion: '0.1.0', serverInfo: { name: 'mcp-docker', version: '0.1.0' }, capabilities: { tools: {} } }); continue; }
      if (method === 'tools/list') { writeResponse(id, listToolsResponse()); continue; }
      if (method === 'tools/call') {
        try { const result = await callToolByName(params?.name, params?.arguments || {}); writeResponse(id, { content: [{ type: 'json', json: result }] }); }
        catch (err) { writeResponse(id, null, toRpcError(err)); }
        continue;
      }
      writeResponse(id, null, { code: -32601, message: `Unknown method: ${method}` });
    }
  });
}

main().catch((e) => { console.error('Fatal:', e.message); process.exit(1); });

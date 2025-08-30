#!/usr/bin/env node
/*
 Quick Spec Gate — conformance + fuzzed dry runs (no network)
 - Validates OpenAPI (dereference + structural checks)
 - Generates MCP tools in-memory with a mock requester (no network)
 - For each operation/tool, runs N randomized dry-run calls to catch schema/path issues

 Usage:
   node examples/scripts/spec-gate.js --file <openapi.json> [--runs 3]
   node examples/scripts/spec-gate.js --url <openapi-url> [--runs 3]
   node examples/scripts/spec-gate.js --dir <folder-with-specs> [--runs 3]

 Exit non-zero on any failure.
 */

const fs = require('fs');
const path = require('path');
const https = require('https');
const http = require('http');
let SwaggerParser; try { SwaggerParser = require('swagger-parser'); } catch (_) { SwaggerParser = null; }
const { generateMcpTools } = require('../../lib/openapi-generator');

function parseArgs(argv){ const o={}; for(let i=0;i<argv.length;i++){ const t=argv[i]; if(t.startsWith('--')){ const k=t.slice(2); const v=argv[i+1]&&!argv[i+1].startsWith('--')?argv[++i]:'true'; o[k]=v; } } return o; }

function httpGet(urlString){ return new Promise((resolve,reject)=>{ try{ const u=new URL(urlString); const lib=u.protocol==='https:'?https:http; const req=lib.request({protocol:u.protocol,hostname:u.hostname,port:u.port|| (u.protocol==='https:'?443:80),path:u.pathname+u.search,method:'GET',headers:{Accept:'application/json'}},(res)=>{ let b=''; res.setEncoding('utf8'); res.on('data',(c)=>b+=c); res.on('end',()=>resolve({status:res.statusCode,body:b})); }); req.on('error',reject); req.end(); }catch(e){reject(e);} }); }

async function loadSpec({ file, url }){
  if (file) { return JSON.parse(fs.readFileSync(path.resolve(process.cwd(), file), 'utf8')); }
  if (url) { const res = await httpGet(url); if(res.status!==200) throw new Error(`Fetch failed ${res.status}`); try { return JSON.parse(res.body); } catch(e){ throw new Error('URL did not return JSON'); } }
  throw new Error('Provide --file or --url or --dir');
}

function pick(arr){ return arr[Math.floor(Math.random()*arr.length)]; }
function randInt(min,max){ return Math.floor(Math.random()*(max-min+1))+min; }
function uuid(){ return '00000000-0000-4000-8000-000000000000'; }

function genValue(schema){
  if (!schema || typeof schema !== 'object') return 'x';
  if (schema.enum && Array.isArray(schema.enum) && schema.enum.length) return pick(schema.enum);
  const t = Array.isArray(schema.type)?schema.type[0]:schema.type;
  switch (t) {
    case 'boolean': return Math.random() < 0.5;
    case 'integer': {
      let min = Number.isFinite(schema.minimum)?schema.minimum:0;
      let max = Number.isFinite(schema.maximum)?schema.maximum:(min+10);
      if (schema.exclusiveMinimum === true) min += 1;
      if (schema.exclusiveMaximum === true) max -= 1;
      let val = randInt(min, max);
      if (Number.isFinite(schema.multipleOf) && schema.multipleOf > 0) {
        const m = Math.max(1, Math.floor(schema.multipleOf));
        val = Math.ceil(val / m) * m;
      }
      return val;
    }
    case 'number': {
      let min = Number.isFinite(schema.minimum)?schema.minimum:0;
      let max = Number.isFinite(schema.maximum)?schema.maximum:(min+10);
      if (schema.exclusiveMinimum === true) min = min + Number.EPSILON;
      if (schema.exclusiveMaximum === true) max = max - Number.EPSILON;
      let val = min + Math.random()*(max-min);
      if (Number.isFinite(schema.multipleOf) && schema.multipleOf > 0) {
        const m = schema.multipleOf;
        val = Math.ceil(val / m) * m;
      }
      return val;
    }
    case 'array': {
      const minItems = Number.isFinite(schema.minItems)?schema.minItems:0;
      const maxItems = Number.isFinite(schema.maxItems)?schema.maxItems:Math.max(minItems, 2);
      const n = Math.min(maxItems, Math.max(minItems, randInt(minItems, Math.max(minItems, 2))));
      const items = schema.items || { type: 'string' };
      const out = []; for (let i=0;i<n;i++) out.push(genValue(items)); return out;
    }
    case 'object': {
      const out = {};
      const props = schema.properties || {};
      const req = Array.isArray(schema.required)?schema.required:[];
      for (const k of req) out[k] = genValue(props[k] || {});
      const optProb = Math.max(0, Math.min(1, Number(process.env.SPEC_GATE_OPT_PROB || '0.25')));
      for (const k of Object.keys(props)) { if (!req.includes(k) && Math.random()<optProb) out[k]=genValue(props[k]); }
      return out;
    }
    case 'string': default: {
      if (schema.format === 'email') return 'dev@example.com';
      if (schema.format === 'uuid') return uuid();
      if (schema.format === 'date') return '2025-01-01';
      if (schema.format === 'date-time') return '2025-01-01T00:00:00.000Z';
      if (schema.format === 'uri' || schema.format === 'url') return 'https://api.example.com/resource';
      if (schema.format === 'hostname') return 'example.com';
      if (schema.format === 'ipv4') return '192.168.0.1';
      if (schema.format === 'ipv6') return '2001:0db8:85a3:0000:0000:8a2e:0370:7334';
      if (schema.pattern) {
        try { const re = new RegExp(schema.pattern); if (re.test('alpha-123')) return 'alpha-123'; } catch (_) {}
      }
      if (typeof schema.minLength === 'number') return 'x'.repeat(Math.max(1,schema.minLength));
      return 'id';
    }
  }
}

function buildArgs(inputSchema){ const args={}; const s=inputSchema||{type:'object'}; const props=s.properties||{}; const req=Array.isArray(s.required)?s.required:[]; for(const k of req){ args[k]=genValue(props[k]||{}); } const optProb = Math.max(0, Math.min(1, Number(process.env.SPEC_GATE_OPT_PROB || '0.25'))); for(const k of Object.keys(props)){ if(!req.includes(k) && Math.random()<optProb) args[k]=genValue(props[k]); } return args; }

// Mock requester: never touches network; returns 200 OK with minimal bodies. Simulates OAuth token responses.
async function mockRequester(method, url, { headers, body } = {}){
  const u = String(url||'');
  const m = String(method||'GET').toUpperCase();
  if (m === 'POST' && (/token/i.test(u) || /grant_type=client_credentials/.test(String(body||'')))) {
    return { statusCode: 200, statusMessage: 'OK', headers: {}, body: JSON.stringify({ access_token: 'spec-gate' }) };
  }
  return { statusCode: 200, statusMessage: 'OK', headers: {}, body: '{}' };
}

async function gateForSpec(spec, { runs, filters }){
  // Validate/dereference
  if (SwaggerParser) {
    await SwaggerParser.validate(spec);
  }
  const baseUrl = (spec.servers && spec.servers[0] && spec.servers[0].url) || process.env.OPENAPI_BASE_URL || process.env.N8N_API_URL || 'http://localhost';
  const tools = await generateMcpTools(spec, { baseUrl, httpRequest: mockRequester, filters });
  let total=0, passed=0, failed=0;
  for (const t of tools) {
    for (let i=0;i<runs;i++){
      total++;
      try {
        const args = buildArgs(t.inputSchema);
        await t.handler(args);
        passed++;
      } catch (e) {
        failed++;
        console.error(`[spec-gate] FAIL tool=${t.name} iter=${i+1}: ${e.message}`);
      }
    }
  }
  console.log(`[spec-gate] tools=${tools.length} runs=${runs} total=${total} passed=${passed} failed=${failed}`);
  if (failed) process.exit(1);
}

async function main(){
  const args = parseArgs(process.argv.slice(2));
  const runs = Math.max(1, Number(args.runs || 3));
  // Parse include/exclude tags (comma-separated) and regex variants
  const toList = (v)=> (v? String(v).split(',').map(s=>s.trim()).filter(Boolean): undefined);
  const filters = {
    includeTags: toList(args['include-tags']),
    excludeTags: toList(args['exclude-tags']),
    includeTagsRe: toList(args['include-tags-re']),
    excludeTagsRe: toList(args['exclude-tags-re']),
    includeOps: toList(args['include-ops']),
    excludeOps: toList(args['exclude-ops']),
    includeOpsRe: toList(args['include-ops-re']),
    excludeOpsRe: toList(args['exclude-ops-re']),
    includePaths: toList(args['include-paths']),
    excludePaths: toList(args['exclude-paths']),
    includePathsRe: toList(args['include-paths-re']),
    excludePathsRe: toList(args['exclude-paths-re'])
  };
  if (args.dir) {
    const dir = path.resolve(process.cwd(), args.dir);
    const files = fs.existsSync(dir) ? fs.readdirSync(dir).filter(f=>/\.(json|ya?ml)$/i.test(f)).map(f=>path.join(dir,f)) : [];
    if (!files.length) { console.log(`[spec-gate] No specs in ${dir}`); return; }
    let anyFail = false;
    for (const f of files) {
      try {
        console.log(`[spec-gate] Validate + fuzz: ${f}`);
        const spec = JSON.parse(fs.readFileSync(f,'utf8'));
        await gateForSpec(spec, { runs, filters });
      } catch (e) {
        anyFail = true; console.error(`[spec-gate] FAILED: ${f}: ${e.message}`);
      }
    }
    if (anyFail) process.exit(1);
    return;
  }
  const spec = await loadSpec({ file: args.file, url: args.url });
  await gateForSpec(spec, { runs, filters });
}

main().catch((e)=>{ console.error('[spec-gate] Fatal:', e.message); process.exit(1); });

#!/usr/bin/env node
/**
 * Discover OpenAPI/Swagger and WSDL spec URLs from a documentation page.
 * Heuristics support Swagger UI (url/urls), Redoc (spec-url), Stoplight Elements (apiDescriptionUrl),
 * direct links to *.json/*.yaml/*.yml, and *.wsdl. Also scans linked JS bundles for embedded configs.
 * Domain-aware hints for dev.wix.com and dev.proof.com are included.
 *
 * Usage:
 *   node examples/scripts/discover-spec-from-docs.js --url https://docs.example.com/ref [--max-js 8]
 */

const { request: httpsRequest } = require('node:https');
const { request: httpRequest } = require('node:http');
const { URL } = require('node:url');

function parseArgs(argv){ const o={}; for(let i=0;i<argv.length;i++){ const a=argv[i]; if(a.startsWith('--')){ const k=a.slice(2); const v=argv[i+1]&&!argv[i+1].startsWith('--')?argv[++i]:true; o[k]=v; } } return o; }

function fetchText(url, { maxRedirects = 5, headers = {} } = {}) {
  return new Promise((resolve, reject) => {
    let redirects = 0;
    const doReq = (urlStr) => {
      try {
        const u = new URL(urlStr);
        const lib = u.protocol === 'https:' ? httpsRequest : httpRequest;
        const req = lib({
          method: 'GET',
          protocol: u.protocol,
          hostname: u.hostname,
          port: u.port || (u.protocol === 'https:' ? 443 : 80),
          path: u.pathname + u.search,
          headers: Object.assign({ 'User-Agent': 'spec-discover/1.1' }, headers)
        }, (res) => {
          const { statusCode, headers: resHeaders } = res;
          if (statusCode >= 300 && statusCode < 400 && resHeaders.location) {
            if (redirects++ >= maxRedirects) return reject(new Error('Too many redirects'));
            const next = new URL(resHeaders.location, u).toString();
            res.resume();
            return doReq(next);
          }
          let body = '';
          res.setEncoding('utf8');
          res.on('data', (c)=> body+=c);
          res.on('end', ()=> resolve({ url: u.toString(), statusCode, headers: resHeaders, body }));
        });
        req.on('error', reject);
        req.end();
      } catch (e) { reject(e); }
    };
    doReq(url);
  });
}

function unique(arr) { return Array.from(new Set(arr)); }
function resolveUrl(base, maybe) { try { return new URL(maybe, base).toString(); } catch { return null; } }

function extractFromHtml(html, baseUrl) {
  const out = [];
  const add = (u, type='openapi', source='html', hint=null) => { const r = resolveUrl(baseUrl, u); if (r) out.push({ url: r, type, source, hint }); };
  // Direct links that look like OpenAPI/Swagger or WSDL
  const linkRe = /(href|src)=["']([^"']*(?:openapi|swagger)[^"']*\.(?:json|ya?ml|js))["']/ig;
  let m; while ((m = linkRe.exec(html))) add(m[2], 'openapi');
  const wsdlRe = /(href|src)=["']([^"']*\.(?:wsdl))(?:["'])/ig;
  while ((m = wsdlRe.exec(html))) add(m[2], 'soap');
  // Redoc: <redoc spec-url="...">
  const redocRe = /<redoc[^>]*spec[-_ ]?url=["']([^"']+)["']/ig;
  while ((m = redocRe.exec(html))) add(m[1], 'openapi', 'html', 'redoc');
  // Stoplight Elements: <elements-api apiDescriptionUrl="...">
  const stoplightRe = /<elements-api[^>]*apiDescriptionUrl=["']([^"']+)["']/ig;
  while ((m = stoplightRe.exec(html))) add(m[1], 'openapi', 'html', 'stoplight');
  // Swagger UI: url: '...' or urls: [{url: '...'}]
  const swaggerUrlRe = /\burl\s*:\s*["']([^"']+)["']/ig;
  while ((m = swaggerUrlRe.exec(html))) add(m[1], 'openapi', 'html', 'swagger-ui');
  const swaggerUrlsArrayRe = /\burls\s*:\s*\[(.*?)\]/igs;
  while ((m = swaggerUrlsArrayRe.exec(html))) {
    const chunk = m[1] || '';
    const urlItemRe = /\burl\s*:\s*["']([^"']+)["']/ig;
    let mi; while ((mi = urlItemRe.exec(chunk))) add(mi[1], 'openapi', 'html', 'swagger-ui');
  }
  // Collect all script src to scan later
  const scripts = [];
  const scriptRe = /<script[^>]*src=["']([^"']+)["'][^>]*>/ig;
  let sm; while ((sm = scriptRe.exec(html))) { const u = resolveUrl(baseUrl, sm[1]); if (u) scripts.push(u); }
  return { candidates: unique(out), scripts: unique(scripts) };
}

function extractFromJs(js, baseUrl) {
  const out = [];
  const add = (u, type='openapi', source='js', hint=null) => { const r = resolveUrl(baseUrl, u); if (r) out.push({ url: r, type, source, hint }); };
  const swaggerUrlRe = /\burl\s*:\s*["']([^"']+)["']/ig;
  let m; while ((m = swaggerUrlRe.exec(js))) add(m[1], 'openapi', 'js', 'swagger-ui');
  const urlsArrayRe = /\burls\s*:\s*\[(.*?)\]/igs;
  while ((m = urlsArrayRe.exec(js))) {
    const part = m[1] || '';
    const itemRe = /\burl\s*:\s*["']([^"']+)["']/ig;
    let mi; while ((mi = itemRe.exec(part))) add(mi[1], 'openapi', 'js', 'swagger-ui');
  }
  const directOpenApiRe = /https?:\/\/[^\s"']+(?:openapi|swagger)[^\s"']*\.(?:json|ya?ml)/ig;
  while ((m = directOpenApiRe.exec(js))) add(m[0], 'openapi', 'js');
  const directWsdlRe = /https?:\/\/[^\s"']+\.(?:wsdl)/ig;
  while ((m = directWsdlRe.exec(js))) add(m[0], 'soap', 'js');
  return unique(out);
}

async function discover(url, { maxJs = 8 } = {}) {
  const res = await fetchText(url);
  const html = res.body || '';
  const base = res.url;
  const { candidates, scripts } = extractFromHtml(html, base);
  // Domain-aware hints
  const hints = [];
  try { const host = new URL(base).hostname; hints.push(host); } catch {}
  const out = [...candidates];
  // Scan up to N JS assets
  let scanned = 0;
  for (const jsUrl of scripts) {
    if (scanned >= maxJs) break;
    scanned++;
    try {
      const jsRes = await fetchText(jsUrl);
      const more = extractFromJs(jsRes.body || '', jsRes.url);
      for (const u of more) out.push(u);
    } catch (_) {}
  }
  // If wix docs, nudge likely API host patterns
  const host = (new URL(base)).hostname || '';
  if (/dev\.wix\.com$/i.test(host) && !out.length) {
    // Provide guidance if nothing was found; many Wix specs require auth.
    out.push({ url: 'https://www.wixapis.com/<product>/<vN>/swagger.json', type: 'openapi', source: 'hint', hint: 'wix-template' });
  }
  return unique(out);
}

async function main() {
  const args = parseArgs(process.argv.slice(2));
  const url = args.url || args.u;
  const maxJs = Number(args['max-js'] || 8);
  if (!url) { console.error('Usage: node examples/scripts/discover-spec-from-docs.js --url <docs_url> [--max-js 8]'); process.exit(1); }
  try {
    const list = await discover(url, { maxJs });
    console.log(JSON.stringify(list, null, 2));
  } catch (e) {
    console.error('Discovery failed:', e.message);
    process.exit(1);
  }
}

main();

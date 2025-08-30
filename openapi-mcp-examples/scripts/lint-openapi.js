/* Simple OpenAPI lint: schema presence, unique operationIds, non-empty paths, at least one server */
const fs = require('fs');
const path = require('path');
let SwaggerParser; try { SwaggerParser = require('swagger-parser'); } catch (_) { SwaggerParser = null; }

function parseArgs(argv){ const o={}; for(let i=0;i<argv.length;i++){ const t=argv[i]; if(t.startsWith('--')){ const k=t.slice(2); const v=argv[i+1]&&!argv[i+1].startsWith('--')?argv[++i]:'true'; o[k]=v;} } return o; }

async function main(){
  const args = parseArgs(process.argv.slice(2));
  const file = args.file || args.from || '';
  if (!file) { console.error('Usage: node examples/lint-openapi.js --file <path>'); process.exit(1); }
  const raw = fs.readFileSync(path.resolve(process.cwd(), file), 'utf8');
  let spec; try { spec = JSON.parse(raw); } catch (e) { console.error('Invalid JSON'); process.exit(2); }
  if (SwaggerParser) { try { await SwaggerParser.validate(spec); } catch (e) { console.error('OpenAPI validation failed:', e.message); process.exit(3); } }
  const errs = [];
  if (!spec.openapi) errs.push('Missing openapi field');
  if (!spec.paths || !Object.keys(spec.paths).length) errs.push('No paths defined');
  if (!spec.servers || !spec.servers.length) errs.push('No servers defined');
  const seen = new Set();
  if (spec.paths){
    for (const p of Object.keys(spec.paths)){
      const obj = spec.paths[p];
      for (const m of Object.keys(obj)){
        const op = obj[m];
        if (!op || typeof op !== 'object') continue;
        if (!op.operationId) errs.push(`Missing operationId for ${m.toUpperCase()} ${p}`);
        else if (seen.has(op.operationId)) errs.push(`Duplicate operationId: ${op.operationId}`);
        else seen.add(op.operationId);
      }
    }
  }
  if (errs.length){ console.error('LINT ERRORS:\n- ' + errs.join('\n- ')); process.exit(4); }
  console.log('OpenAPI OK');
}

main().catch((e)=>{ console.error('Fatal:', e.message); process.exit(5); });


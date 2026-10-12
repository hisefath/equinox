// Render docs/architecture/diagrams/*.mmd to PNG pictures at natural size, 2x, white background, via headless Chrome.
// usage: node scripts/render-diagram-pngs.mjs docs/architecture/diagrams docs/architecture/diagrams/png "$TMPDIR"
// Needs Google Chrome; loads Mermaid 11.4.1 (and its ELK layout for the diagrams listed in ELK) from jsDelivr.
// be-class and deployment use ELK: the default layout makes be-class 6,700 px wide and overlaps deployment's titles.
import { spawn } from 'node:child_process';
import { readFileSync, writeFileSync, readdirSync, mkdirSync } from 'node:fs';
const [, , srcDir, outDir, tmp] = process.argv; mkdirSync(outDir, { recursive: true });
const chrome = spawn('/Applications/Google Chrome.app/Contents/MacOS/Google Chrome', ['--headless=new', '--remote-debugging-port=9335', '--user-data-dir=' + tmp + '/prof-mmd', '--hide-scrollbars', 'about:blank'], { stdio: 'ignore' });
const sleep = ms => new Promise(r => setTimeout(r, ms));
let ws; for (let i = 0; i < 50 && !ws; i++) { await sleep(200); try { const t = await (await fetch('http://127.0.0.1:9335/json')).json(); const p = t.find(x => x.type === 'page'); if (p) ws = new WebSocket(p.webSocketDebuggerUrl); } catch {} }
await new Promise(r => ws.onopen = r); let id = 0; const pend = new Map();
ws.onmessage = m => { const d = JSON.parse(m.data); if (d.id && pend.has(d.id)) { pend.get(d.id)(d); pend.delete(d.id); } };
const send = (method, params = {}) => new Promise(r => { const i = ++id; pend.set(i, r); ws.send(JSON.stringify({ id: i, method, params })); });
const ev = async e => (await send('Runtime.evaluate', { expression: e, awaitPromise: true, returnByValue: true })).result?.result?.value;
const ELK = new Set(['be-class', 'deployment']);
const esc = s => s.replace(/&/g, '&amp;').replace(/</g, '&lt;');
for (const f of readdirSync(srcDir).filter(f => f.endsWith('.mmd')).sort()) {
  const name = f.slice(0, -4), src = readFileSync(`${srcDir}/${f}`, 'utf8');
  const html = `<!doctype html><html><head><meta charset="utf-8"><style>html,body{margin:0;background:#fff}#d{display:inline-block;padding:24px;background:#fff}</style></head><body><div id="d"><pre class="mermaid">${esc(src)}</pre></div>
<script type="module">import m from "https://cdn.jsdelivr.net/npm/mermaid@11.4.1/dist/mermaid.esm.min.mjs"; import elk from "https://cdn.jsdelivr.net/npm/@mermaid-js/layout-elk@0.1.7/dist/mermaid-layout-elk.esm.min.mjs"; m.registerLayoutLoaders(elk);
m.initialize({startOnLoad:false,${ELK.has(name) ? 'layout:"elk",markdownAutoWrap:false,' : ''}theme:"default",themeVariables:{fontSize:"16px"},flowchart:{useMaxWidth:false},sequence:{useMaxWidth:false},er:{useMaxWidth:false},class:{useMaxWidth:false},state:{useMaxWidth:false}});
try{await m.run();}catch(e){document.body.dataset.err=String(e)}
const s=document.querySelector("#d svg"); if(s){const vb=s.viewBox.baseVal; s.style.maxWidth="none"; s.setAttribute("width",vb.width); s.setAttribute("height",vb.height);} document.body.dataset.ready=1;</script></body></html>`;
  writeFileSync(`${tmp}/r_${name}.html`, html);
  await send('Emulation.setDeviceMetricsOverride', { width: 1400, height: 900, deviceScaleFactor: 2, mobile: false });
  await send('Page.navigate', { url: `file://${tmp}/r_${name}.html` });
  for (let i = 0; i < 60 && !(await ev('document.body && document.body.dataset.ready')); i++) await sleep(250);
  const err = await ev('document.body.dataset.err||""');
  const b = await ev('(()=>{const r=document.getElementById("d").getBoundingClientRect();return [r.width,r.height]})()');
  await send('Emulation.setDeviceMetricsOverride', { width: Math.ceil(b[0]), height: Math.ceil(b[1]), deviceScaleFactor: 2, mobile: false });
  await sleep(300);
  const png = await send('Page.captureScreenshot', { format: 'png', clip: { x: 0, y: 0, width: Math.ceil(b[0]), height: Math.ceil(b[1]), scale: 1 }, captureBeyondViewport: true });
  writeFileSync(`${outDir}/${name}.png`, Buffer.from(png.result.data, 'base64'));
  console.log(name, Math.ceil(b[0]) * 2 + 'x' + Math.ceil(b[1]) * 2, err);
}
ws.close(); chrome.kill();

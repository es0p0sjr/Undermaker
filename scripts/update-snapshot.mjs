// Local update coordinator. Only the explicitly run browser snippet downloads data.
import fs from 'node:fs';
import path from 'node:path';
import {fileURLToPath} from 'node:url';
import {spawn, spawnSync} from 'node:child_process';
import {createHash} from 'node:crypto';
import {deflateSync} from 'node:zlib';
import {once} from 'node:events';
import assert from 'node:assert/strict';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const read = name => JSON.parse(fs.readFileSync(name, 'utf8').replace(/^\uFEFF/, ''));
const save = (name, value) => fs.writeFileSync(name, JSON.stringify(value, null, 2) + '\n');
const hash = bytes => createHash('sha256').update(bytes).digest('hex');
const env = {...process.env, GOCACHE:path.join(root, '.go-cache'), GOPROXY:'off', GOTOOLCHAIN:'local'};

export function checkCapture(raw, english, previousRaw) {
  if (raw.capture?.source !== 'loaded game' || !Number.isFinite(Date.parse(raw.capture?.capturedAt)))
    throw new Error('Expected a loaded-game capture with its capture timestamp. Run the prepared snippet again.');
  if (Date.parse(raw.capture.capturedAt) > Date.now() + 300000)
    throw new Error('Capture timestamp is in the future; check the clock and capture again.');
  if (previousRaw?.capture?.capturedAt && Date.parse(raw.capture.capturedAt) < Date.parse(previousRaw.capture.capturedAt))
    throw new Error('This capture is older than the installed capture. Use a fresh capture; restore backups separately.');
  if (!Array.isArray(raw.cards) || !raw.cards.length) throw new Error('Capture has no cards.');
  const ids = new Set(raw.cards.map(c => c.id));
  if (ids.size !== raw.cards.length || [...ids].some(id => !Number.isSafeInteger(id) || id < 0))
    throw new Error('Capture contains invalid or duplicate card IDs.');
  if (!english || Array.isArray(english) || Object.values(english).some(v => typeof v !== 'string'))
    throw new Error('Invalid English dictionary.');
  for (const prefix of ['card-name-', 'card-']) {
    const translated = new Set(Object.keys(english).filter(k => k.startsWith(prefix) && /^\d+$/.test(k.slice(prefix.length))).map(k => Number(k.slice(prefix.length))));
    const missing = [...ids].filter(id => !translated.has(id));
    const extra = [...translated].filter(id => !ids.has(id));
    if (missing.length || extra.length)
      throw new Error(`Catalog/English mismatch (${prefix}): missing text IDs [${missing}], dictionary-only IDs [${extra}]. Reload the collection and capture again.`);
  }
}

export function compare(before, after, oldManifest = {checksums:{}}, newManifest = {checksums:{}}) {
  const old = new Map(before.map(c => [c.id, c])), next = new Map(after.map(c => [c.id, c]));
  const fields = ['name','cost','attack','health','description','rarity','extension','type','image','statuses','tribes','soul','artVariants'];
  const brief = c => ({id:c.id, name:c.name});
  const changed = [];
  for (const c of after) {
    if (!old.has(c.id)) continue;
    const changes = {};
    for (const field of fields) {
      const a = old.get(c.id)[field] ?? null, b = c[field] ?? null;
      if (JSON.stringify(a) !== JSON.stringify(b)) changes[field] = {before:a, after:b};
    }
    if (Object.keys(changes).length) changed.push({...brief(c), changes});
  }
  return {previousCount:before.length, updatedCount:after.length,
    added:after.filter(c => !old.has(c.id)).map(brief), removed:before.filter(c => !next.has(c.id)).map(brief), changed,
    changedAssets:Object.keys(newManifest.checksums).filter(k => k.startsWith('assets/') && oldManifest.checksums[k] && oldManifest.checksums[k] !== newManifest.checksums[k])};
}

// Each rename has an inverse. Keep the previous pair on disk, including a running EXE.
// A journal remains if the machine loses power; see UPDATING.md for recovery.
export function publish({data, release, candidateData, candidateRelease, backup}, rename = fs.renameSync) {
  fs.mkdirSync(backup, {recursive:true});
  const journal = path.join(backup, 'transaction.json');
  const steps = [];
  if (fs.existsSync(data)) steps.push([data, path.join(backup, 'data')]);
  if (fs.existsSync(release)) steps.push([release, path.join(backup, 'uc-print.exe')]);
  steps.push([candidateData, data], [candidateRelease, release]);
  const state = {status:'installing', steps, completed:0};
  save(journal, state);
  const completed = [];
  try {
    for (const [from, to] of steps) {
      if (fs.existsSync(to)) throw new Error(`Refusing to overwrite ${to}`);
      rename(from, to);
      completed.push([from, to]);
      state.completed = completed.length;
      save(journal, state);
    }
    state.status = 'installed';
    save(journal, state);
  } catch (error) {
    const failures = [];
    for (const [from, to] of completed.reverse()) {
      try { rename(to, from); } catch (rollback) { failures.push(rollback.message); }
    }
    state.status = failures.length ? 'recovery-required' : 'rolled-back';
    state.error = error.message;
    state.rollbackErrors = failures;
    try { save(journal, state); } catch { /* Preserve the original journal if storage failed. */ }
    throw new Error(`Install failed: ${error.message}. ${failures.length ? `Recovery required: ${journal}; ${failures.join('; ')}` : 'Previous snapshot and executable restored.'}`);
  }
}

function runGo(args, log) {
  const fd = log ? fs.openSync(log, 'w') : null;
  let result;
  try { result = spawnSync('go', args, {cwd:root, env, windowsHide:true, stdio:fd === null ? 'inherit' : ['ignore', fd, fd]}); }
  finally { if (fd !== null) fs.closeSync(fd); }
  if (result.error || result.status !== 0) {
    const tail = log ? fs.readFileSync(log, 'utf8').trim().split(/\r?\n/).slice(-12).join('\n') : '';
    throw new Error(`${result.error?.message || 'Go command failed'}. ${log ? `Log: ${log}` : 'Install Go 1.23+ and Node 18+ (with npm), then retry.'}\n${tail}`);
  }
}

export async function smoke(executable, snapshotDir, emptyDir) {
  fs.mkdirSync(emptyDir, {recursive:true});
  const expected = read(path.join(snapshotDir, 'cards.json')), manifest = read(path.join(snapshotDir, 'manifest.json'));
  // File-backed output also works where Windows restricts Node's pipe handles.
  const log = path.join(path.dirname(emptyDir), 'smoke.log');
  const fd = fs.openSync(log, 'w');
  let child;
  try { child = spawn(executable, ['--port','0'], {cwd:emptyDir, windowsHide:true, stdio:['ignore',fd,fd]}); }
  finally { fs.closeSync(fd); }
  const exited = once(child, 'exit').catch(() => []);
  try {
    const origin = await new Promise((resolve, reject) => {
      const timer = setTimeout(() => finish(new Error(`Candidate startup timed out; see ${log}`)), 60000);
      const finish = (error, value) => { clearTimeout(timer); clearInterval(poll); error ? reject(error) : resolve(value); };
      const collect = () => {
        const output = fs.readFileSync(log, 'utf8');
        const match = output.match(/UC Print offline: (http:\/\/127\.0\.0\.1:\d+)/);
        if (match) finish(null, match[1]);
      };
      const poll = setInterval(collect, 100);
      child.once('error', e => finish(e));
      child.once('exit', () => finish(new Error(`Candidate exited during startup; see ${log}`)));
    });
    const request = async (url, body) => {
      const response = await fetch(origin + url, {method:body ? 'POST' : 'GET',
        headers:{'Content-Type':'application/json', 'X-UCPrint':'1', Origin:origin},
        body:body ? JSON.stringify(body) : undefined, signal:AbortSignal.timeout(30000)});
      if (!response.ok) throw new Error(`Candidate returned HTTP ${response.status} for ${url}`);
      return response;
    };
    const actual = await (await request('/api/cards')).json();
    assert.deepEqual(actual.cards, expected); assert.deepEqual(actual.manifest, manifest);
    for (const card of expected) {
      const bytes = Buffer.from(await (await request(`/api/art/${card.id}`)).arrayBuffer());
      assert.equal(hash(bytes), manifest.checksums[`assets/cards/${card.image}`]);
    }
    for (const [name, digest] of Object.entries(manifest.checksums)) {
      if (name.startsWith('assets/powers/')) {
        const bytes = Buffer.from(await (await request('/api/power/' + encodeURIComponent(path.basename(name)))).arrayBuffer());
        assert.equal(hash(bytes), digest);
      }
    }
    for (const url of ['/', '/app.js', '/renderer.js', '/renderer-assets.json']) await (await request(url)).arrayBuffer();
    const id = expected[0].id;
    const deck = await (await request('/api/deck', {code:JSON.stringify({soul:'KINDNESS',cardIds:Array(25).fill(id),artifactIds:[]})})).json();
    assert.equal(deck.queue.reduce((n, c) => n + c.quantity, 0), 25);
    const rgb = deflateSync(Buffer.alloc(744 * 1039 * 3, 120)).toString('base64');
    const exported = await (await request('/api/export', {cards:[{id,quantity:10,rgb}],cutMarks:true})).json();
    const pdf = Buffer.from(await (await request(exported.url)).arrayBuffer());
    assert.equal(exported.pages, 2); assert.equal(pdf.subarray(0,4).toString(), '%PDF');
    assert.equal([...pdf.toString('latin1').matchAll(/\/Type \/Page\b/g)].length, 2);
    console.log(`PASS: Standalone release, ${expected.length} cards, all standard artwork/status icons, deck import and PDF export.`);
  } finally {
    if (child.exitCode === null) child.kill();
    await exited;
  }
}

async function main(args) {
  if (args.length === 1 && args[0] === '--prepare') {
    runGo(['run','./tools/ucsync','browser-capture']);
    console.log('Next: open the Undercards collection, run the snippet, and save into JSON/update. See UPDATING.md.');
    return;
  }
  let capture, checkOnly = false, allowRemovals = false;
  for (let i = 0; i < args.length; i++) {
    if (args[i] === '--capture' && args[i+1] && !args[i+1].startsWith('--')) capture = path.resolve(args[++i]);
    else if (args[i] === '--check-only') checkOnly = true;
    else if (args[i] === '--allow-removals') allowRemovals = true;
    else throw new Error(`Unknown/incomplete argument: ${args[i]}`);
  }
  if (!capture) throw new Error('Use --prepare or --capture PATH [--check-only] [--allow-removals]. See UPDATING.md.');
  const data = path.join(root, 'data'), release = path.join(root, 'dist/uc-print.exe');
  const backups = path.join(root, 'snapshot-backups');
  if (fs.existsSync(backups)) for (const name of fs.readdirSync(backups)) {
    const journal = path.join(backups, name, 'transaction.json');
    if (fs.existsSync(journal) && !['installed','rolled-back'].includes(read(journal).status))
      throw new Error(`An interrupted update requires recovery first: ${journal}. See UPDATING.md.`);
  }
  const lock = path.join(root, '.data.ucsync.lock');
  let lockFD;
  try { lockFD = fs.openSync(lock, 'wx'); }
  catch { throw new Error(`Another update/import may be running (${lock}). See UPDATING.md for stale-lock recovery.`); }
  try {
    const current = fs.existsSync(data);
    if (current) runGo(['run','./tools/ucsync','validate','--data',data]);
    const before = current ? read(path.join(data,'cards.json')) : [];
    if (before.some(c => c.artVariants?.length))
      throw new Error('Installed snapshot has custom artwork variants. This standard-art updater stops to preserve them; see UPDATING.md.');
    const raw = read(path.join(capture,'AllCards.json')), english = read(path.join(capture,'en.json'));
    checkCapture(raw, english, current ? read(path.join(data,'raw/AllCards.json')) : null);
    const id = new Date().toISOString().replace(/[:.]/g,'-') + '-' + process.pid;
    const work = path.join(root,'.build/snapshot-updates',id);
    fs.mkdirSync(work,{recursive:true});
    const candidateData = path.join(work,'data'), candidateRelease = path.join(work,'uc-print.exe');
    console.log(`Validating capture offline. Logs and comparison: ${work}`);
    runGo(['run','./tools/ucsync','import',path.join(capture,'AllCards.json'),'--translations',path.join(capture,'en.json'),
      '--assets',path.join(capture,'assets'),'--data',candidateData],path.join(work,'import.log'));
    const report = {capture, capturedAt:raw.capture.capturedAt, ...compare(before,read(path.join(candidateData,'cards.json')),
      current ? read(path.join(data,'manifest.json')) : undefined,read(path.join(candidateData,'manifest.json')))};
    save(path.join(work,'comparison.json'),report);
    console.log(`${report.previousCount} -> ${report.updatedCount} cards: ${report.added.length} added, ${report.changed.length} changed, ${report.removed.length} removed; ${report.changedAssets.length} existing images changed.`);
    if (checkOnly) { console.log('CHECK COMPLETE: installed snapshot and executable unchanged.'); return; }
    if (report.removed.length && !allowRemovals)
      throw new Error('Cards would be removed. Review comparison.json; recapture if incomplete, or use -AllowRemovals for verified intentional removals.');
    console.log('Building candidate release (offline)...');
    runGo(['run','./tools/ucbuild','--data',candidateData,'--output',candidateRelease],path.join(work,'build.log'));
    console.log('Testing candidate release from an empty working folder...');
    await smoke(candidateRelease,candidateData,path.join(work,'smoke'));
    const backup = path.join(backups,id);
    fs.mkdirSync(backup,{recursive:true}); fs.mkdirSync(path.dirname(release),{recursive:true});
    save(path.join(backup,'comparison.json'),report);
    publish({data,release,candidateData,candidateRelease,backup});
    console.log(`UPDATE COMPLETE: ${report.updatedCount} cards. Backup and report: ${backup}`);
    console.log('Close the old app and run start.bat. An already-running app continues using its old snapshot.');
  } finally { fs.closeSync(lockFD); fs.unlinkSync(lock); }
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  main(process.argv.slice(2)).catch(error => { console.error('UPDATE STOPPED:', error.message); process.exitCode = 1; });
}

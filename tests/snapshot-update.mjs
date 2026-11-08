import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import {checkCapture, compare, publish, smoke} from '../scripts/update-snapshot.mjs';

const raw = {capture:{source:'loaded game',capturedAt:'2026-01-02T00:00:00Z'},cards:[{id:1},{id:2}]};
const english = {'card-name-1':'One','card-1':'Rules','card-name-2':'Two','card-2':''};
checkCapture(raw,english);
assert.throws(() => checkCapture({...raw,cards:[{id:1}]},english),/dictionary-only IDs \[2\]/);
assert.throws(() => checkCapture(raw,{'card-name-1':'One'}),/missing text IDs/);
assert.throws(() => checkCapture({...raw,cards:[{id:1},{id:1}]},english),/duplicate/);
assert.throws(() => checkCapture({cards:raw.cards},english),/timestamp/);
assert.throws(() => checkCapture(raw,english,{capture:{capturedAt:'2026-01-03T00:00:00Z'}}),/older/);
assert.throws(() => checkCapture({...raw,capture:{...raw.capture,capturedAt:'2999-01-01T00:00:00Z'}},english),/future/);

const report = compare([{id:1,name:'One',cost:2,raw:{temporary:true}},{id:2,name:'Two'}],
  [{id:1,name:'One',cost:3,raw:{temporary:false}},{id:3,name:'Three'}],
  {checksums:{'assets/cards/one.png':'old','raw/en.json':'old'}},
  {checksums:{'assets/cards/one.png':'new','raw/en.json':'new'}});
assert.deepEqual(report.added,[{id:3,name:'Three'}]);
assert.deepEqual(report.removed,[{id:2,name:'Two'}]);
assert.deepEqual(report.changed,[{id:1,name:'One',changes:{cost:{before:2,after:3}}}]);
assert.deepEqual(report.changedAssets,['assets/cards/one.png']);

const parent = path.resolve('.test-data'); fs.mkdirSync(parent,{recursive:true});
const temp = fs.mkdtempSync(path.join(parent,'snapshot-update-'));
function fixture(name, existing = true) {
  const base = path.join(temp,name); fs.mkdirSync(base);
  const paths = Object.fromEntries(['data','release','candidateData','candidateRelease','backup'].map(k => [k,path.join(base,k)]));
  fs.mkdirSync(paths.candidateData); fs.writeFileSync(path.join(paths.candidateData,'marker'),'new data');
  fs.writeFileSync(paths.candidateRelease,'new executable');
  if (existing) {
    fs.mkdirSync(paths.data); fs.writeFileSync(path.join(paths.data,'marker'),'old data');
    fs.writeFileSync(paths.release,'old executable');
  }
  return paths;
}
try {
  const missingExecutable = fixture('missing-executable');
  fs.writeFileSync(path.join(missingExecutable.candidateData,'cards.json'),'[]');
  fs.writeFileSync(path.join(missingExecutable.candidateData,'manifest.json'),'{}');
  await assert.rejects(smoke(path.join(temp,'missing.exe'),missingExecutable.candidateData,path.join(temp,'empty')),/ENOENT/);
  for (let failAt = 1; failAt <= 4; failAt++) {
    const paths = fixture('failure-' + failAt); let calls = 0;
    assert.throws(() => publish(paths,(from,to) => {
      if (++calls === failAt) throw new Error('simulated locked file');
      fs.renameSync(from,to);
    }),/Previous snapshot and executable restored/);
    assert.equal(fs.readFileSync(path.join(paths.data,'marker'),'utf8'),'old data');
    assert.equal(fs.readFileSync(paths.release,'utf8'),'old executable');
    assert.equal(fs.readFileSync(path.join(paths.candidateData,'marker'),'utf8'),'new data');
    assert.equal(fs.readFileSync(paths.candidateRelease,'utf8'),'new executable');
    assert.equal(JSON.parse(fs.readFileSync(path.join(paths.backup,'transaction.json'))).status,'rolled-back');
  }
  const failedRecovery = fixture('failed-recovery'); let calls = 0;
  assert.throws(() => publish(failedRecovery,(from,to) => {
    if (++calls >= 3) throw new Error('simulated disk failure');
    fs.renameSync(from,to);
  }),/Recovery required/);
  assert.equal(JSON.parse(fs.readFileSync(path.join(failedRecovery.backup,'transaction.json'))).status,'recovery-required');
  assert.equal(fs.readFileSync(path.join(failedRecovery.backup,'data/marker'),'utf8'),'old data');
  assert.equal(fs.readFileSync(path.join(failedRecovery.backup,'uc-print.exe'),'utf8'),'old executable');
  for (const existing of [true,false]) {
    const paths = fixture('success-' + existing,existing);
    publish(paths);
    assert.equal(fs.readFileSync(path.join(paths.data,'marker'),'utf8'),'new data');
    assert.equal(fs.readFileSync(paths.release,'utf8'),'new executable');
    if (existing) {
      assert.equal(fs.readFileSync(path.join(paths.backup,'data/marker'),'utf8'),'old data');
      assert.equal(fs.readFileSync(path.join(paths.backup,'uc-print.exe'),'utf8'),'old executable');
    }
    assert.equal(JSON.parse(fs.readFileSync(path.join(paths.backup,'transaction.json'))).status,'installed');
  }
} finally {
  assert.equal(path.dirname(temp),parent);
  fs.rmSync(temp,{recursive:true,force:true});
}
console.log('PASS: Incomplete/stale captures rejected; comparison reports changes; every failed install step rolls back; recovery failures retain backups and journal; complete pairs install successfully.');

import fs from 'node:fs';
import vm from 'node:vm';
import assert from 'node:assert/strict';

const code = fs.readFileSync(new URL('../tools/ucsync/browser-data.js', import.meta.url), 'utf8');
const context = vm.createContext({TextDecoder});
vm.runInContext(code.replace('\nucBrowserData();', ''), context);
const read = context.ucFetchSnapshotJSON;
const raw = fs.readFileSync(new URL('../JSON/AllCards.Json', import.meta.url));
const english = fs.readFileSync(new URL('../JSON/en.json', import.meta.url));
function response(bytes, status = 200) {
  return {ok: status === 200, status, arrayBuffer: async () => Uint8Array.from(bytes).buffer};
}
const requested = [];
const result = await read(async path => {
  requested.push(path);
  return response(path === '/AllCards' ? raw : english);
});
assert.deepEqual(requested, ['/AllCards', '/translation/en.json']);
assert.equal(result.cardCount, JSON.parse(JSON.parse(raw).cards).length);
assert.equal(Buffer.compare(Buffer.from(result.files[0][1]), raw), 0);
assert.equal(Buffer.compare(Buffer.from(result.files[1][1]), english), 0);
await assert.rejects(read(async () => response([], 403)), /HTTP 403/);
await assert.rejects(read(async () => response(Buffer.from("<html>You aren't allowed to view this page</html>"))), /expected JSON/);
await assert.rejects(read(async () => response(Buffer.from('{"cards":[]}'))), /non-empty card database/);
await assert.rejects(read(async path => response(path === '/AllCards' ? raw : Buffer.from('{"error":"denied"}'))), /translation dictionary/);
console.log('PASS: Saved official format accepted; exact response bytes preserved.');
console.log('PASS: HTTP errors, HTML denials, empty card lists and invalid translations rejected.');

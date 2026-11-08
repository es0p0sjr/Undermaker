// No real browser/network/dataset is needed for these acquisition checks.
import fs from 'node:fs';
import vm from 'node:vm';
import assert from 'node:assert/strict';
const code=fs.readFileSync(new URL('../tools/ucsync/browser-assets.js',import.meta.url),'utf8');
const context=vm.createContext({});
vm.runInContext(code.replace('ucBrowserSync(/*UC_CONFIG*/null);',''),context);
const collect=context.ucCollectAssets;
const valid=new Blob(['valid PNG fixture']),corrupt=new Blob(['bad']);
function setup(initial={}) {
  const files=new Map(Object.entries(initial)),requests=[],writes=[];
  const root={async getDirectoryHandle(category){return {
    async getFileHandle(name,options={}) {
      const key=`${category}/${name}`;
      if(!files.has(key)&&!options.create)throw Object.assign(new Error('missing'),{name:'NotFoundError'});
      return {getFile:async()=>files.get(key),createWritable:async()=>{
        let pending;
        return {write:async blob=>{pending=blob;},close:async()=>{files.set(key,pending);writes.push(key);},abort:async()=>{}};
      }};
    }
  };}};
  const io={stopped:()=>false,progress:()=>{},delay:async()=>{},validate:async blob=>{if(await blob.text()!=='valid PNG fixture')throw new Error('invalid image');},
    fetch:async path=>{requests.push(path);return {ok:true,status:200,blob:async()=>valid};}};
  return {files,requests,writes,root,io};
}
let f=setup({'cards/saved.png':valid});
const result=await collect({assets:['cards/saved.png','powers/Taunt.png']},f.root,f.io);
assert.equal(result.reused,1);assert.equal(result.downloaded,1);
assert.deepEqual(f.requests,['/images/powers/Taunt.png']);
console.log('PASS: Resumes valid images; only missing references are fetched.');
f=setup();
f.io.fetch=async path=>({ok:!path.includes('blocked'),status:path.includes('blocked')?403:200,blob:async()=>valid});
await assert.rejects(collect({assets:['cards/first.png','cards/blocked.png','cards/never.png']},f.root,f.io),/HTTP 403/);
assert.deepEqual(f.writes,['cards/first.png']);
assert.equal(f.files.has('cards/blocked.png'),false);
console.log('PASS: 403 stops the batch and preserves completed images.');
f=setup({'cards/old.png':corrupt});
f.io.fetch=async()=>({ok:true,blob:async()=>corrupt});
await assert.rejects(collect({assets:['cards/old.png']},f.root,f.io),/invalid image/);
assert.equal(f.files.get('cards/old.png'),corrupt);assert.equal(f.writes.length,0);
console.log('PASS: Invalid download cannot overwrite an existing file.');
f=setup();f.io.stopped=()=>true;
await assert.rejects(collect({assets:['cards/test.png']},f.root,f.io),/Stopped/);
f.io.stopped=()=>false;
await assert.rejects(collect({assets:['cards/../../escape.png']},f.root,f.io),/Invalid asset path/);
assert.equal(f.requests.length,0);
console.log('PASS: Stop and unsafe-path checks run before downloading.');

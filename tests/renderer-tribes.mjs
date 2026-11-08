import fs from 'node:fs';
import assert from 'node:assert/strict';
const source=fs.readFileSync(new URL('../web/renderer.js',import.meta.url),'utf8');
const {resolveTribes}=await import(`data:text/javascript;base64,${Buffer.from(source).toString('base64')}`);
const bundle=JSON.parse(fs.readFileSync(new URL('../web/renderer-assets.json',import.meta.url),'utf8'));
for(const [key,uri] of Object.entries(bundle.tribes)) {
  assert.deepEqual(Buffer.from(uri.split(',')[1],'base64'),fs.readFileSync(new URL(`../Sources/CardEditor/public/resources/tribes/${key}.png`,import.meta.url)));
}
assert.deepEqual(resolveTribes([],bundle.tribes),[]);
assert.deepEqual(resolveTribes(['Tem','Frog','All monster tribes','Royal Guard'],bundle.tribes).map(b=>b.key),['TEMMIE','FROGGIT','ALL','ROYAL_GUARD']);
assert.deepEqual(resolveTribes(['snail','Snail','Dog'],bundle.tribes).map(b=>b.key),['SNAIL','DOG']);
assert.throws(()=>resolveTribes(['Unknown tribe'],bundle.tribes),/Missing local tribe symbol/);
assert.throws(()=>resolveTribes(['Snail']),/Missing local tribe symbol/);
assert.throws(()=>resolveTribes(Object.keys(bundle.tribes).slice(0,9),bundle.tribes),/Too many tribe icons/);
const cards=JSON.parse(fs.readFileSync(new URL('../data/cards.json',import.meta.url),'utf8'));
const reworks=JSON.parse(fs.readFileSync(new URL('../ReworkedCards/reworks.json',import.meta.url),'utf8'));
for(const card of [...cards,...reworks.map(r=>r.card)]) {
  const badges=resolveTribes(card.tribes,bundle.tribes);
  assert.equal(badges.length,new Set(card.tribes).size,card.name);
}
console.log(`PASS: All ${Object.keys(bundle.tribes).length} original tribe icons preserved; every official/reworked tribe resolves, including aliases and multiple tribes.`);

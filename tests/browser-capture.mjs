import fs from 'node:fs';
import vm from 'node:vm';
import assert from 'node:assert/strict';
const code = fs.readFileSync(new URL('../tools/ucsync/browser-capture.js', import.meta.url), 'utf8');
const context = vm.createContext({});
vm.runInContext(code.replace('\nucCaptureLoadedGame();', ''), context);
const capture = context.ucLoadedCatalog;
const cards = [
  {id:1, fixedId:1, cost:3, image:'First', rarity:'COMMON', typeCard:0, attack:2, hp:4, statuses:[{name:'Taunt'}]},
  {id:2, fixedId:2, cost:0, image:'Token.png', rarity:'TOKEN', typeCard:1, statuses:[{name:'Armor', image:'Armor.jpg'}]}
];
for (const value of [cards, {cards:JSON.stringify(cards)}, {allCards:cards}, JSON.stringify(cards)]) {
  const result = capture(value);
  assert.deepEqual(JSON.parse(JSON.stringify(result.cards)), cards);
  assert.deepEqual(Array.from(result.assets), ['cards/First.png','cards/Token.png','powers/Armor.jpg','powers/Taunt.png']);
  result.cards[0].cost = 99;
  assert.equal(cards[0].cost,3);
}
for (const value of [null, [], [{...cards[0], image:'../../escape.png'}], [cards[0],cards[0]], [{...cards[0],statuses:{}}]]) {
  assert.throws(() => capture(value));
}
// The acquisition routine must not depend on the refused endpoint or stale storage.
assert.doesNotMatch(code, /fetch\(['"]\/AllCards|localStorage|sessionStorage/);
console.log('PASS: Loaded catalog captured without changing it; tokens, stats and exact asset references retained.');
console.log('PASS: Empty catalogs, duplicate IDs, invalid status lists and unsafe asset names rejected.');

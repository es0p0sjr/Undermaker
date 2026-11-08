import {renderCard, compressedRGB, png300, clearArtwork} from './renderer.js';
import {renderCompleteCard} from './complete-card.js';

const $ = id => document.getElementById(id);
let cards=[], byId=new Map(), queue=new Map(), metadata={}, deckMeta=null, limit=40, busy=false;
let prepared=[], page=0, activeCard=null, activeRender=null;
const renderCache=new Map();
let artChoices=new Map();
function chosenArt(card) { return (card.artVariants || []).find(v=>v.id===artChoices.get(card.id)); }
function artURL(card) { if(card.completeImage)return card.completeImage;const v=chosenArt(card);return `/api/art/${card.id}${v?'?variant='+encodeURIComponent(v.id):''}`; }
// Discard the prior imported deck selection when switching to the fresh source.
const QUEUE_KEY='uc-print-queue-snapshot-v1';
try { localStorage.removeItem('uc-print-queue'); } catch { /* Storage can be disabled. */ }

function notice(message,error=false) { $('notice').textContent=message; $('notice').classList.toggle('error',error); }
async function api(path,body) {
  const response=await fetch(path,body===undefined ? {} : {method:'POST',headers:{'Content-Type':'application/json','X-UCPrint':'1'},body:JSON.stringify(body)});
  const data=await response.json();
  if(!response.ok) {
    throw new Error(data.error || 'Request failed.');
  }
  return data;
}
function operation(fn) { return async (...args)=>{
  if(busy) return;
  busy=true; document.body.classList.add('busy');
  const disabled=[...document.querySelectorAll('button,input,select,textarea')].map(el=>[el,el.disabled]);
  disabled.forEach(([el])=>el.disabled=true);
  try { await fn(...args); } catch(error) {
    notice(error.message,true);
  }
  finally {
    busy=false;document.body.classList.remove('busy'); disabled.forEach(([el,was])=>el.disabled=was); updateExportButtons();
    if(activeCard) $('art-variant').disabled=!(activeCard.artVariants || []).length;
    if(prepared.length) {
      $('prev-page').disabled=page===0;
      $('next-page').disabled=page>=Math.ceil(prepared.reduce((n,item)=>n+item.quantity,0)/9)-1;
    }
  }
}; }
function isToken(card) { return ['TOKEN','GENERATED'].includes(card.rarity); }
function total() { return [...queue.values()].reduce((a,b)=>a+b,0); }
function updateExportButtons() { $('preview').disabled=$('export').disabled=busy || !queue.size; }
function save() {
  try { localStorage.setItem(QUEUE_KEY,JSON.stringify({queue:[...queue],deckMeta,artChoices:[...artChoices],reworksImported:cards.some(c=>c.completeImage)})); } catch { /* The app also works with storage disabled. */ }
}
function invalidate() {
  prepared=[]; $('sheets-panel').hidden=true; $('export-result').replaceChildren(); save();
}
function setData(data,restore=false) {
  cards=data.cards; metadata={...data.metadata,demo:!!data.manifest?.demo}; byId=new Map(cards.map(card=>[card.id,card]));
  renderCache.clear(); clearArtwork();
  if(restore) {
    try {
      const saved=JSON.parse(localStorage.getItem(QUEUE_KEY) || '{}');
      if(Array.isArray(saved.artChoices)) artChoices=new Map(saved.artChoices.filter(([id,v])=>(byId.get(id)?.artVariants || []).some(art=>art.id===v)));
      if(Array.isArray(saved.queue)) {
        let count=0;
        for(const [id,q] of saved.queue) {
          if(!Number.isInteger(id) || !Number.isInteger(q) || q<1 || (count+=q)>250) throw new Error('Invalid saved queue');
        }
        queue=new Map(saved.queue); deckMeta=saved.deckMeta || null;
      }
    } catch { queue.clear(); deckMeta=null; }
  }
  $('source').textContent=data.manifest?.demo?'DEMO SNAPSHOT':'OFFLINE · CARDEDITOR STYLE';
  $('snapshot-info').textContent=`Snapshot: ${data.manifest?.downloadedAt || metadata.updated} / ${cards.length} cards / ${data.manifest?.imageCount || 0} images`;
  const updated=metadata.updated ? ` Last updated ${new Date(metadata.updated).toLocaleString()}.` : '';
  const coverage=data.coverage;
  notice(data.message || `${cards.length} cards in the library.${coverage ? ` ${coverage.rulesReady} with resolved rules; ${coverage.artworkCached} with saved artwork.` : ''}${updated} Load a deck code to choose your print cards.`);
  $('rarity').replaceChildren(new Option('All rarities','ALL'), ...[...new Set(cards.map(c=>c.rarity))].sort().map(r=>new Option(r,r)));
  limit=40; renderResults(); renderQueue(); invalidate();
}
function cardInfo(card,showRules=false) {
  const info=document.createElement('div');info.className='card-info';
  const name=document.createElement('button');name.className='card-name';name.textContent=card.name;
  name.onclick=operation(()=>showCard(card));
  const details=document.createElement('small');details.textContent=`${card.cost ?? '?'} G · ${card.type} · ${card.rarity} · #${card.originalId ?? card.id}${card.issues.length?' · DATA INCOMPLETE':''}`;
  info.append(name,details);
  if(card.completeImage) { const label=document.createElement('small');label.textContent='Reworked · complete PNG';info.append(label); }
  if(chosenArt(card)) { const label=document.createElement('small');label.textContent=`Artwork: ${chosenArt(card).name}`;info.append(label); }
  if(showRules) {
    const rules=document.createElement('p');rules.className='rules-snippet';rules.textContent=card.description || (card.issues.length?'Rules text unavailable.':'No rules text.');rules.title=rules.textContent;info.append(rules);
  }
  return info;
}
function renderResults() {
  const search=$('search').value.trim().toLowerCase(), type=$('filter').value, rarity=$('rarity').value;
  const results=cards.filter(c=>(!search || `${c.name} ${c.id} ${c.description}`.toLowerCase().includes(search)) &&
    ($('library-source').value==='ALL' || ($('library-source').value==='REWORKED')===!!c.completeImage) &&
    (type==='ALL' || (type==='TOKEN'?isToken(c):c.type===type)) && (rarity==='ALL' || c.rarity===rarity));
  $('results-count').textContent=`${results.length} cards found`;
  $('results').replaceChildren(); $('more').hidden=results.length<=limit;
  for(const card of results.slice(0,limit)) {
    const row=document.createElement('div');row.className='card-row';
    const icon=document.createElement(card.artworkCached?'button':'span');icon.className='mini-icon'+(card.type==='SPELL'?' spell':'');
    if(card.artworkCached) {
      icon.classList.add('art-thumbnail');icon.setAttribute('aria-label',`Preview ${card.name}`);icon.onclick=operation(()=>showCard(card));
      const img=document.createElement('img');img.src=artURL(card);img.alt=card.name;img.loading='lazy';
      img.onerror=()=>{icon.textContent='Error';notice(card.completeImage?`Missing PNG for ${card.name}. Import the reworked PNGs again.`:`Missing local artwork for ${card.name}. Rebuild the snapshot.`,true);};icon.append(img);
    } else icon.textContent=card.cost ?? '?';
    const quantity=document.createElement('input');quantity.type='number';quantity.min=1;quantity.max=250;quantity.value=1;quantity.setAttribute('aria-label',`Quantity of ${card.name}`);
    const add=document.createElement('button');add.textContent='Add';add.setAttribute('aria-label',`Add ${card.name}`);
    add.onclick=()=>{
      if(busy) return;
      const q=Number(quantity.value);
      if(!Number.isInteger(q) || q<1 || total()+q>250) return notice('Choose a whole quantity from 1 to 250. The print queue can hold up to 250 cards.',true);
      queue.set(card.id,(queue.get(card.id)||0)+q); invalidate();renderQueue();
    };
    row.append(icon,cardInfo(card,true),quantity,add);$('results').append(row);
  }
  if(!results.length) { const p=document.createElement('p');p.className='empty';p.textContent='No matching cards. Try another name or filter.';$('results').append(p); }
}
function renderQueue() {
  const root=$('queue');root.replaceChildren();
  if(!queue.size) { const p=document.createElement('p');p.className='empty';p.textContent='Your print queue is empty. Add cards or load your deck code.';root.append(p); }
  for(const token of [false,true]) {
    const entries=[...queue].filter(([id])=>isToken(byId.get(id)||{})===token);
    if(!entries.length) continue;
    const title=document.createElement('h3');title.textContent=token?'Generated / token pool':'Deck cards';root.append(title);
    for(const [id,q] of entries) {
      const card=byId.get(id),row=document.createElement('div');row.className='card-row';
      if(card) row.append(cardInfo(card)); else { const missing=document.createElement('span');missing.className='card-info';missing.textContent=id<0?`Reworked card #${-id} unavailable — import the reworked PNGs again`:`Unknown card #${id} — rebuild with the matching snapshot`;row.append(missing); }
      const input=document.createElement('input');input.type='number';input.min=1;input.max=250;input.value=q;input.setAttribute('aria-label',`Print quantity of ${card?.name || id}`);
      input.onchange=()=>{
        if(busy) return;
        const value=Number(input.value);
        if(!Number.isInteger(value)||value<1||total()-q+value>250) { input.value=q;return notice('Use whole quantities from 1 to 250, up to 250 total cards.',true); }
        queue.set(id,value);invalidate();renderQueue();
      };
      const remove=document.createElement('button');remove.textContent='×';remove.setAttribute('aria-label',`Remove ${card?.name || id}`);
      remove.onclick=()=>{if(!busy) {queue.delete(id);invalidate();renderQueue();}};
      row.append(input,remove);root.append(row);
    }
  }
  const tokens=[...queue].reduce((n,[id,q])=>n+(isToken(byId.get(id)||{})?q:0),0), count=total();
  $('summary').textContent=`${count} cards · ${Math.ceil(count/9)} A4 sheets${tokens?` · ${tokens} in token pool`:''}`;
  $('deck-meta').textContent=deckMeta?`${deckMeta.soul || 'No SOUL supplied'}${deckMeta.artifactIds.length?' · Artifact IDs: '+deckMeta.artifactIds.join(', '):''}. ${deckMeta.warnings.join(' ')}`:'';
  updateExportButtons();
}
async function rendered(card) {
  const variant=chosenArt(card),key=JSON.stringify([card.id,variant?.id || '']);
  if(!renderCache.has(key)) {
    try { renderCache.set(key,await (card.completeImage?renderCompleteCard(card):renderCard(card,variant?.id))); }
    catch(error) {throw new Error(`${card.name}: ${error.message}`);}
  }
  return renderCache.get(key);
}
async function showCard(card) {
  activeCard=card;activeRender=await rendered(card);
  $('card-preview').replaceChildren(activeRender.canvas.cloneNode());
  $('card-preview').firstChild.getContext('2d').drawImage(activeRender.canvas,0,0);
  $('card-notes').textContent=activeRender.warnings.join(' ');
  $('raw').textContent=JSON.stringify(card.raw,null,2);
  $('art-variant').replaceChildren(new Option('Standard artwork',''),...(card.artVariants || []).map(v=>new Option(`${v.name}${v.layout==='full'?' (full art)':''}`,v.id)));
  $('art-variant').value=chosenArt(card)?.id || '';
  $('art-variant').disabled=!(card.artVariants || []).length;
  $('art-hint').textContent=(card.artVariants || []).length?'This artwork is used for all copies of this card in your queue, PNG and PDF.':'No alternate artwork is installed for this card. Import a skin catalog and its images to add choices.';
  if(card.completeImage) $('art-hint').textContent='Complete reworked PNG. The original image, including its text and frame, is printed as supplied.';
  if(!$('card-dialog').open) $('card-dialog').showModal();
}
function download(blob,name) {
  const url=URL.createObjectURL(blob), link=document.createElement('a');link.href=url;link.download=name;link.click();setTimeout(()=>URL.revokeObjectURL(url),30000);
}
function assertPrintable(card,result) {
  if(card.issues.length) throw new Error(`${card.name}: ${card.issues.join(' ')}`);

}
async function prepare(forExport=false) {
  if(!queue.size) throw new Error('Add at least one card.');
  const out=[],warnings=[];
  for(const [id,quantity] of queue) {
    const card=byId.get(id);
    if(!card) throw new Error(id<0?`Reworked card #${-id} is unavailable. Import the reworked PNGs again; no cards will be omitted.`:`Unknown card #${id}. Import its database; no cards will be omitted.`);
    notice(`Preparing ${card.name}…`);
    const result=await rendered(card);
    if(forExport) assertPrintable(card,result);
    warnings.push(...result.warnings.map(w=>`${card.name}: ${w}`));
    out.push({card,quantity,...result});
  }
  prepared=out; page=0; showSheet();
  $('preview-notes').textContent=[metadata.demo?'DEMO: Fictional cards for print testing.':'',...new Set(warnings)].filter(Boolean).join(' ');
  return out;
}
function showSheet() {
  const expanded=prepared.flatMap(item=>Array(item.quantity).fill(item));
  const pages=Math.ceil(expanded.length/9);page=Math.max(0,Math.min(page,pages-1));
  $('sheet').replaceChildren();
  const markLines=[];
  expanded.slice(page*9,page*9+9).forEach((item,slot)=>{
    const x=7.5+(slot%3)*66,y=13.5+Math.floor(slot/3)*91;
    const canvas=item.canvas.cloneNode();canvas.getContext('2d').drawImage(item.canvas,0,0);
    canvas.style.left=`${x/210*100}%`;canvas.style.top=`${y/297*100}%`;$('sheet').append(canvas);
    for(const xx of [x,x+63]) for(const yy of [y,y+88]) {
      const dx=xx===x?-1:1,dy=yy===y?-1:1;
      markLines.push(`<path d="M${xx+dx*.35},${yy} H${xx+dx*1.3} M${xx},${yy+dy*.35} V${yy+dy*1.3}"/>`);
    }
  });
  if($('cut-marks').checked) {
    const svg=document.createElementNS('http://www.w3.org/2000/svg','svg');svg.setAttribute('viewBox','0 0 210 297');
    svg.innerHTML=`<g stroke="black" stroke-width="0.1" fill="none">${markLines.join('')}</g>`;$('sheet').append(svg);
  }
  $('sheets-panel').hidden=false;$('page-label').textContent=`Sheet ${page+1} of ${pages}`;
  $('prev-page').disabled=page===0;$('next-page').disabled=page>=pages-1;
}

$('search').oninput=$('filter').onchange=$('rarity').onchange=$('library-source').onchange=()=>{limit=40;renderResults();};
$('import-reworks').onclick=operation(async()=>{
  const data=await api('/api/reworked-cards');
  cards=[...cards.filter(c=>!c.completeImage),...data.cards];
  byId=new Map(cards.map(c=>[c.id,c]));renderCache.clear();
  $('rarity').replaceChildren(new Option('All rarities','ALL'),...[...new Set(cards.map(c=>c.rarity))].sort().map(r=>new Option(r,r)));
  $('library-source').value='REWORKED';$('search').value='';$('filter').value='ALL';limit=40;
  invalidate();renderResults();renderQueue();
  notice(`Imported ${data.cards.length} complete PNG cards from ReworkedCards/reworks.json. Add cards individually or choose One of each rework.`);
});
$('add-reworks').onclick=()=>{
  if(busy)return;
  const reworks=cards.filter(c=>c.completeImage);
  if(!reworks.length)return notice('Import the reworked PNGs first.',true);
  if(total()+reworks.length>250)return notice('The print queue can hold up to 250 cards.',true);
  for(const card of reworks)queue.set(card.id,(queue.get(card.id)||0)+1);
  invalidate();renderQueue();notice(`Added one copy of each of the ${reworks.length} reworked cards.`);
};
$('more').onclick=()=>{limit+=40;renderResults();};
$('clear').onclick=()=>{queue.clear();deckMeta=null;invalidate();renderQueue();};
$('load-deck').onclick=operation(async()=>{
  const deck=await api('/api/deck',{code:$('deck-code').value});
  queue=new Map(deck.queue.map(item=>[item.id,item.quantity]));deckMeta=deck;invalidate();renderQueue();
  notice(`Loaded ${total()} cards. ${deck.warnings.join(' ')}`);
});
$('preview').onclick=operation(async()=>{await prepare();notice('Preview ready. Check all sheets and any warnings below.');$('sheets-panel').scrollIntoView({behavior:'smooth'});});
$('export').onclick=operation(async()=>{
  const items=await prepare(true), entries=[];
  for(const item of items) {notice(`Exporting ${item.card.name}…`);entries.push({id:item.card.id,quantity:item.quantity,rgb:await compressedRGB(item.canvas)});}
  const result=await api('/api/export',{cards:entries,cutMarks:$('cut-marks').checked});
  const link=document.createElement('a');link.href=result.url;link.download=result.filename;link.textContent=`Download PDF · ${result.cards} cards · ${result.pages} sheets`;
  const preview=document.createElement('a');preview.href=result.url;preview.target='_blank';preview.rel='noopener';preview.textContent='Open PDF';
  $('export-result').replaceChildren(link,document.createTextNode(' · '),preview);link.click();
  notice(`PDF ready: ${result.pages} A4 sheets. Save the download before closing the app. Print at Actual Size / 100%.`);
});
$('close-dialog').onclick=()=>$('card-dialog').close();
$('art-variant').onchange=operation(async()=>{
  const card=activeCard,previous=artChoices.get(card.id),next=$('art-variant').value;
  if(next) artChoices.set(card.id,next);else artChoices.delete(card.id);
  try { await showCard(card); }
  catch(error) { if(previous) artChoices.set(card.id,previous);else artChoices.delete(card.id);$('art-variant').value=previous || '';throw error; }
  invalidate();renderQueue();renderResults();
});
$('png').onclick=operation(async()=>{
  assertPrintable(activeCard,activeRender);
  download(await png300(activeRender.canvas),`${activeCard.id}-${activeCard.name.replace(/[^a-z0-9]+/gi,'-')}.png`);
});
$('prev-page').onclick=()=>{page--;showSheet();};$('next-page').onclick=()=>{page++;showSheet();};
$('cut-marks').onchange=()=>{if(prepared.length)showSheet();};

await operation(async()=>{
  const data=await api('/api/cards');
  let importError='';
  try {
    const saved=JSON.parse(localStorage.getItem(QUEUE_KEY) || '{}');
    if(saved.reworksImported || saved.queue?.some(([id])=>id<0)) {
      const reworks=await api('/api/reworked-cards');data.cards.push(...reworks.cards);
    }
  } catch(error) { importError=error.message; }
  setData(data,true);
  if(importError)notice(`Could not restore reworked cards: ${importError} Use Import reworked PNGs to retry.`,true);
  const requested=new URLSearchParams(location.search).get('card');
  if(requested) {
    const card=byId.get(Number(requested));
    if(!card)throw new Error('The requested card is absent from this snapshot.');
    await showCard(card);
  }
})();

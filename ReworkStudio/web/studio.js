import {renderCard, png300, compressedRGB} from '/renderer.js';
const $=id=>document.getElementById(id);
let reworks=[], originals=new Map(), active=null, draft=null, dirty=false, serial=0, revision=0, saving=false, timer;
const quantities=new Map(), statusOptions=new Map();
function notice(message,error=false){$('notice').textContent=message;$('notice').classList.toggle('error',error);}
async function api(path,body){
  const response=await fetch(path,body===undefined?{}:{method:'POST',headers:{'Content-Type':'application/json','X-UCPrint':'1'},body:JSON.stringify(body)});
  const value=await response.json();if(!response.ok)throw new Error(value.error||'Request failed.');return value;
}
function download(blob,name){const url=URL.createObjectURL(blob),a=document.createElement('a');a.href=url;a.download=name;a.click();setTimeout(()=>URL.revokeObjectURL(url),30000);}
function operation(fn){return async event=>{const button=event?.currentTarget;if(button?.tagName==='BUTTON')button.disabled=true;try{await fn(event);}catch(e){notice(e.message,true);}finally{if(button?.tagName==='BUTTON')button.disabled=false;}};}
function total(){const n=[...quantities.values()].reduce((a,b)=>a+b,0);$('total').textContent=`${n} cards · ${Math.ceil(n/9)} sheets`;}
function renderList(){
  const search=$('search').value.trim().toLowerCase();$('cards').replaceChildren();
  $('count').textContent=`${reworks.filter(r=>r.reviewed).length}/${reworks.length} checked`;
  for(const r of reworks.filter(r=>r.card.name.toLowerCase().includes(search))){
    const row=document.createElement('div');row.className='card-row'+(r.card.id===active?' active':'');
    const button=document.createElement('button');button.textContent=r.card.name;
    const sub=document.createElement('small');sub.textContent=r.reviewed?'Checked':'Needs review';button.append(sub);button.onclick=operation(()=>select(r.card.id));
    const qty=document.createElement('input');qty.type='number';qty.min='0';qty.max='99';qty.value=quantities.get(r.card.id)||0;qty.setAttribute('aria-label',`Copies of ${r.card.name}`);
    qty.onchange=()=>{const n=Number(qty.value);if(!Number.isInteger(n)||n<0||n>99){qty.value=quantities.get(r.card.id)||0;notice('Quantity must be a whole number from 0 to 99.',true);return;}quantities.set(r.card.id,n);total();};
    row.append(button,qty);$('cards').append(row);
  }
}
function markDirty(){revision++;dirty=true;$('state').textContent='Unsaved changes';clearTimeout(timer);timer=setTimeout(()=>preview().catch(e=>notice(e.message,true)),250);}
function readForm(){
  if(!$('form').reportValidity())throw new Error('Check the highlighted field.');
  const result=structuredClone(draft),c=result.card;
  for(const key of ['name','description','rarity','extension'])c[key]=$(key).value;
  for(const key of ['cost','attack','health'])c[key]=Number($(key).value);
  c.statuses=[];
  for(const row of $('statuses').children){
    if(!row.querySelector('[type=checkbox]').checked)continue;
    const status=structuredClone(statusOptions.get(row.dataset.image)),counter=row.querySelector('[type=number]').value;
    delete status.counter;if(counter!=='')status.counter=Number(counter);c.statuses.push(status);
  }
  result.reviewed=$('reviewed').checked;return result;
}
async function preview(){
  const ticket=++serial,card=readForm().card;
  try {const result=await renderCard(card);if(ticket!==serial)return;$('preview').replaceChildren(result.canvas);$('warnings').textContent=result.warnings.join(' ');}
  catch(e){if(ticket===serial){$('preview').replaceChildren();$('warnings').textContent=e.message;}throw e;}
}
function fillForm(){
  const c=draft.card,base=originals.get(c.id);$('title').textContent=base.name;$('state').textContent=draft.reviewed?'Checked · saved':'Needs review · saved';
  for(const key of ['name','cost','attack','health','description','rarity','extension'])$(key).value=c[key];
  $('reviewed').checked=draft.reviewed;$('reference').src='/reference/'+encodeURIComponent(draft.reference);$('reference').alt=base.name+' reworked PNG reference';
  $('official-stats').textContent=`Cost ${base.cost} · Attack ${base.attack} · Health ${base.health}`;$('official-rules').textContent=base.description;
  $('statuses').replaceChildren();
  for(const [image,status] of statusOptions){
    const current=c.statuses.find(s=>s.image===image),row=document.createElement('label');row.dataset.image=image;
    const check=document.createElement('input');check.type='checkbox';check.checked=!!current;
    const counter=document.createElement('input');counter.type='number';counter.min='0';counter.max='999';counter.placeholder='Count';counter.value=current?.counter??'';counter.setAttribute('aria-label',status.name+' counter');
    row.append(check,document.createTextNode(status.name),counter);$('statuses').append(row);
  }
}
async function select(id){
  if(dirty&&!confirm('Discard unsaved edits to this card?'))return;
  clearTimeout(timer);serial++;revision++;active=id;draft=structuredClone(reworks.find(r=>r.card.id===id));dirty=false;fillForm();renderList();await preview();
}
async function save(){
  if(saving)return;
  saving=true;
  try {
  const updated=readForm(),submittedRevision=revision;await renderCard(updated.card);
  const saved=await api('/api/reworks',updated);reworks[reworks.findIndex(r=>r.card.id===saved.card.id)]=saved;
  // A user may have selected another card while the request was in flight.
  if(active===saved.card.id && revision===submittedRevision){draft=structuredClone(saved);dirty=false;fillForm();await preview();}
  renderList();notice(`Saved ${saved.card.name} in ReworkedCards/reworks.json.`);
  } finally {saving=false;}
}
$('search').oninput=renderList;
$('form').oninput=markDirty;
$('form').onsubmit=event=>{event.preventDefault();operation(save)();};
$('revert').onclick=operation(()=>select(active));
$('png').onclick=operation(async()=>{const r=readForm(),result=await renderCard(r.card);download(await png300(result.canvas),`${r.card.id}-${r.card.name.replace(/[^a-z0-9]+/gi,'-')}-physical.png`);notice('PNG downloaded from the current editor. Save rework to keep your edits.');});
$('backup').onclick=operation(async()=>{const saved=await api('/api/reworks');download(new Blob([JSON.stringify(saved,null,2)+'\n'],{type:'application/json'}),'reworks.json');notice('Downloaded a copy of your saved reworks.');});
$('all').onclick=()=>{for(const r of reworks)quantities.set(r.card.id,1);renderList();total();};
$('clear').onclick=()=>{quantities.clear();renderList();total();};
$('pdf').onclick=operation(async()=>{
  if(dirty)throw new Error('Save or discard your edits before exporting the collection.');
  const items=reworks.filter(r=>(quantities.get(r.card.id)||0)>0).map(r=>({rework:structuredClone(r),quantity:quantities.get(r.card.id)}));
  if(!items.length)throw new Error('Choose at least one copy in the card list.');
  if(items.reduce((n,item)=>n+item.quantity,0)>250)throw new Error('Export at most 250 cards at a time.');
  const entries=[],warnings=[];
  for(const {rework:r,quantity} of items){notice(`Preparing ${r.card.name}…`);const result=await renderCard(r.card);warnings.push(...result.warnings);entries.push({id:r.card.id,quantity,rgb:await compressedRGB(result.canvas)});}
  const result=await api('/api/export',{cards:entries,cutMarks:$('cuts').checked});
  const link=document.createElement('a');link.href=result.url;link.download='physical-reworks.pdf';link.textContent=`Download PDF · ${result.cards} cards · ${result.pages} sheets`;
  const open=document.createElement('a');open.href=result.url;open.target='_blank';open.rel='noopener';open.textContent='Open PDF';
  $('export-links').replaceChildren(link,document.createTextNode(' · '),open);link.click();
  const unchecked=items.filter(item=>!item.rework.reviewed).length;
  notice(`PDF ready.${unchecked?` ${unchecked} selected reworks still need review.`:''} ${[...new Set(warnings)].join(' ')}`);
});
window.addEventListener('beforeunload',e=>{if(dirty){e.preventDefault();e.returnValue='';}});
await operation(async()=>{
  const [data,saved]=await Promise.all([api('/api/cards'),api('/api/reworks')]);reworks=saved;originals=new Map(data.cards.map(c=>[c.id,c]));
  for(const c of data.cards)for(const s of c.statuses)statusOptions.set(s.image,{name:s.name,image:s.image});
  if(!reworks.length)throw new Error('No reworks found in ReworkedCards/reworks.json.');
  await select(reworks[0].card.id);total();notice(`${reworks.length} editable reworks loaded. Compare each one with its reference before printing.`);
})();

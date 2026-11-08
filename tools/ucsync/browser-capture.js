// Reads the same loaded allCards catalog used by the successful September capture.
// Combined with browser-assets.js by: ucsync browser-capture
// No AllCards request, cache fallback, account export, or game modifications.

function ucLoadedCatalog(value) {
  if (typeof value === 'string') value = JSON.parse(value);
  if (!Array.isArray(value)) value = value?.cards ?? value?.allCards;
  if (typeof value === 'string') value = JSON.parse(value);
  if (!Array.isArray(value) || !value.length) throw new Error('No loaded catalog found. Open the game collection and wait for its cards before running this snippet.');
  // Snapshot the catalog so game navigation cannot mutate it during downloads.
  const cards = JSON.parse(JSON.stringify(value));
  const ids = new Set(), assets = new Set();
  function asset(category, name) {
    if (typeof name !== 'string' || !name) throw new Error('Missing artwork reference.');
    if (!/\.[^ .]+$/.test(name)) name += '.png';
    if (!/^[A-Za-z0-9_ .()!'-]+\.(png|jpe?g)$/i.test(name) || name.includes('..') || name.startsWith('.')) throw new Error(`Invalid artwork filename: ${name}`);
    assets.add(`${category}/${name}`);
  }
  for (const card of cards) {
    if (!card || !Number.isSafeInteger(Number(card.id)) || card.id == null || Number(card.id) < 0 || ids.has(Number(card.id))) throw new Error('Invalid or duplicate card ID.');
    ids.add(Number(card.id));
    if (!Number.isFinite(Number(card.cost)) || card.cost == null) throw new Error(`Card ${card.id} has no cost.`);
    asset('cards', card.image);
    if (card.statuses != null && !Array.isArray(card.statuses)) throw new Error(`Card ${card.id} has invalid statuses.`);
    for (const status of card.statuses || []) asset('powers', status.image ?? status.name);
  }
  return {cards, assets: [...assets].sort()};
}

async function ucWriteCaptureFile(folder, name, bytes) {
  const file = await folder.getFileHandle(name, {create:true});
  const writer = await file.createWritable();
  try { await writer.write(bytes); await writer.close(); }
  catch (error) { try { await writer.abort(); } catch {} throw error; }
}

function ucCaptureLoadedGame() {
  if (location.origin !== 'https://undercards.net') throw new Error('Run this snippet on https://undercards.net with the collection open.');
  if (document.getElementById('uc-print-capture')) throw new Error('The capture panel is already open.');
  const catalog = ucLoadedCatalog(typeof allCards !== 'undefined' ? allCards : window.allCards);
  const capturedAt = new Date().toISOString();
  const runName = 'capture-' + capturedAt.replace(/[:.]/g, '-');
  const panel = document.createElement('div'); panel.id = 'uc-print-capture';
  Object.assign(panel.style, {position:'fixed', top:'16px', right:'16px', width:'460px', maxWidth:'90vw', padding:'20px', background:'#fff', color:'#151515', border:'3px solid #167b67', zIndex:'2147483647', font:'15px Arial'});
  const title = document.createElement('strong'); title.textContent = `UC Print: capture ${catalog.cards.length} loaded cards`;
  const status = document.createElement('p'); status.style.whiteSpace = 'pre-wrap'; status.style.overflowWrap = 'anywhere';
  status.textContent = `Found the game's loaded catalog. Choose JSON/update to save current JSON and ${catalog.assets.length} images in a new capture folder. Keep this tab open until SUCCESS.`;
  const start = document.createElement('button'); start.textContent = 'Save complete capture';
  const stop = document.createElement('button'); stop.textContent = 'Stop'; stop.disabled = true;
  const close = document.createElement('button'); close.textContent = 'Close';
  let stopped = false, controller, folder, englishBytes;
  stop.onclick = () => { stopped = true; controller?.abort(); };
  close.onclick = () => panel.remove();
  panel.append(title, status, start, stop, close); document.body.append(panel);
  start.onclick = async () => {
    if (!window.showDirectoryPicker) { status.textContent = 'Use Chrome or Edge for folder access.'; return; }
    start.disabled = true; stop.disabled = false; close.disabled = true; stopped = false;
    controller = new AbortController();
    const get = path => fetch(path, {credentials:'same-origin', mode:'same-origin', redirect:'error', cache:'no-store', signal:AbortSignal.any([controller.signal, AbortSignal.timeout(30000)])});
    try {
      if (!folder) {
        const root = await window.showDirectoryPicker({id:'uc-print-capture', mode:'readwrite'});
        folder = await root.getDirectoryHandle(runName, {create:true});
      }
      if (!englishBytes) {
        status.textContent = 'Reading current English text...';
        const response = await get('/translation/en.json');
        if (!response.ok) throw new Error(`English text returned HTTP ${response.status}.`);
        const bytes = await response.arrayBuffer();
        const english = JSON.parse(new TextDecoder().decode(bytes));
        if (!english || Array.isArray(english) || !Object.keys(english).some(key => key.startsWith('card-name-')) || Object.values(english).some(value => typeof value !== 'string')) throw new Error('Invalid English dictionary.');
        englishBytes = bytes;
      }
      await ucWriteCaptureFile(folder, 'AllCards.json', JSON.stringify({cards:catalog.cards, capture:{source:'loaded game', capturedAt, page:location.pathname}}, null, 2));
      await ucWriteCaptureFile(folder, 'en.json', englishBytes);
      const assetRoot = await folder.getDirectoryHandle('assets', {create:true});
      const result = await ucCollectAssets({assets:catalog.assets}, assetRoot, {
        stopped:() => stopped, fetch:get,
        validate:async blob => {
          if (!blob.size || blob.size > 32*1024*1024) throw new Error('Empty or oversized image.');
          const b = new Uint8Array(await blob.slice(0,8).arrayBuffer());
          const png = b.length === 8 && [137,80,78,71,13,10,26,10].every((v,i) => b[i] === v);
          if (!png && !(b[0] === 255 && b[1] === 216 && b[2] === 255)) throw new Error('Expected a PNG/JPG image, received an error page or another format.');
          const image = await createImageBitmap(blob);
          const valid = image.width > 0 && image.height > 0 && image.width * image.height <= 25000000;
          image.close(); if (!valid) throw new Error('Invalid image dimensions.');
        },
        progress:p => { status.textContent = `${catalog.cards.length} cards saved. Images: ${p.ready}/${p.total}\n${p.asset}\nFolder: ${runName}`; },
        delay:() => new Promise(resolve => setTimeout(resolve,200))
      });
      status.textContent = `SUCCESS: ${catalog.cards.length} cards and ${result.ready} images saved in ${runName}.\nFrom the project folder, run:\npowershell -NoProfile -ExecutionPolicy Bypass -File .\\update-snapshot.ps1 -Capture ".\\JSON\\update\\${runName}"\nSee UPDATING.md for checks, backups and recovery.`;
      start.textContent = 'Check saved capture';
    } catch (error) {
      status.textContent = `STOPPED: ${error.message}\nSaved files remain in ${runName}. Click Resume to continue. The installed snapshot is unchanged.`;
      start.textContent = 'Resume';
    } finally { start.disabled = false; stop.disabled = true; close.disabled = false; }
  };
}

ucCaptureLoadedGame();

// UC Print browser JSON acquisition. Run as a DevTools Snippet on Undercards.
// Uses the open page's normal session. No uploads or cookie extraction.
// Saves raw responses only; the offline importer performs full validation.

async function ucFetchSnapshotJSON(fetchResponse) {
  async function read(path) {
    const response = await fetchResponse(path);
    if (!response.ok) throw new Error(`${path}: HTTP ${response.status}. The browser request was refused; no files were saved.`);
    const bytes = await response.arrayBuffer();
    if (!bytes.byteLength || bytes.byteLength >= 32 * 1024 * 1024) throw new Error(`${path}: empty or oversized response.`);
    let value;
    try { value = JSON.parse(new TextDecoder().decode(bytes)); }
    catch { throw new Error(`${path}: expected JSON, received an error page or invalid response. No files were saved.`); }
    return {bytes, value};
  }
  const raw = await read('/AllCards');
  let cards = Array.isArray(raw.value) ? raw.value : raw.value?.cards ?? raw.value?.allCards;
  if (typeof cards === 'string') {
    try { cards = JSON.parse(cards); }
    catch { throw new Error('AllCards contains an invalid card array. No files were saved.'); }
  }
  if (!Array.isArray(cards) || !cards.length || cards.some(card => !card || typeof card !== 'object' || Array.isArray(card) || card.id == null || !card.image)) {
    throw new Error('AllCards does not contain a non-empty card database with IDs and image references. No files were saved.');
  }
  const english = await read('/translation/en.json');
  if (!english.value || Array.isArray(english.value) || typeof english.value !== 'object' ||
      !Object.keys(english.value).some(key => key.startsWith('card-name-')) ||
      Object.values(english.value).some(value => typeof value !== 'string')) {
    throw new Error('English response does not contain the expected translation dictionary. No files were saved.');
  }
  return {cardCount: cards.length, files: [['AllCards.json', raw.bytes], ['en.json', english.bytes]]};
}

function ucBrowserData() {
  if (location.origin !== 'https://undercards.net') throw new Error('Run this snippet in your open https://undercards.net tab.');
  if (document.getElementById('uc-print-data-sync')) throw new Error('The UC Print JSON panel is already open.');
  const panel = document.createElement('div');
  panel.id = 'uc-print-data-sync';
  Object.assign(panel.style, {position:'fixed', top:'16px', right:'16px', width:'420px', maxWidth:'90vw', padding:'20px', background:'#fff', color:'#151515', border:'3px solid #167b67', zIndex:'2147483647', font:'15px Arial'});
  const title = document.createElement('strong');
  title.textContent = 'UC Print: download current card JSON';
  const status = document.createElement('p');
  status.style.whiteSpace = 'pre-wrap';
  status.textContent = 'Choose this project’s JSON/update folder. This downloads the card database and English text using your open browser session.';
  const start = document.createElement('button');
  start.textContent = 'Save current JSON';
  const close = document.createElement('button');
  close.textContent = 'Close';
  close.onclick = () => panel.remove();
  panel.append(title, status, start, close);
  document.body.append(panel);
  start.onclick = async () => {
    if (!window.showDirectoryPicker) { status.textContent = 'Use Chrome or Edge for folder access.'; return; }
    start.disabled = true;
    close.disabled = true;
    try {
      const folder = await window.showDirectoryPicker({id:'uc-print-json', mode:'readwrite'});
      status.textContent = 'Fetching current cards and English text…';
      const result = await ucFetchSnapshotJSON(path => fetch(path, {
        credentials:'same-origin', mode:'same-origin', redirect:'error', cache:'no-store', signal:AbortSignal.timeout(30000)
      }));
      // Fetch and check both responses before creating or replacing any files.
      for (const [name, bytes] of result.files) {
        status.textContent = `Saving ${name}…`;
        const handle = await folder.getFileHandle(name, {create:true});
        const writer = await handle.createWritable();
        try { await writer.write(bytes); await writer.close(); }
        catch (error) { try { await writer.abort(); } catch {} throw error; }
      }
      status.textContent = `SUCCESS: saved AllCards.json (${result.cardCount} cards) and en.json.\nNext: generate the artwork helper for these new files. The installed snapshot has not changed.`;
    } catch (error) {
      status.textContent = `STOPPED: ${error.message}\nThe installed card list is unchanged. If this was a file-writing error, both files must be saved successfully before importing.`;
    } finally { start.disabled = false; close.disabled = false; }
  };
}

ucBrowserData();

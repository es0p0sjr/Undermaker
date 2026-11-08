export const WIDTH = 744, HEIGHT = 1039;
const artwork = new Map();
let resources;
export function clearArtwork() { artwork.clear(); }

// Only local URLs are read. Each image is embedded before SVG rasterization,
// so the same source HTML/CSS layout supplies the preview, PNG and PDF.
async function localImage(url) {
  if (!artwork.has(url)) artwork.set(url, (async () => {
    const response = await fetch(url);
    if (!response.ok) throw new Error('Required local image is missing: ' + url + '. Rebuild the snapshot.');
    const blob = await response.blob();
    const uri = await new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => resolve(reader.result);
      reader.onerror = () => reject(new Error('Cannot read local image: ' + url));
      reader.readAsDataURL(blob);
    });
    await decodeImage(uri);
    return uri;
  })().catch(error => { artwork.delete(url); throw error; }));
  return artwork.get(url);
}
function decodeImage(uri) {
  return new Promise((resolve, reject) => {
    const img = new Image();
    img.onload = () => resolve(img);
    img.onerror = () => reject(new Error('Cannot render the local card image. Use Chrome or Edge and check the renderer bundle.'));
    img.src = uri;
  });
}
async function loadResources() {
  if (!resources) resources = (async () => {
    const response = await fetch('/renderer-assets.json');
    if (!response.ok) throw new Error('CardEditor resources are missing. Run npm run build, then rebuild UC Print.');
    const bundle = await response.json();
    if (bundle.version !== 1 || !bundle.templates?.MONSTER || !bundle.templates?.SPELL || !bundle.font || !bundle.rarities)
      throw new Error('Invalid local renderer bundle. Rebuild UC Print.');
    const font = new FontFace('DTM-Mono', 'url("' + bundle.font + '")');
    document.fonts.add(await font.load());
    return bundle;
  })().catch(error => { resources = null; throw error; });
  return resources;
}

// These additions bind dataset status and tribe icons to the source artwork cell.
// The frame, colours, typography, stats and rarity layout come from CardEditor.
const adapterCSS = [
  '.uc-root { color:white; background:black; margin:0; padding:0; font:16px/normal "DTM-Mono"; display:flow-root; }',
  // Anchor the bordered wrapper at (0,0) and contain its floated card table.
  // An inline-table can shift down on the root's text baseline during SVG
  // layout, pushing the bottom border outside the measured capture height.
  '.uc-root > .cardWrapper { display:flow-root; width:max-content; margin:0; }',
  '.uc-root > .cardWrapper > .cardBoard { display:table; float:none; margin:0; }',
  '.cardBoard img { image-rendering:pixelated; }',
  '.cardBoard .image { position:relative; }',
  '.cardBoard .uc-statuses { position:absolute; right:2px; top:2px; display:flex; gap:2px; }',
  '.cardBoard .uc-status { display:block; position:relative; width:14px; height:20px; font-size:10px; line-height:10px; color:white; }',
  '.cardBoard .uc-status img { display:block; width:14px; height:14px; object-fit:contain; }',
  '.cardBoard .uc-status span { position:absolute; bottom:0; left:0; right:0; text-align:center; text-shadow:1px 1px black,-1px -1px black; }',
  '.cardBoard .uc-tribes { position:absolute; right:4px; bottom:4px; display:flex; align-items:center; gap:2px; }',
  '.cardBoard .uc-tribes img.tribe { position:static; display:block; width:auto; height:auto; max-width:16px; max-height:16px; }',
  '.cardBoard .uc-tribes img.tribe.smallerIcon { max-width:14px; max-height:14px; }',
  '.cardBoard .name > span { display:block; }',
  '.cardBoard .description div { white-space:pre-wrap; overflow-wrap:break-word; }'
].join('\n');

export function resolveTribes(tribes = [], icons = {}) {
  const aliases = {TEM:'TEMMIE', FROG:'FROGGIT', ALL_MONSTER_TRIBES:'ALL'};
  const seen = new Set(), badges = [];
  for (const name of tribes) {
    const normalized = name.trim().toUpperCase().replace(/[\s-]+/g, '_');
    const key = aliases[normalized] || normalized;
    if (seen.has(key)) continue;
    if (!Object.hasOwn(icons, key)) throw new Error('Missing local tribe symbol: ' + name + '. Rebuild the renderer bundle.');
    seen.add(key); badges.push({name, key, src:icons[key]});
  }
  if (badges.length > 8) throw new Error('Too many tribe icons to fit over the artwork.');
  return badges;
}

function fitText(element, vertical) {
  for (let size = 12; size >= 7; size -= 0.5) {
    element.style.fontSize = size + 'px';
    if (vertical ? element.scrollHeight <= element.clientHeight + 1 : element.scrollWidth <= element.clientWidth + 1) return size;
  }
  throw new Error('Text does not fit the source card template. Inspect this card before printing.');
}

// A print layout for full artwork: keep current gameplay text legible on paper.
// It deliberately does not depend on the game's hover-only rules presentation.
const variantCSS = [
  '.uc-variant .image > img { display:block; width:100%; max-width:100%; height:90px; object-fit:contain; }',
  '.uc-full-art .cardBoard { background-size:cover; background-position:center; background-repeat:no-repeat; }',
  '.uc-full-art .cardBoard td { border-color:transparent; }',
  '.uc-full-art .cardBoard .image { background:transparent; }',
  '.uc-full-art .image > img { visibility:hidden; }',
  '.uc-full-art .cardBoard .name, .uc-full-art .cardBoard .cost, .uc-full-art .cardBoard .description, .uc-full-art .cardBoard .attack, .uc-full-art .cardBoard .health, .uc-full-art .cardBoard .rarity { background:rgba(0,0,0,.78); text-shadow:1px 1px black; }'
].join('\n');

export async function renderCard(card, variantID) {
  if (!['MONSTER', 'SPELL'].includes(card.type)) throw new Error('No source template for card type: ' + card.type);
  if (!card.image) throw new Error('Snapshot has no artwork reference. Rebuild the snapshot.');
  const bundle = await loadResources();
  const rarity = card.rarity === 'GENERATED' ? 'TOKEN' : card.rarity;
  const suffix = ({DELTARUNE:'_DR', DR:'_DR', UTY:'_UTY', BASE:''})[card.extension || 'BASE'];
  if (suffix === undefined) throw new Error('Unsupported card extension: ' + card.extension);
  const icon = bundle.rarities[rarity + suffix];
  if (!icon) throw new Error('Missing local rarity symbol: ' + rarity + suffix);
  const statuses = card.statuses || [];
  if (statuses.length > 9) throw new Error('Too many status icons to fit over the artwork.');
  const variant = variantID ? (card.artVariants || []).find(v => v.id === variantID) : null;
  if (variantID && !variant) throw new Error('Unknown artwork variant: ' + variantID);
  const art = await localImage('/api/art/' + card.id + (variant ? '?variant=' + encodeURIComponent(variant.id) : ''));
  const powers = await Promise.all(statuses.map(status => localImage('/api/power/' + encodeURIComponent(status.image))));
  const warnings = [...(card.issues || [])];
  const tribes = resolveTribes(card.tribes || [], bundle.tribes);

  // Template parsing is inert. Remove editor controls/default images before
  // attachment, so none of the source editor's URLs can trigger requests.
  const template = document.createElement('template');
  template.innerHTML = bundle.templates[card.type];
  template.content.querySelectorAll('input,textarea,.no-save,.footer').forEach(el => el.remove());
  const wrapper = document.createElement('span');
  wrapper.className = 'cardWrapper';
  wrapper.append(template.content);
  wrapper.querySelector('.name span').textContent = card.name;
  wrapper.querySelector('.cost span').textContent = String(card.cost ?? '?');
  wrapper.querySelector('.description div').textContent = card.description || ' ';
  wrapper.querySelector('.image > img').src = art;
  if (variant) {
    wrapper.classList.add('uc-variant');
    if (variant.layout === 'full') {
      wrapper.classList.add('uc-full-art');
      wrapper.querySelector('.cardBoard').style.backgroundImage = 'url("' + art + '")';
      warnings.push('Full-art print layout: artwork is cropped to fill the frame; rules and stats remain visible.');
    }
  }
  wrapper.querySelector('.rarity img').src = icon;
  if (card.type === 'MONSTER') {
    wrapper.querySelector('.attack span').textContent = String(card.attack ?? '?');
    wrapper.querySelector('.health span').textContent = String(card.health ?? '?');
  }
  const statusRoot = document.createElement('span');
  statusRoot.className = 'uc-statuses';
  statuses.forEach((status, i) => {
    const badge = document.createElement('span');
    badge.className = 'uc-status';
    const img = document.createElement('img');
    img.src = powers[i]; img.alt = status.name;
    badge.append(img);
    if (status.counter != null) {
      const count = document.createElement('span');
      count.textContent = String(status.counter);
      badge.append(count);
    }
    statusRoot.append(badge);
  });
  wrapper.querySelector('.image').append(statusRoot);
  if (tribes.length) {
    const tribeRoot = document.createElement('span');
    tribeRoot.className = 'uc-tribes';
    tribes.forEach(({name, key, src}) => {
      const img = document.createElement('img');
      img.className = 'tribe' + (key === 'SPIDER' ? ' smallerIcon' : '');
      img.src = src; img.alt = name; img.title = name;
      tribeRoot.append(img);
    });
    wrapper.querySelector('.image').append(tribeRoot);
  }

  // Isolate the original stylesheet from the application's own UI styles.
  // Measure at one device pixel per CSS pixel. At fractional display/browser
  // scale (e.g. 125%), Chromium snaps a 2px border to 1.6px in the live page,
  // but the SVG image lays it out at 2px. The shorter measured table then clips
  // its bottom border on export. The host is not included in the SVG snapshot.
  const measurementZoom = 1 / (window.devicePixelRatio || 1);
  const host = document.createElement('div');
  host.style.cssText = 'position:fixed;left:-10000px;top:0;width:180px;pointer-events:none;';
  host.style.zoom = measurementZoom;
  const measure = element => {
    const bounds = element.getBoundingClientRect();
    // Restore layout's 1/64px precision after undoing zoom, avoiding floating
    // point residue that would round an exact 164px frame up to 165px.
    const unzoom = value => Math.round(value / measurementZoom * 64) / 64;
    return {width: unzoom(bounds.width), height: unzoom(bounds.height)};
  };
  host.setAttribute('aria-hidden', 'true');
  const shadow = host.attachShadow({mode:'open'});
  const root = document.createElement('div');
  root.className = 'uc-root';
  const style = document.createElement('style');
  style.textContent = bundle.styles.join('\n') + '\n' + adapterCSS + '\n' + variantCSS;
  root.append(style, wrapper);
  shadow.append(root);
  document.body.append(host);
  try {
    await Promise.all([...wrapper.querySelectorAll('img')].map(img => img.decode()));
    fitText(wrapper.querySelector('.name span'), false);
    const size = fitText(wrapper.querySelector('.description div'), true);
    if (size < 9) warnings.push('Small rules text: check readability on the printed test sheet.');
    if (card.demo) warnings.push('DEMO: fictional card for print testing.');
    const bounds = measure(wrapper);
    if (!bounds.width || !bounds.height) throw new Error('Source card layout has no printable size.');
    const captureWidth = Math.ceil(bounds.width), captureHeight = Math.ceil(bounds.height);
    // SVG images lay out their HTML again, independently of the live document.
    // Freeze the measured artwork and frame sizes so percentage image heights
    // cannot expand the table into the wrapper's bottom border during that pass.
    const capturedRoot = root.cloneNode(true);
    const captureSelector = '.cardWrapper, .cardBoard, .image > img';
    const originals = root.querySelectorAll(captureSelector);
    capturedRoot.querySelectorAll(captureSelector).forEach((element, index) => {
      const measured = measure(originals[index]);
      element.style.boxSizing = 'border-box';
      element.style.width = measured.width + 'px';
      element.style.height = measured.height + 'px';
    });
    // Size only the snapshot, after measuring the live layout. Round outward
    // so a fractional CSS pixel cannot trim the outer black border.
    capturedRoot.style.width = captureWidth + 'px';
    capturedRoot.style.height = captureHeight + 'px';

    // Serialize the actual source DOM, with all fonts and images embedded.
    // SVG scales it to print resolution without redrawing the frame by hand.
    const svgNS = 'http://www.w3.org/2000/svg';
    const svg = document.createElementNS(svgNS, 'svg');
    svg.setAttribute('width', WIDTH);
    svg.setAttribute('height', HEIGHT);
    svg.setAttribute('viewBox', '0 0 ' + captureWidth + ' ' + captureHeight);
    svg.setAttribute('preserveAspectRatio', 'none');
    const foreign = document.createElementNS(svgNS, 'foreignObject');
    foreign.setAttribute('width', captureWidth);
    foreign.setAttribute('height', captureHeight);
    foreign.append(capturedRoot);
    svg.append(foreign);
    const xml = new XMLSerializer().serializeToString(svg);
    const rendered = await decodeImage('data:image/svg+xml;charset=utf-8,' + encodeURIComponent(xml));
    const canvas = document.createElement('canvas');
    canvas.width = WIDTH; canvas.height = HEIGHT;
    const ctx = canvas.getContext('2d', {willReadFrequently:true});
    ctx.fillStyle = '#000'; ctx.fillRect(0,0,WIDTH,HEIGHT);
    ctx.imageSmoothingEnabled = false;
    ctx.drawImage(rendered, 0, 0, WIDTH, HEIGHT);
    // Fail here if the browser forbids export, before presenting an unusable card.
    ctx.getImageData(0, 0, 1, 1);
    return {canvas, warnings};
  } finally {
    host.remove();
  }
}

export async function compressedRGB(canvas) {
  if (!globalThis.CompressionStream) throw new Error('Use a current Chrome, Edge or Firefox browser for PDF export.');
  const rgba = canvas.getContext('2d').getImageData(0,0,WIDTH,HEIGHT).data;
  const rgb = new Uint8Array(WIDTH*HEIGHT*3);
  for(let source=0,target=0;source<rgba.length;source+=4) {
    rgb[target++]=rgba[source]; rgb[target++]=rgba[source+1]; rgb[target++]=rgba[source+2];
  }
  const stream = new Blob([rgb]).stream().pipeThrough(new CompressionStream('deflate'));
  const bytes = new Uint8Array(await new Response(stream).arrayBuffer());
  let binary='';
  for(let i=0;i<bytes.length;i+=32768) binary += String.fromCharCode(...bytes.subarray(i,i+32768));
  return btoa(binary);
}

function crc32(bytes) {
  let crc=0xffffffff;
  for(const byte of bytes) {
    crc ^= byte;
    for(let bit=0;bit<8;bit++) crc=(crc>>>1)^((crc&1)?0xedb88320:0);
  }
  return (crc^0xffffffff)>>>0;
}

export async function png300(canvas) {
  // Browsers encode canvas PNGs as 96 dpi. Write a real PNG pHYs chunk at 300 dpi.
  const blob = await new Promise(resolve=>canvas.toBlob(resolve,'image/png'));
  const source = new Uint8Array(await blob.arrayBuffer());
  const chunk = new Uint8Array(21), view = new DataView(chunk.buffer);
  view.setUint32(0,9); chunk.set([112,72,89,115],4);
  view.setUint32(8,11811); view.setUint32(12,11811); chunk[16]=1;
  view.setUint32(17,crc32(chunk.subarray(4,17)));
  const pieces=[source.subarray(0,33),chunk];
  for(let pos=33;pos<source.length;) {
    const length=new DataView(source.buffer,source.byteOffset+pos,4).getUint32(0)+12;
    if(String.fromCharCode(...source.subarray(pos+4,pos+8))!=='pHYs') pieces.push(source.subarray(pos,pos+length));
    pos+=length;
  }
  return new Blob(pieces,{type:'image/png'});
}

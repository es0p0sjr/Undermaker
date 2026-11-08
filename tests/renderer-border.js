(async () => {
  const {renderCard, png300, compressedRGB} = await import('/renderer.js');
  const {cards, manifest} = await (await fetch('/api/cards')).json();
  const ids = manifest.demo ? [900001, 900002, 900003] : [642, 771];
  const results = [];
  for (const id of ids) {
    const card = cards.find(card => card.id === id);
    if (!card) throw new Error('Border fixture card is absent: ' + id);
    const {canvas} = await renderCard(card);
    const {width:w, height:h} = canvas;
    const pixels = canvas.getContext('2d').getImageData(0, 0, w, h).data;
    const black = (x, y) => {
      const p = (y*w+x)*4;
      return pixels[p] < 11 && pixels[p+1] < 11 && pixels[p+2] < 11 && pixels[p+3] === 255;
    };
    const white = (x, y) => {
      const p = (y*w+x)*4;
      return pixels[p] > 235 && pixels[p+1] > 235 && pixels[p+2] > 235;
    };
    const edges = {};
    for (const [name, point, length] of [
      ['top', (i, j) => [j, i], w], ['bottom', (i, j) => [j, h-1-i], w],
      ['left', (i, j) => [i, j], h], ['right', (i, j) => [w-1-i, j], h]
    ]) {
      // Check a continuous strip, not just the midpoint or one black pixel.
      for (let inset = 0; inset < 6; inset++) {
        for (let along = 0; along < length; along++) {
          if (!black(...point(inset, along))) throw new Error(`${card.name}: missing ${name} black border at ${inset},${along}`);
        }
      }
      let inset = 0;
      while (inset < 30 && black(...point(inset, Math.floor(length/2)))) inset++;
      if (inset === 30 || !white(...point(inset+2, Math.floor(length/2)))) {
        throw new Error(`${card.name}: missing white frame inside the ${name} black border`);
      }
      edges[name] = inset;
    }
    const png = await png300(canvas);
    const bitmap = await createImageBitmap(png);
    const decoded = document.createElement('canvas');
    decoded.width = w; decoded.height = h;
    decoded.getContext('2d').drawImage(bitmap, 0, 0);
    bitmap.close();
    const pngPixels = decoded.getContext('2d').getImageData(0, 0, w, h).data;
    if (!pixels.every((byte, i) => byte === pngPixels[i])) throw new Error(`${card.name}: PNG changed rendered pixels`);
    const pngBytes = new Uint8Array(await png.arrayBuffer());
    let binary = '';
    for (let pos = 0; pos < pngBytes.length; pos += 32768) binary += String.fromCharCode(...pngBytes.subarray(pos, pos+32768));
    results.push({id, name:card.name, edges, png:btoa(binary), rgb:await compressedRGB(canvas)});
  }
  return results;
})()

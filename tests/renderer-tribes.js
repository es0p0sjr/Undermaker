(async () => {
  const {renderCard,png300,compressedRGB}=await import('/renderer.js');
  const {cards}=await (await fetch('/api/cards')).json();
  const results=[];
  for(const tribes of [['Snail'],['Tem'],['Frog'],['All monster tribes'],['Dog','Royal Guard']]) {
    const base=cards.find(c=>c.tribes?.includes(tribes[0]));
    if(!base)throw Error('Tribe test card missing: '+tribes[0]);
    const card={...base,tribes};
    const {canvas}=await renderCard(card),plain=(await renderCard({...card,tribes:[]})).canvas;
    const w=canvas.width,h=canvas.height;
    const pixels=canvas.getContext('2d').getImageData(0,0,w,h).data;
    const before=plain.getContext('2d').getImageData(0,0,w,h).data;
    let changed=0,minX=w,maxX=0,minY=h,maxY=0;
    for(let i=0;i<pixels.length;i+=4) {
      if(pixels[i]===before[i]&&pixels[i+1]===before[i+1]&&pixels[i+2]===before[i+2])continue;
      const x=(i/4)%w,y=Math.floor(i/4/w);
      changed++;minX=Math.min(minX,x);maxX=Math.max(maxX,x);minY=Math.min(minY,y);maxY=Math.max(maxY,y);
    }
    if(changed<20 || minX<w*.60 || maxX>w*.97 || minY<h*.25 || maxY>h*.60) {
      throw Error(`${tribes}: tribe icons missing or outside the lower-right artwork area: ${JSON.stringify({changed,minX,maxX,minY,maxY})}`);
    }
    const png=await png300(canvas),bitmap=await createImageBitmap(png);
    const decoded=document.createElement('canvas');decoded.width=w;decoded.height=h;decoded.getContext('2d').drawImage(bitmap,0,0);bitmap.close();
    const roundtrip=decoded.getContext('2d').getImageData(0,0,w,h).data;
    if(!pixels.every((v,i)=>v===roundtrip[i]))throw Error('PNG lost tribe pixels');
    const bytes=new Uint8Array(await png.arrayBuffer());let binary='';
    for(let i=0;i<bytes.length;i+=32768)binary+=String.fromCharCode(...bytes.subarray(i,i+32768));
    results.push({id:card.id,name:card.name+' '+tribes.join('/'),fileTag:'-'+tribes.join('-').replaceAll(' ','_'),edges:{changed,minX,maxX,minY,maxY},png:btoa(binary),rgb:await compressedRGB(canvas)});
  }
  return results;
})()

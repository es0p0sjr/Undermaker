(async () => {
  const checks=[];
  const assert=(test,label)=>{if(!test)throw new Error(label);checks.push(label);};
  const wait=async(test,label)=>{
    const deadline=Date.now()+20000;
    while(!test()) {if(Date.now()>deadline)throw new Error(`Timeout: ${label}`);await new Promise(r=>setTimeout(r,80));}
  };
  const $=id=>document.getElementById(id);
  await wait(()=>$('source').textContent==='DEMO SNAPSHOT' && !document.body.classList.contains('busy'),'app startup');
  assert($('results-count').textContent==='4 cards found','Embedded library loads with all four fixture cards');
  $('deck-code').value=btoa(JSON.stringify({soul:'JUSTICE',cardIds:[...Array(12).fill(900001),...Array(12).fill(900002),900004],artifactIds:[]}));
  $('load-deck').click();await wait(()=>$('summary').textContent.startsWith('25 cards')&&!document.body.classList.contains('busy'),'local deck import');
  $('filter').value='TOKEN';$('filter').dispatchEvent(new Event('change'));
  let initial=$('results').firstChild;initial.querySelector('input').value=4;initial.lastChild.click();
  assert($('summary').textContent.startsWith('29 cards') && $('summary').textContent.includes('4 in token pool'),'Token pool excluded from deck count');
  $('filter').value='ALL';$('filter').dispatchEvent(new Event('change'));
  $('search').value='Practice';$('search').dispatchEvent(new Event('input'));
  assert($('results').children.length===1,'Search filters cards');
  $('search').value='';$('search').dispatchEvent(new Event('input'));
  $('filter').value='TOKEN';$('filter').dispatchEvent(new Event('change'));
  assert($('results').children.length===1 && $('results').textContent.includes('Generated Test Token'),'Token filter');
  const row=$('results').firstChild;row.querySelector('input').value=2;row.lastChild.click();
  assert($('summary').textContent.startsWith('31 cards'),'Add quantities to existing token');
  const tokenInput=$('queue').querySelector('input[aria-label="Print quantity of Generated Test Token"]');
  tokenInput.value=4;tokenInput.dispatchEvent(new Event('change'));
  assert($('summary').textContent.startsWith('29 cards'),'Change print quantity');
  $('deck-code').value='not a valid code';$('load-deck').click();
  await wait(()=>!document.body.classList.contains('busy'),'invalid deck');
  assert($('notice').classList.contains('error') && $('summary').textContent.startsWith('29 cards'),'Bad deck preserves queue');
  $('preview').click();await wait(() => !$('sheets-panel').hidden&&!document.body.classList.contains('busy'),'preview');
  assert($('sheet').querySelectorAll('canvas').length===9,'Nine cards on first sheet');
  $('next-page').click();$('next-page').click();$('next-page').click();
  assert($('sheet').querySelectorAll('canvas').length===2,'Two cards on fourth sheet');
  $('export').click();await wait(()=>$('export-result').querySelector('a')&&!document.body.classList.contains('busy'),'PDF export');
  const pdfURL=$('export-result').querySelector('a').href;
  const pdfBytes=new Uint8Array(await (await fetch(pdfURL)).arrayBuffer());
  const pdfText=new TextDecoder('latin1').decode(pdfBytes);
  assert(pdfText.startsWith('%PDF-1.4') && (pdfText.match(/\/Type \/Page\b/g)||[]).length===4,'Four-page PDF from real UI');
  const {renderCard,png300,compressedRGB}=await import('/renderer.js');
  const data=await (await fetch('/api/cards')).json();
  for(const card of data.cards) {
    const {canvas}=await renderCard(card);assert(canvas.width===744&&canvas.height===1039,`Render ${card.name}`);
    const pixels=canvas.getContext('2d').getImageData(0,0,744,1039).data;
    let white=0,cyan=0;
    for(let p=0;p<pixels.length;p+=4) {
      if(pixels[p]>235&&pixels[p+1]>235&&pixels[p+2]>235)white++;
      if(pixels[p]<40&&pixels[p+1]>130&&pixels[p+2]>220)cyan++;
    }
    assert(white>3000&&cyan>30,`Source frame and cyan cost are visible for ${card.name}`);
    let bottomBlack=true, bottomWhite=false;
    for(let x=0;x<744;x++) {
      const p=((1039-2)*744+x)*4;
      if(pixels[p]>10||pixels[p+1]>10||pixels[p+2]>10)bottomBlack=false;
    }
    for(let y=1039-30;y<1039-2;y++) {
      const p=(y*744+372)*4;
      if(pixels[p]>235&&pixels[p+1]>235&&pixels[p+2]>235)bottomWhite=true;
    }
    assert(bottomBlack&&bottomWhite,`Black strip survives below the white bottom border for ${card.name}`);
  }
  const canvas=(await renderCard(data.cards[0])).canvas;
  const pngBytes=new Uint8Array(await (await png300(canvas)).arrayBuffer());
  let found=false;
  for(let p=8;p<pngBytes.length;) {
    const v=new DataView(pngBytes.buffer,p),n=v.getUint32(0),name=String.fromCharCode(...pngBytes.subarray(p+4,p+8));
    if(name==='pHYs')found=v.getUint32(8)===11811&&v.getUint32(12)===11811;
    p+=n+12;
  }
  assert(found,'PNG contains 300 dpi metadata');
  assert(performance.getEntriesByType('resource').every(r=>new URL(r.name).origin===location.origin || r.name.startsWith('data:')),'Every app resource is embedded or uses the loopback origin');
  let missingFailed=false;try{await renderCard({...data.cards[0],id:99999999});}catch{missingFailed=true;}
  assert(missingFailed,'Missing artwork fails without a placeholder or download');
  const compressed=await compressedRGB(canvas);
  const bytes=Uint8Array.from(atob(compressed),c=>c.charCodeAt(0));
  const rgb=await new Response(new Blob([bytes]).stream().pipeThrough(new DecompressionStream('deflate'))).arrayBuffer();
  assert(rgb.byteLength===744*1039*3,'Lossless RGB payload dimensions');
  let overflow=false;
  try {await renderCard({...data.cards[0],description:'A very long rule. '.repeat(500)});}catch{overflow=true;}
  assert(overflow,'Overlong rules fail visibly instead of clipping');
  $('filter').value='ALL';$('filter').dispatchEvent(new Event('change'));
  return {checks,pdfURL,pngBase64:btoa(Array.from(pngBytes,b=>String.fromCharCode(b)).join(''))};
})()

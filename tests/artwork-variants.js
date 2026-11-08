(async () => {
  // Test-only catalog: do not install fabricated skins in the real snapshot.
  const nativeFetch=window.fetch.bind(window);
  const data=await (await nativeFetch('/api/cards')).json();
  const fixtures=[data.cards.find(c=>c.id===642),data.cards.find(c=>c.id===771)];
  const other=data.cards.find(c=>!fixtures.includes(c));
  const portrait=document.createElement('canvas');portrait.width=400;portrait.height=600;
  const ctx=portrait.getContext('2d'),gradient=ctx.createLinearGradient(0,0,400,600);
  gradient.addColorStop(0,'#167b67');gradient.addColorStop(1,'#96357e');ctx.fillStyle=gradient;ctx.fillRect(0,0,400,600);
  ctx.fillStyle='#fff';ctx.fillRect(180,150,40,300);
  const portraitBlob=await new Promise(resolve=>portrait.toBlob(resolve));
  for(const card of fixtures) card.artVariants=[
    {id:'test-alternate',name:'Test alternate',image:other.image,layout:'alternate'},
    {id:'test-full',name:'Test full',image:other.image,layout:'full'}
  ];
  window.fetch=(url,...args)=>{
    if(url==='/api/cards') return Promise.resolve(new Response(JSON.stringify(data),{headers:{'Content-Type':'application/json'}}));
    if(String(url).includes('?variant=test-full'))return Promise.resolve(new Response(portraitBlob));
    if(String(url).includes('?variant=test-')) return nativeFetch('/api/art/'+other.id,...args);
    return nativeFetch(url,...args);
  };
  const wait=async()=>{for(let i=0;i<200 && document.body.classList.contains('busy');i++)await new Promise(r=>setTimeout(r,20));if(document.body.classList.contains('busy'))throw Error('App remained busy');};
  await wait();
  localStorage.removeItem('uc-print-queue-snapshot-v1');
  await import('/app.js?variant-test');
  const {renderCard,png300,compressedRGB}=await import('/renderer.js');
  const output=[];
  for(const card of fixtures) {
    const standard=(await renderCard(card)).canvas.toDataURL();
    const info=JSON.stringify([card.description,card.cost,card.attack,card.health]);
    document.getElementById('search').value=card.name;
    document.getElementById('search').oninput();
    const row=[...document.querySelectorAll('#results .card-row')].find(r=>r.querySelector('.card-name').textContent===card.name);
    row.querySelector('button:last-child').click();
    await row.querySelector('.card-name').onclick();
    let previous=standard;
    for(const variant of card.artVariants) {
      const select=document.getElementById('art-variant');
      if(select.disabled || select.options.length!==3)throw Error('Artwork selector unavailable');
      select.value=variant.id;await select.onchange();
      const preview=document.querySelector('#card-preview canvas');
      const direct=(await renderCard(card,variant.id)).canvas;
      if(preview.toDataURL()!==direct.toDataURL())throw Error('Preview did not select variant');
      if(preview.toDataURL()===previous)throw Error('Artwork change did not change pixels');
      previous=preview.toDataURL();
      await document.getElementById('preview').onclick();
      if(![...document.querySelectorAll('#sheet canvas')].some(c=>c.toDataURL()===preview.toDataURL()))throw Error('Sheet lost artwork choice');
      if(JSON.stringify([card.description,card.cost,card.attack,card.health])!==info)throw Error('Variant changed gameplay data');
      const saved=JSON.parse(localStorage.getItem('uc-print-queue-snapshot-v1'));
      if(!saved.artChoices.some(([id,v])=>id===card.id&&v===variant.id))throw Error('Choice was not saved');
      const png=await png300(direct),bitmap=await createImageBitmap(png);
      if(bitmap.width!==744||bitmap.height!==1039)throw Error('Incorrect print dimensions');
      const decoded=document.createElement('canvas');decoded.width=744;decoded.height=1039;decoded.getContext('2d').drawImage(bitmap,0,0);bitmap.close();
      if(decoded.toDataURL()!==direct.toDataURL())throw Error('PNG changed pixels');
      output.push({id:card.id,fileTag:'-'+variant.id,name:card.name+' '+variant.id,edges:{},rgb:await compressedRGB(direct)});
      output.at(-1).png=await new Promise(resolve=>{const reader=new FileReader();reader.onload=()=>resolve(reader.result.split(',')[1]);reader.readAsDataURL(png);});
    }
    document.getElementById('close-dialog').click();
  }
  await import('/app.js?variant-test-restored');
  document.getElementById('search').value=fixtures[0].name;document.getElementById('search').oninput();
  await document.querySelector('#results .card-name').onclick();
  if(document.getElementById('art-variant').value!=='test-full')throw Error('Saved artwork did not restore');
  return output;
})()

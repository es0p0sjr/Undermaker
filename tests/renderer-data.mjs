// Optional Node check for actual browser export serialization, without a DOM.
// Usage: node tests/renderer-data.mjs
import fs from 'node:fs';
import assert from 'node:assert/strict';
import zlib from 'node:zlib';
const source=fs.readFileSync(new URL('../web/renderer.js',import.meta.url),'utf8');
const {png300,compressedRGB,WIDTH,HEIGHT}=await import(`data:text/javascript;base64,${Buffer.from(source).toString('base64')}`);
const rgba=new Uint8ClampedArray(WIDTH*HEIGHT*4);
for(let i=0;i<rgba.length;i+=4) {rgba[i]=i%251;rgba[i+1]=27;rgba[i+2]=91;rgba[i+3]=255;}
const packed=await compressedRGB({getContext:()=>({getImageData:()=>({data:rgba})})});
const rgb=zlib.inflateSync(Buffer.from(packed,'base64'));
assert.equal(rgb.length,WIDTH*HEIGHT*3);
for(let i=0,j=0;i<rgba.length;i+=4,j+=3) assert.deepEqual([...rgb.subarray(j,j+3)],[...rgba.subarray(i,i+3)]);
console.log('PASS: Lossless RGB export preserves every colour channel.');
const original=Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+aWQAAAABJRU5ErkJggg==','base64');
const png=Buffer.from(await (await png300({toBlob:callback=>callback(new Blob([original]))})).arrayBuffer());
const chunk=png.subarray(33,54);
assert.equal(chunk.readUInt32BE(0),9);assert.equal(chunk.subarray(4,8).toString(),'pHYs');
assert.equal(chunk.readUInt32BE(8),11811);assert.equal(chunk.readUInt32BE(12),11811);assert.equal(chunk[16],1);
assert.deepEqual(Buffer.concat([png.subarray(0,33),png.subarray(54)]),original);
let crc=0xffffffff;for(const value of chunk.subarray(4,17)){crc^=value;for(let i=0;i<8;i++)crc=crc&1?(crc>>>1)^0xedb88320:crc>>>1;}
assert.equal(chunk.readUInt32BE(17),(crc^0xffffffff)>>>0);
console.log('PASS: PNG gets valid 300 dpi metadata while original image bytes are preserved.');

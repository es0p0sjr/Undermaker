// Plain JavaScript/CSS needs no bundler or downloaded npm dependencies.
import fs from 'node:fs';
import path from 'node:path';
import {spawnSync} from 'node:child_process';
const root=process.cwd(),data=process.env.UC_DATA||'data';
const validation=spawnSync('go',['run','./tools/ucsync','validate','--data',data],{stdio:'inherit',env:{...process.env,GOPROXY:'off',GOTOOLCHAIN:'local'}});
if(validation.error)console.error(validation.error.message);
if(validation.status!==0)process.exit(validation.status||1);
const renderer=spawnSync('go',['run','./tools/ucsync','renderer-bundle'],{stdio:'inherit',env:{...process.env,GOPROXY:'off',GOTOOLCHAIN:'local'}});
if(renderer.error)console.error(renderer.error.message);
if(renderer.status!==0)process.exit(renderer.status||1);
const files=['index.html','style.css','library.css','theme.css','theme.js','app.js','renderer.js','complete-card.js','renderer-assets.json'];
for(const name of files.filter(n=>n.endsWith('.js'))){
  const input=fs.openSync(path.join(root,'web',name),'r');
  const check=spawnSync(process.execPath,['--input-type=module','--check'],{stdio:[input,'inherit','inherit']});
  fs.closeSync(input);
  if(check.error)console.error(check.error.message);
  if(check.status!==0)process.exit(check.status||1);
}
const output=path.resolve(root,'.build','frontend');
if(path.dirname(output)!==path.resolve(root,'.build'))throw new Error('Unsafe build output');
fs.rmSync(output,{recursive:true,force:true});fs.mkdirSync(output,{recursive:true});
for(const name of files)fs.copyFileSync(path.join(root,'web',name),path.join(output,name));
console.log('Frontend built from local files.');

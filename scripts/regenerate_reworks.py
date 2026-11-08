"""Regenerate complete reworked PNGs from reviewed JSON with the shared renderer.

Creates timestamped output and original-image backups. Only installs PNGs after
all cards and their A4 PDF have rendered and passed lossless export checks.
"""
import argparse
import base64
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import struct
import subprocess
import tempfile
import time
import urllib.request
import zlib

from browser_test import DevTools

ROOT = Path(__file__).resolve().parents[1]
HIDDEN = subprocess.CREATE_NO_WINDOW if os.name == 'nt' else 0


def validate_png(data):
    if not data.startswith(b'\x89PNG\r\n\x1a\n'):
        raise ValueError('Renderer did not return a PNG')
    pos, dimensions, dpi = 8, None, None
    while pos < len(data):
        length = struct.unpack('>I', data[pos:pos+4])[0]
        chunk = data[pos+4:pos+8+length]
        crc = struct.unpack('>I', data[pos+8+length:pos+12+length])[0]
        if zlib.crc32(chunk) & 0xffffffff != crc:
            raise ValueError('PNG checksum mismatch')
        if chunk[:4] == b'IHDR': dimensions = struct.unpack('>II', chunk[4:12])
        if chunk[:4] == b'pHYs': dpi = struct.unpack('>IIB', chunk[4:13])
        pos += length + 12
    if pos != len(data) or dimensions != (744, 1039) or dpi != (11811, 11811, 1):
        raise ValueError(f'Incorrect PNG dimensions or resolution: {dimensions}, {dpi}')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--browser')
    parser.add_argument('--single-process', action='store_true', help='Use the existing local headless-shell rendering mode')
    args = parser.parse_args()
    candidates = [args.browser, r'C:\Program Files\Google\Chrome\Application\chrome.exe',
                  r'C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe']
    browser = next((p for p in candidates if p and Path(p).is_file()), None)
    if not browser: parser.error('Supply an installed Chromium browser with --browser')
    source_path = ROOT / 'ReworkedCards/reworks.json'
    source = source_path.read_bytes()
    reworks = json.loads(source)
    references = [r['reference'] for r in reworks]
    if not references or len(set(references)) != len(references):
        raise ValueError('Expected a nonempty collection with unique reference filenames')
    for name in references:
        if Path(name).name != name or '/' in name or '\\' in name or not name.lower().endswith('.png'):
            raise ValueError('Invalid reference filename: ' + name)
    originals = {name: (source_path.parent / name).read_bytes() for name in references}
    stamp = time.strftime('%Y%m%d-%H%M%S')
    output = ROOT / 'output' / ('reworked-cards-300dpi-' + stamp)
    output.mkdir(parents=True, exist_ok=False)
    work = ROOT / '.browser-test'
    work.mkdir(exist_ok=True)
    http = urllib.request.build_opener(urllib.request.ProxyHandler({}))
    process = server = dev = None
    with tempfile.TemporaryDirectory(prefix='regenerate-', dir=work) as temporary, (output/'browser.log').open('w') as log:
        try:
            server = subprocess.Popen([str(ROOT/'dist/rework-studio.exe'), '--root', str(ROOT), '--port', '0'],
                                      stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True, creationflags=HIDDEN)
            line = server.stdout.readline().strip()
            if not line.startswith('Rework Studio: http'): raise RuntimeError(line)
            url = line.split(': ', 1)[1]
            profile = Path(temporary)/'profile'
            command = [browser, '--headless=new', '--disable-gpu', '--no-first-run', '--no-default-browser-check',
                       '--disable-background-networking', '--remote-debugging-port=0', '--force-device-scale-factor=1',
                       '--user-data-dir='+str(profile), 'about:blank']
            if args.single_process: command += ['--no-sandbox', '--single-process', '--in-process-gpu']
            process = subprocess.Popen(command, stdout=log, stderr=log, creationflags=HIDDEN)
            deadline = time.monotonic()+20
            while True:
                try:
                    port = int((profile/'DevToolsActivePort').read_text().splitlines()[0])
                    targets = json.loads(http.open(f'http://127.0.0.1:{port}/json', timeout=2).read())
                    break
                except (OSError, ValueError, IndexError):
                    if process.poll() is not None or time.monotonic()>deadline:
                        raise RuntimeError('Browser could not start; see '+str(output/'browser.log'))
                    time.sleep(.1)
            dev = DevTools(next(t for t in targets if t['type']=='page')['webSocketDebuggerUrl'])
            dev.call('Network.enable')
            dev.call('Network.setBlockedURLs', {'urls':['https://*']})
            dev.call('Page.navigate', {'url':url})
            deadline = time.monotonic()+20
            while True:
                ready = dev.call('Runtime.evaluate', {'expression':"document.readyState === 'complete' && !!document.getElementById('notice')"})
                if ready.get('result',{}).get('value'): break
                if time.monotonic()>deadline: raise RuntimeError('Studio failed to load')
                time.sleep(.1)
            result = dev.call('Runtime.evaluate', {'awaitPromise':True, 'returnByValue':True, 'expression':r'''
              (async () => {
                const {renderCard,png300,compressedRGB}=await import('/renderer.js');
                const reworks=await (await fetch('/api/reworks')).json(), cards=[], entries=[];
                const base64=blob=>new Promise((resolve,reject)=>{const reader=new FileReader();reader.onload=()=>resolve(reader.result.split(',')[1]);reader.onerror=reject;reader.readAsDataURL(blob);});
                for(const r of reworks){
                  const {canvas,warnings}=await renderCard(r.card);
                  const png=await png300(canvas),bitmap=await createImageBitmap(png);
                  const decoded=document.createElement('canvas');decoded.width=canvas.width;decoded.height=canvas.height;
                  decoded.getContext('2d').drawImage(bitmap,0,0);bitmap.close();
                  const pixels=canvas.getContext('2d').getImageData(0,0,canvas.width,canvas.height).data;
                  const roundtrip=decoded.getContext('2d').getImageData(0,0,canvas.width,canvas.height).data;
                  if(!pixels.every((v,i)=>v===roundtrip[i]))throw Error(r.card.name+': PNG changed rendered pixels');
                  const rgb=await compressedRGB(canvas);
                  cards.push({id:r.card.id,name:r.card.name,reference:r.reference,png:await base64(png),rgb,warnings});
                  entries.push({id:r.card.id,quantity:1,rgb});
                }
                const response=await fetch('/api/export',{method:'POST',headers:{'Content-Type':'application/json','X-UCPrint':'1'},body:JSON.stringify({cards:entries,cutMarks:true})});
                const pdf=await response.json();if(!response.ok)throw Error(pdf.error);
                return {cards,pdf};
              })()
            '''})
            if result.get('exceptionDetails'): raise RuntimeError(json.dumps(result['exceptionDetails']))
            value = result['result']['value']
            if [c['reference'] for c in value['cards']] != references: raise ValueError('Rendered collection differs from source')
            if value['pdf']['cards'] != len(reworks) or value['pdf']['pages'] != (len(reworks)+8)//9:
                raise ValueError('Incorrect PDF card or page count')
            pdf = http.open(url+value['pdf']['url']).read()
            streams = list(re.finditer(rb'<< /Length (\d+) [^>]* /Subtype /Image [^>]*>>\nstream\n', pdf))
            if len(streams) != len(reworks): raise ValueError('Missing PDF card images')
            for card, stream in zip(value['cards'], streams):
                png = base64.b64decode(card.pop('png'))
                validate_png(png)
                rgb = zlib.decompress(base64.b64decode(card.pop('rgb')))
                embedded = zlib.decompress(pdf[stream.end():stream.end()+int(stream[1])])
                if embedded != rgb or len(rgb) != 744*1039*3: raise ValueError('PDF pixels differ from rendered card')
                (output/card['reference']).write_bytes(png)
                card['sha256'] = hashlib.sha256(png).hexdigest()
                print('Rendered:', card['name'], '744 x 1039, 300 DPI', flush=True)
            (output/'reworked-cards-A4.pdf').write_bytes(pdf)
        finally:
            if dev: dev.sock.close()
            if process:
                if process.poll() is None: process.terminate()
                process.wait(timeout=10)
            if server:
                server.terminate(); server.wait(timeout=10); server.stdout.close()
    # Do not install a stale render if someone edited the collection meanwhile.
    if source_path.read_bytes() != source: raise RuntimeError('JSON changed during rendering; generated files were not installed')
    for name, original in originals.items():
        if (source_path.parent/name).read_bytes() != original: raise RuntimeError('Source PNG changed during rendering: '+name)
    backup = source_path.parent / ('originals-'+stamp)
    backup.mkdir(exist_ok=False)
    (backup/'reworks.json').write_bytes(source)
    for name, original in originals.items(): (backup/name).write_bytes(original)
    installed=[]
    try:
        for name in references:
            temporary=source_path.parent/('.regenerate-'+name)
            shutil.copyfile(output/name, temporary)
            os.replace(temporary, source_path.parent/name)
            installed.append(name)
    except Exception:
        for name in installed: shutil.copyfile(backup/name, source_path.parent/name)
        raise
    (output/'manifest.json').write_text(json.dumps({
        'source':'ReworkedCards/reworks.json','sourceSHA256':hashlib.sha256(source).hexdigest(),
        'rendererSHA256':hashlib.sha256((ROOT/'web/renderer.js').read_bytes()).hexdigest(),
        'width':744,'height':1039,'dpi':300,'backup':str(backup.relative_to(ROOT)),
        'cards':value['cards'],'pdf':'reworked-cards-A4.pdf',
        'note':'Rendered from reviewed JSON with tribe badges. Rules text remains plain; keyword colors are not implemented.'
    },indent=2)+'\n',encoding='utf-8')
    print('Installed',len(installed),'high-resolution PNGs in ReworkedCards.')
    print('Backup:',backup)
    print('PNG collection and A4 PDF:',output)


if __name__=='__main__': main()

"""Render every rework and export a PDF with the installed Chrome/Edge browser."""
import base64
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import time
import urllib.request

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / 'scripts'))
from browser_test import DevTools


def main():
    browser = next((Path(p) for p in [
        r'C:\Program Files\Google\Chrome\Application\chrome.exe',
        r'C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe'
    ] if Path(p).exists()), None)
    if browser is None:
        raise RuntimeError('Chrome or Edge is required for this check.')
    work = ROOT / '.browser-test'
    work.mkdir(exist_ok=True)
    folder = Path(tempfile.mkdtemp(prefix='reworks-', dir=work))
    output = ROOT / 'output'
    output.mkdir(exist_ok=True)
    flags = subprocess.CREATE_NO_WINDOW if os.name == 'nt' else 0
    server = subprocess.Popen([str(ROOT / 'dist/rework-studio.exe'), '--root', str(ROOT)],
                              stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True, creationflags=flags)
    process = dev = None
    log = (folder / 'browser.log').open('w')
    try:
        line = server.stdout.readline().strip()
        if not line.startswith('Rework Studio: http'):
            raise RuntimeError(line)
        url = line.split(': ', 1)[1]
        process = subprocess.Popen([str(browser), '--headless=new', '--disable-gpu', '--no-first-run',
            '--no-default-browser-check', '--disable-background-networking', '--remote-debugging-port=0',
            f'--user-data-dir={folder / "profile"}', '--window-size=1500,1100', 'about:blank'],
            stdout=log, stderr=log, creationflags=flags)
        portfile = folder / 'profile/DevToolsActivePort'
        deadline = time.monotonic() + 20
        while not portfile.exists():
            if time.monotonic() > deadline or process.poll() is not None:
                raise RuntimeError(f'Browser could not start. See {folder / "browser.log"}')
            time.sleep(.1)
        port = int(portfile.read_text().splitlines()[0])
        opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
        targets = json.loads(opener.open(f'http://127.0.0.1:{port}/json').read())
        dev = DevTools(next(t for t in targets if t['type'] == 'page')['webSocketDebuggerUrl'])
        dev.call('Page.navigate', {'url': url})
        result = dev.call('Runtime.evaluate', {'awaitPromise': True, 'returnByValue': True, 'expression': r'''
          (async () => {
            const wait = async fn => {const end=Date.now()+30000;while(!fn()){if(Date.now()>end)throw Error('UI timed out: '+document.body.innerText);await new Promise(r=>setTimeout(r,100));}};
            await wait(()=>document.querySelector('#preview canvas'));
            if(document.querySelectorAll('.card-row').length!==15)throw Error('Expected 15 cards');
            const {renderCard,png300,compressedRGB}=await import('/renderer.js');
            const reworks=await (await fetch('/api/reworks')).json(), entries=[];
            for(const r of reworks){
              const {canvas}=await renderCard(r.card);
              if(canvas.width!==744||canvas.height!==1039)throw Error('Wrong resolution');
              entries.push({id:r.card.id,quantity:1,rgb:await compressedRGB(canvas)});
            }
            const png=await png300((await renderCard(reworks[0].card)).canvas);
            const pngBase64=await new Promise(resolve=>{const fr=new FileReader();fr.onload=()=>resolve(fr.result.split(',')[1]);fr.readAsDataURL(png);});
            const response=await fetch('/api/export',{method:'POST',headers:{'Content-Type':'application/json','X-UCPrint':'1'},body:JSON.stringify({cards:entries,cutMarks:true})});
            const pdf=await response.json();if(!response.ok||pdf.cards!==15||pdf.pages!==2)throw Error(JSON.stringify(pdf));
            document.getElementById('all').click();
            if(!document.getElementById('total').textContent.startsWith('15 cards'))throw Error('Quantity selection failed');
            return {pngBase64,pdfURL:location.origin+pdf.url,cards:entries.length};
          })()
        '''})
        if result.get('exceptionDetails'):
            raise RuntimeError(json.dumps(result['exceptionDetails']))
        value = result['result']['value']
        (output / 'rework-sample.png').write_bytes(base64.b64decode(value['pngBase64']))
        (output / 'reworks-check.pdf').write_bytes(opener.open(value['pdfURL']).read())
        shot = dev.call('Page.captureScreenshot', {'format': 'png'})
        (output / 'rework-studio.png').write_bytes(base64.b64decode(shot['data']))
        print(f'PASS: {value["cards"]} cards rendered; PNG and two-page PDF exported; quantity controls work.')
    finally:
        if dev:
            try:
                dev.call('Browser.close')
            except (OSError, RuntimeError):
                pass
            dev.sock.close()
        if process:
            try:
                process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait()
        server.terminate()
        server.wait(timeout=5)
        server.stdout.close()
        log.close()


if __name__ == '__main__':
    main()

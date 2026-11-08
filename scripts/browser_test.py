"""Real Chrome/Edge UI smoke test, using the browser's DevTools protocol.

No Selenium, Playwright, npm install or browser download is required.
"""
import base64
import json
import os
from pathlib import Path
import shutil
import socket
import struct
import subprocess
import sys
import tempfile
import time
import urllib.parse
import urllib.request

ROOT=Path(__file__).resolve().parents[1]
sys.path.insert(0,str(ROOT))


class DevTools:
    def __init__(self,url):
        parsed=urllib.parse.urlsplit(url)
        self.sock=socket.create_connection((parsed.hostname,parsed.port),timeout=60)
        key=base64.b64encode(os.urandom(16)).decode()
        self.sock.sendall((f'GET {parsed.path} HTTP/1.1\r\nHost: {parsed.netloc}\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: {key}\r\nSec-WebSocket-Version: 13\r\n\r\n').encode())
        header=b''
        while not header.endswith(b'\r\n\r\n'):header+=self.sock.recv(1)
        if b' 101 ' not in header:raise RuntimeError('DevTools WebSocket refused the connection.')
        self.next_id=0

    def read(self,n):
        data=bytearray()
        while len(data)<n:
            chunk=self.sock.recv(n-len(data))
            if not chunk:raise RuntimeError('Browser closed its DevTools connection.')
            data.extend(chunk)
        return bytes(data)

    def call(self,method,params=None):
        self.next_id+=1
        payload=json.dumps(dict(id=self.next_id,method=method,params=params or {})).encode()
        n=len(payload);head=bytes([0x81,0x80|n]) if n<126 else bytes([0x81,0xfe])+struct.pack('!H',n) if n<65536 else bytes([0x81,0xff])+struct.pack('!Q',n)
        mask=os.urandom(4);self.sock.sendall(head+mask+bytes(v^mask[i%4] for i,v in enumerate(payload)))
        parts=b''
        while True:
            first,second=self.read(2);n=second&127
            if n==126:n=struct.unpack('!H',self.read(2))[0]
            if n==127:n=struct.unpack('!Q',self.read(8))[0]
            data=self.read(n)
            if first&15==8:raise RuntimeError('DevTools closed.')
            parts+=data
            if not first&128:continue
            message=json.loads(parts);parts=b''
            if message.get('id')!=self.next_id:continue
            if 'error' in message:raise RuntimeError(message['error'])
            return message.get('result',{})


def main():
    browsers=[shutil.which('google-chrome'),shutil.which('chromium'),
              r'C:\Program Files\Google\Chrome\Application\chrome.exe',
              r'C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe']
    browser=next((b for b in browsers if b and Path(b).exists()),None)
    if not browser:raise SystemExit('Chrome/Edge not found. Run the manual checklist in TODO.md.')
    output=ROOT/'output';output.mkdir(exist_ok=True)
    work=ROOT/'.browser-test';work.mkdir(exist_ok=True)
    with tempfile.TemporaryDirectory(dir=work) as temporary:
        folder=Path(temporary)
        executable=ROOT/(sys.argv[1] if len(sys.argv)>1 else 'dist/uc-print-demo.exe')
        if not executable.exists():raise RuntimeError('Build the offline demo first; see TODO.md.')
        server=subprocess.Popen([str(executable),'--port','0'],cwd=folder,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True,
                                creationflags=subprocess.CREATE_NO_WINDOW if os.name=='nt' else 0)
        line=server.stdout.readline().strip()
        if not line.startswith('UC Print offline: '):
            server.terminate();server.wait()
            raise RuntimeError('Offline executable failed: '+line)
        base_url=line.split('offline: ',1)[1]
        command=[browser,'--headless=new','--disable-gpu','--no-first-run','--no-default-browser-check',
                 '--remote-debugging-port=0',f'--user-data-dir={folder / "profile"}',
                 '--disable-background-networking','--window-size=1440,1150','about:blank']
        browser_log=(work/'browser.log').open('w')
        process=subprocess.Popen(command,stdout=browser_log,stderr=browser_log,
                                 creationflags=subprocess.CREATE_NO_WINDOW if os.name=='nt' else 0)
        dev=None
        try:
            portfile=folder/'profile'/'DevToolsActivePort'
            deadline=time.monotonic()+20
            while not portfile.exists():
                if time.monotonic()>deadline or process.poll() is not None:
                    raise RuntimeError('Headless browser could not start. See .browser-test/browser.log. '
                                       'If Windows denies process access, run python scripts/browser_test.py in a normal terminal or use TODO.md.')
                time.sleep(.1)
            port=int(portfile.read_text().splitlines()[0])
            opener=urllib.request.build_opener(urllib.request.ProxyHandler({}))
            targets=json.loads(opener.open(f'http://127.0.0.1:{port}/json').read())
            target=next(t for t in targets if t['type']=='page')
            dev=DevTools(target['webSocketDebuggerUrl'])
            dev.call('Browser.setDownloadBehavior',{'behavior':'deny'})
            dev.call('Network.enable')
            dev.call('Network.setBlockedURLs',{'urls':['https://*','http://undercards.net/*','http://*.undercards.net/*']})
            dev.call('Page.navigate',{'url':base_url})
            deadline=time.monotonic()+10
            while time.monotonic()<deadline:
                result=dev.call('Runtime.evaluate',{'expression':'document.readyState'})
                if result.get('result',{}).get('value')=='complete':
                    ready=dev.call('Runtime.evaluate',{'expression':'!!document.getElementById("notice")'})
                    if ready.get('result',{}).get('value'):break
                time.sleep(.1)
            test_script=ROOT/(sys.argv[2] if len(sys.argv)>2 else 'tests/browser.js')
            result=dev.call('Runtime.evaluate',{'expression':test_script.read_text(encoding='utf-8'),'awaitPromise':True,'returnByValue':True})
            if result.get('exceptionDetails'):raise RuntimeError(json.dumps(result['exceptionDetails'],indent=2))
            value=result['result']['value']
            for check in value['checks']:print('PASS:',check)
            prefix='reworked' if test_script.stem=='reworked-cards' else 'sample'
            (output/f'{prefix}-card.png').write_bytes(base64.b64decode(value['pngBase64']))
            pdf=opener.open(value['pdfURL']).read();(output/f'{prefix}-deck.pdf').write_bytes(pdf)
            (output/'calibration.pdf').write_bytes(opener.open(base_url+'/api/calibration').read())
            dev.call('Runtime.evaluate',{'expression':'window.scrollTo(0,0)'})
            screenshot=dev.call('Page.captureScreenshot',{'format':'png','captureBeyondViewport':False})
            (output/'app-screenshot.png').write_bytes(base64.b64decode(screenshot['data']))
            print(f'\n{len(value["checks"])} browser checks passed. Samples saved to output/.')
        finally:
            if dev:
                try:dev.call('Browser.close')
                except (OSError,RuntimeError):pass
                dev.sock.close()
            try:process.wait(timeout=5)
            except subprocess.TimeoutExpired:process.kill();process.wait()
            server.terminate();server.wait(timeout=5);server.stdout.close()
            browser_log.close()


if __name__=='__main__':main()

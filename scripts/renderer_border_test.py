"""Check source-frame borders in a packaged app at native browser scale factors.

No browser download or third-party Python dependency is required. --single-process
is for a trusted local test with an installed Chromium headless shell when Windows
denies browser subprocess IPC. It is not used by the application.
"""
import argparse
import base64
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile
import time
import urllib.request
import zlib

from browser_test import DevTools

ROOT = Path(__file__).resolve().parents[1]
HIDDEN = subprocess.CREATE_NO_WINDOW if os.name == 'nt' else 0


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--executable', default='dist/uc-print.exe')
    parser.add_argument('--browser')
    parser.add_argument('--single-process', action='store_true')
    parser.add_argument('--test-script', default='tests/renderer-border.js')
    parser.add_argument('--scales', nargs='+', type=float, default=[1, 1.25, 1.5, 1.75, 2])
    args = parser.parse_args()
    browsers = [args.browser, shutil.which('google-chrome'), shutil.which('chromium'),
                r'C:\Program Files\Google\Chrome\Application\chrome.exe',
                r'C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe']
    browser = next((path for path in browsers if path and Path(path).is_file()), None)
    if not browser:
        parser.error('Supply an installed Chromium browser using --browser.')
    output = ROOT / 'output' / ('border-tests' if args.test_script == 'tests/renderer-border.js' else Path(args.test_script).stem + '-tests')
    output.mkdir(parents=True, exist_ok=True)
    work = ROOT / '.browser-test'
    work.mkdir(exist_ok=True)
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
    reports = []
    with tempfile.TemporaryDirectory(dir=work) as temporary:
        server = subprocess.Popen([str((ROOT / args.executable).resolve()), '--port', '0'],
                                  cwd=temporary, stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                                  text=True, creationflags=HIDDEN)
        try:
            line = server.stdout.readline().strip()
            if not line.startswith('UC Print offline: '):
                raise RuntimeError('Packaged app failed to start: ' + line)
            url = line.split('offline: ', 1)[1]
            with opener.open(url + '/renderer.js') as response:
                packed = response.read()
            if packed != (ROOT / 'web/renderer.js').read_bytes():
                print('NOTE: This executable embeds a different renderer than web/renderer.js.', flush=True)
            for scale in args.scales:
                profile = Path(temporary) / ('profile-' + str(scale))
                command = [browser, '--headless=new', '--disable-gpu', '--no-first-run',
                           '--no-default-browser-check', '--disable-background-networking',
                           '--remote-debugging-port=0', '--user-data-dir=' + str(profile),
                           '--force-device-scale-factor=' + str(scale), 'about:blank']
                if args.single_process:
                    command += ['--no-sandbox', '--single-process', '--in-process-gpu']
                dev = None
                with (output / f'browser-{scale}.log').open('w') as log:
                    process = subprocess.Popen(command, stdout=log, stderr=log, creationflags=HIDDEN)
                    try:
                        deadline = time.monotonic() + 20
                        portfile = profile / 'DevToolsActivePort'
                        while True:
                            try:
                                port = int(portfile.read_text().splitlines()[0])
                                targets = json.loads(opener.open(f'http://127.0.0.1:{port}/json', timeout=2).read())
                                break
                            except (OSError, ValueError, IndexError):
                                if process.poll() is not None or time.monotonic() > deadline:
                                    raise RuntimeError(f'Browser failed to start; see {log.name}')
                                time.sleep(.1)
                        target = next(t for t in targets if t['type'] == 'page')
                        dev = DevTools(target['webSocketDebuggerUrl'])
                        dev.call('Network.enable')
                        dev.call('Network.setBlockedURLs', {'urls':['https://*']})
                        dev.call('Page.navigate', {'url':url})
                        deadline = time.monotonic() + 10
                        while True:
                            ready = dev.call('Runtime.evaluate', {'expression':"document.readyState === 'complete' && !!document.getElementById('notice')"})
                            if ready.get('result', {}).get('value'): break
                            if time.monotonic() > deadline: raise RuntimeError('App page failed to load')
                            time.sleep(.1)
                        result = dev.call('Runtime.evaluate', {
                            'expression':(ROOT / args.test_script).read_text(),
                            'awaitPromise':True, 'returnByValue':True})
                        if result.get('exceptionDetails'):
                            raise RuntimeError(f'Scale {scale}: ' + json.dumps(result['exceptionDetails']))
                        for card in result['result']['value']:
                            label = f'{int(scale*100)}-{card["id"]}{card.get("fileTag", "")}'
                            (output / (label + '.png')).write_bytes(base64.b64decode(card['png']))
                            request = urllib.request.Request(url + '/api/export',
                                data=json.dumps({'cards':[{'id':card['id'], 'quantity':1, 'rgb':card['rgb']}], 'cutMarks':True}).encode(),
                                headers={'Content-Type':'application/json', 'X-UCPrint':'1', 'Origin':url})
                            exported = json.loads(opener.open(request).read())
                            pdf = opener.open(url + exported['url']).read()
                            (output / (label + '.pdf')).write_bytes(pdf)
                            image = re.search(rb'<< /Length (\d+) [^>]* /Subtype /Image [^>]*>>\nstream\n', pdf)
                            if not image: raise RuntimeError('PDF image stream missing')
                            rgb = zlib.decompress(pdf[image.end():image.end()+int(image[1])])
                            if rgb != zlib.decompress(base64.b64decode(card['rgb'])):
                                raise RuntimeError('PDF changed card pixels')
                            reports.append({'scale':scale, 'id':card['id'], 'name':card['name'], 'edges':card['edges']})
                            print(f'PASS: {scale:g}x {card["name"]}: browser assertions and identical PNG/PDF pixels.', flush=True)
                    finally:
                        if dev: dev.sock.close()
                        if process.poll() is None: process.terminate()
                        process.wait(timeout=10)
            (output / 'report.json').write_text(json.dumps(reports, indent=2))
        finally:
            server.terminate()
            server.wait(timeout=10)
            server.stdout.close()


if __name__ == '__main__':
    main()

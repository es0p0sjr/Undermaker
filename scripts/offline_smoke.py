"""Exercise the embedded demo executable from an empty working directory.

All HTTP requests target the loopback listener. Requires Python 3.10+ for this
optional test only; the application itself needs no Python installation.
"""
import base64
import json
import io
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import urllib.error
import urllib.request
import zlib

ROOT = Path(__file__).resolve().parents[1]


def main():
    executable = ROOT / 'dist/uc-print-demo.exe'
    if not executable.exists():
        raise SystemExit('Build the demo using TODO.md first.')
    work = ROOT / '.test-data'
    work.mkdir(exist_ok=True)
    with tempfile.TemporaryDirectory(dir=work) as folder:
        process = subprocess.Popen([str(executable), '--port', '0'], cwd=folder,
                                   stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True,
                                   creationflags=subprocess.CREATE_NO_WINDOW if os.name == 'nt' else 0)
        try:
            line = process.stdout.readline().strip()
            assert line.startswith('UC Print offline: '), line
            origin = line.split('offline: ', 1)[1]
            assert origin.startswith('http://127.0.0.1:')
            opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))

            def request(path, body=None):
                req = urllib.request.Request(origin + path,
                    data=None if body is None else json.dumps(body).encode(),
                    headers={'Content-Type': 'application/json', 'X-UCPrint': '1', 'Origin': origin})
                try:
                    with opener.open(req, timeout=30) as response:
                        result = io.BytesIO(response.read())
                        result.status = response.status
                        result.headers = response.headers
                        return result
                except urllib.error.HTTPError:
                    raise
                except OSError as error:
                    raise RuntimeError(f'Local request failed: {path}: {error}') from error

            with request('/api/cards') as response:
                assert "connect-src 'self'" in response.headers['Content-Security-Policy']
                data = json.load(response)
            assert len(data['cards']) == 4 and data['manifest']['demo']
            assert all(c['name'] and c['description'] and c['image'] for c in data['cards'])
            for card in data['cards']:
                assert request(f'/api/art/{card["id"]}').read().startswith(b'\x89PNG')
            assert request('/api/power/Taunt.png').read().startswith(b'\x89PNG')
            for path in ['/', '/app.js', '/renderer.js', '/style.css', '/library.css']:
                assert request(path).status == 200
            bundle = json.load(request('/renderer-assets.json'))
            assert bundle['version'] == 1 and len(bundle['rarities']) == 21
            assert set(bundle['templates']) == {'MONSTER', 'SPELL'}
            assert bundle['font'].startswith('data:font/otf;base64,')
            assert all(icon.startswith('data:image/png;base64,') for icon in bundle['rarities'].values())
            assert request('/api/calibration').read().startswith(b'%PDF')
            ids = [900001] * 12 + [900002] * 12 + [900004] + [900003] * 4
            code = base64.b64encode(json.dumps({'soul': 'JUSTICE', 'cardIds': ids, 'artifactIds': []}).encode()).decode()
            deck = json.load(request('/api/deck', {'code': code}))
            assert sum(q['quantity'] for q in deck['queue']) == 29
            assert not any('Main deck has' in w for w in deck['warnings'])
            for path, body, status in [('/api/art/987654321', None, 404), ('/api/refresh', {}, 404),
                    ('/api/import', {}, 404), ('/api/deck', {'code': '{"cardIds":[987654321]}'}, 400)]:
                try:
                    request(path, body)
                    raise AssertionError(f'{path} unexpectedly succeeded')
                except urllib.error.HTTPError as error:
                    assert error.code == status
            # Test real PDF transport/layout with a deterministic RGB fixture.
            rgb = base64.b64encode(zlib.compress(bytes([32, 90, 120]) * (744 * 1039))).decode()
            exported = json.load(request('/api/export', {'cards': [{'id': 900001, 'quantity': 10, 'rgb': rgb}], 'cutMarks': True}))
            pdf = request(exported['url']).read()
            assert exported['pages'] == 2 and len(re.findall(rb'/Type /Page\b', pdf)) == 2
            assert b'/PrintScaling /None' in pdf
            assert not list(Path(folder).iterdir()), 'Runtime wrote unexpected files'
            print('PASS: Standalone embedded executable runs from an empty folder.')
            print('PASS: Local cards, rules, artwork, powers, frontend, deck import and PDF export.')
            print('PASS: Source templates, pixel font and 21 rarity icons are embedded locally.')
            print('PASS: Unknown cards fail; runtime download/import endpoints do not exist.')
        finally:
            process.terminate()
            process.wait(timeout=5)
            if sys.exc_info()[0] is not None:
                print(process.stdout.read(), file=sys.stderr)
            process.stdout.close()


if __name__ == '__main__':
    main()

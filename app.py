"""Compatibility launcher for the offline Go app."""
from pathlib import Path
import os
import subprocess
import sys
def main():
    root=Path(__file__).resolve().parent
    exe=root/'dist'/'uc-print.exe'
    if not exe.exists() and not (root/'data/manifest.json').exists() and '--data' not in sys.argv[1:]:
        print('UC Print setup is not complete.\n'
              'The importer creates manifest.json only after all cards, rules and images pass validation.\n'
              'Do not create it yourself; validate does not create a snapshot.\n\n'
              'Use JSON/download-assets.js on the Undercards website; UC Print does not need to be running.\n'
              'Follow tools/ucsync/BROWSER.md, then run:\n'
              '  go run ./tools/ucsync import ./JSON/AllCards.Json --translations ./JSON/en.json --assets ./JSON/assets\n\n'
              'Then start UC Print again. For a fictional print test now: .\\dist\\uc-print-demo.exe --open')
        return 1
    env=dict(os.environ,GOPROXY='off',GOTOOLCHAIN='local',GOCACHE=str(root/'.go-cache'))
    command=[str(exe)] if exe.exists() else ['go','run','./tools/ucdev']
    return subprocess.call(command+sys.argv[1:],cwd=root,env=env)

if __name__ == '__main__':
    raise SystemExit(main())

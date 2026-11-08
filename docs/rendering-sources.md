# Existing card rendering sources

Investigated after comparing `CardTest/OurSave.png` with `IntendedSave.png`. Prefer existing game styling and assets over manually approximating the frame.

The strongest verified public starting point is the community [UCProjects CardEditor](https://github.com/UCProjects/CardEditor). UCProjects describes itself as unaffiliated with Undercards; this is not evidence that its renderer is identical to today's game client.

| Resource | Verified contents |
| --- | --- |
| [undercards.css](https://github.com/UCProjects/CardEditor/blob/master/public/resources/undercards.css) | White table/cell borders, black panels, cyan cost, centered descriptions, monster stats and rarity-cell layout |
| [DTM-Mono.otf](https://github.com/UCProjects/CardEditor/blob/master/public/resources/DTM-Mono.otf) | Font referenced by the stylesheet |
| [rarity directory](https://github.com/UCProjects/CardEditor/tree/master/public/rarity) | TOKEN and other rarity icons, including DR/UTY variants |
| [card.js](https://github.com/UCProjects/CardEditor/blob/master/public/resources/modules/card.js) | Existing monster/spell HTML templates with separate name/cost cells, description area and rarity image |

The stylesheet provides an existing basis for several visible differences in our reference comparison. Exact agreement with `IntendedSave.png` remains untested. The inspected template/CSS does not establish the current game client's Loop overlay placement or decorative corner treatment. Capture those from the current game's actual rendering DOM/CSS and referenced assets through the explicit sync workflow rather than infer their dimensions from a screenshot.

Recommended integration sequence:

1. Acquire the existing styles, font, rarity icons and any additional frame resources through explicit tooling; preserve source URLs, versions and checksums.
2. Inspect reuse terms before bundling third-party code/fonts/assets. No license declaration was found in the inspected CardEditor package metadata or cached repository tree; public availability alone does not establish redistribution terms.
3. Render ACT Button locally using the source layout and compare against IntendedSave, including Loop, TOKEN, font and borders. Use the game client as the final visual reference.
4. Adapt the normalized card database to the source template, then render at print resolution and retain the exact physical PDF dimensions. Screen-resolution CSS/images are not themselves a print-ready export.
5. Include every required resource in snapshot validation and the embedded executable. No font CDN, live stylesheets or artwork requests at runtime.

The app renderer now uses these source templates and resources. `npm run build` runs the offline `ucsync renderer-bundle` command after snapshot validation. It verifies the source checksums and packages both templates, three stylesheets, the font and all 21 rarity images into `web/renderer-assets.json`, which the release build embeds with the frontend.

`web/renderer.js` fills the source template from normalized cards, isolates its CSS in a shadow root, fits text using the source's 12-to-7px range, and serializes the filled HTML through SVG into a 744 x 1039 canvas. Preview, PNG and PDF all use that same canvas. Artwork, powers, font and rarity images are embedded in the SVG; runtime requests are restricted to the local app. The normalized `extension` field selects BASE, DELTARUNE or UTY rarity variants without consulting raw AllCards data at runtime.

The offscreen measurement host cancels the browser's device pixel ratio before layout. At 125% native display/browser scale, Chromium otherwise measures the source's 2px borders as 1.6px, while SVG image layout uses 2px. That mismatch makes the captured card too short and removes the bottom black border. Dimensions are converted back to unzoomed CSS pixels before serialization; the measurement host is excluded from the SVG. This was reproduced with Policeblook using `--force-device-scale-factor=1.25`; DevTools device-metrics emulation alone did not reproduce it.

`python scripts/renderer_border_test.py` checks the packaged standard executable at 100%, 125%, 150%, 175% and 200% native browser scales. It checks continuous black strips and the white frame on all four sides of Policeblook and ACT Button, round-trips the PNG pixels, and verifies that the PDF embeds identical RGB pixels. An installed Chromium browser can be selected with `--browser`; the optional `--single-process` mode supports a local Chromium headless shell in this restricted Windows environment. Results and PNG/PDF samples are saved under `output/border-tests/`. The previous packaged renderer fails this regression at 125%; the corrected standard executable passes all five scales.

A small adapter places the existing status icons/counters at the top right of the artwork. That positioning is an approximation based on the supplied reference, not a recovered source rule. Tribe icons use the 26 original PNGs from the verified CardEditor archive, embedded in the renderer bundle. They sit at the lower right of the artwork, with multiple tribes side by side. Dataset names `Tem`, `Frog`, and `All monster tribes` map to `TEMMIE`, `FROGGIT`, and `ALL`. Unknown tribe icons fail visibly rather than being silently omitted. Decorative corners are still absent; exact current-game fidelity is not claimed.

`npm test` checks tribe mappings for the installed snapshot and reworked JSON, plus byte-for-byte preservation of the source icons. `python scripts/renderer_border_test.py --executable dist/uc-print.exe --test-script tests/renderer-tribes.js --scales 1 1.25` checks tribe placement and matching PNG/PDF pixels in a browser. Rework Studio reads the shared renderer and bundle from disk; restart/reload it to pick up the change. Complete imported PNGs remain unchanged; tribe rendering applies to cards rendered from JSON.

Acquisition is implemented as `go run ./tools/ucsync renderer-download`, or `renderer-import ZIP` for a browser-saved archive. The user successfully acquired 52 source files in `Sources/CardEditor`.

`go run ./tools/ucsync renderer-preview` now builds `CardTest/RendererComparison.html`. It verifies the source checksums, extracts the source monster/spell template without executing editor code, and embeds the exact source styles, font and TOKEN image alongside the local ACT Button data and unchanged reference screenshots. Source CSS is isolated from the comparison page using a shadow root. No card frame is manually redrawn. Editor input controls, the unused default tribe image and editor footer are removed for display; name, cost, description and image values are filled from the local dataset.

The older comparison remains useful for inspecting the unadapted source. For the current app, launch `start.bat`, which prefers the standard `dist/uc-print.exe`, and use the browser address it opens. Regular Chrome startup is blocked by Windows process/IPC permissions here, but the installed Chromium headless shell can run the border regression in single-process mode. The broader demo UI test has a separate long-name fixture that does not fit the source template; its full-suite result is not represented by the focused border test.

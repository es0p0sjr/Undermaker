# Snapshot adapter and source investigation

Only `tools/ucsync` makes HTTP requests to Undercards. The application uses `internal/snapshot` types and local immutable bytes. Source-format parsing runs during sync; it is not called by the application.

## Observed card format

A previously captured real response established `cards` as a JSON-encoded array (plain arrays are accepted too), `id`, `fixedId`, `typeCard` (0 monster, 1 spell), `hp`, `image`, `statuses`, tribes and a `soul` restriction object. That incomplete import and its artwork were removed at the user's request. A new complete snapshot has since been installed from the user-saved JSON, English translations and browser-downloaded assets, and embedded in `dist/uc-print.exe`.

Names and rules often come from `/translation/en.json`, especially `card-name-<fixedId>` and `card-<fixedId>`. The adapter normalizes `hp` to `health`, resolves known translation macros/plurals, preserves rules and status labels, converts HTML to plain text, and retains original fields under `raw`. Unknown macros, missing rules, missing names, duplicate IDs or missing images fail the sync. An explicitly empty description is valid for a card with no rules; an absent description requires translations.

Image references are explicit filenames. Extensionless references get `.png`; PNG/JPG/JPEG files are supported and fully decoded for validation. Card names are never guessed into artwork paths. Status images use their explicit `image` or their original `name`, preserving case. Images are fetched once per unique reference during sync or supplied through `--assets`.

## Primary source code consulted

- [CardEditor image downloader](https://github.com/UCProjects/CardEditor/blob/master/scripts/downloadImages.js): `/AllCards`, serialized `cards`, card image references and status-name image paths.
- [CardEditor resource downloader](https://github.com/UCProjects/CardEditor/blob/master/scripts/downloadResources.js): `/translation/en.json`, keyword and tribe translation keys.
- [UnderScript card helper](https://github.com/UCProjects/UnderScript/blob/master/src/utils/cardHelper.js): fixed IDs, localization and generated-card rarities.
- [Deck-code consumer by an UnderScript author](https://gist.github.com/feildmaster/c7ddffd9e7dc5120fe234ac744393c04): Base64 JSON deck fields, a separate artifact collection, artifact artwork under `images/artifacts/<image>.png`, and SOUL artwork under `images/souls/<SOUL>.png`.

These community tools are primary evidence for their own integration code, not a guarantee that the current game endpoints/schema remain unchanged. The new adapter has synthetic regression tests; the user-supplied 931-card official snapshot has now passed normalization, asset and manifest validation.

## Tokens, artifacts and SOULs

Tokens are included among cards and have TOKEN/GENERATED rarity. They use the normal renderer, appear in a separate queue section, and do not count toward the main deck's 25 cards.

The observed card array contains no complete artifact catalog. The deck-code integration above calls `underscript.user.getAllArtifacts()` separately; its source does not establish a standalone downloadable artifact endpoint. Do not invent an `/AllArtifacts` response or substitute a player's inventory. A verified full raw response is required before adding an artifact adapter.

The card `soul` field is a restriction (ID/name), not the SOUL's full printable gameplay rules. The integration establishes artwork paths and the seven SOUL names, but not a complete current rules dataset. No SOUL rules or images are guessed. The manifest explicitly records these components as unavailable; runtime retains supplied deck metadata and warns that reference cards are missing.

When sources are verified, acquisition must go through `ucsync`, preserve exact responses under `data/raw`, normalize separately, gather all referenced artwork, and participate in staging/validation and embedding. There must still be no runtime downloads. No optional component files are created prematurely.

## Normalized schema version 1

`cards.json` is an array with `id`, `name`, `cost`, `attack`, `health`, `description`, `rarity`, `type`, `image`, `statuses`, plus `tribes`, `soul`, original `raw` fields and UI metadata. A spell's unused attack/health can be null. Each status has `name`, `image`, and optional `counter`. Artwork paths are relative to the appropriate asset category.

`manifest.json` includes schemaVersion, ISO timestamp, source URL, card count, unique card-image count, unique power-image count, component availability and SHA-256 checksums. The fixture has `demo: true`; real data is never labelled as demo. Build-time validation checks exact raw input checksums. The executable omits raw inputs while retaining normalized data, required asset bytes and manifest checksums.

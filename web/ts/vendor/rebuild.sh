#!/usr/bin/env bash
# Maintainer-only script. Fetches the pinned upstream sources for the vendored
# browser libraries (Preact, Leaflet, Quill) via npm and copies each into
# web/static/vendor/ under a version-stamped filename, plus Preact's prebuilt ESM
# modules and .d.ts type stubs.
#
# Every served asset filename is version-stamped (e.g. leaflet-1.9.4.js,
# preact-10.29.7.module.js) from the installed package version, so a file's name
# records exactly which upstream release it came from. These filenames are
# referenced BY HAND from three places, which MUST be updated whenever a version
# bumps — this script prints the current names at the end as a reminder:
#   - web/static/index.html            (import map: preact, preact/hooks, preact/jsx-runtime)
#   - web/ts/components/MapPicker.tsx   (vendor/leaflet.js, vendor/leaflet.css)
#   - web/ts/components/RichEditor.tsx  (vendor/quill.js, vendor/quill.snow.css)
#
# NOT invoked by build.sh or CI. Run this by hand only when adding or updating a
# vendored library, then commit the regenerated files.
#
# npm only ever touches a throwaway node_modules here, installed with
# --ignore-scripts (no install-time lifecycle scripts). Leaflet and Quill ship
# browser-ready builds and Preact ships prebuilt ESM, so nothing is bundled — the
# assets are copied verbatim and build.sh / CI need neither npm nor this script.
#
# Requires on $PATH: npm, node.
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")"

VENDOR_DIR="$(pwd)"
BROWSER_OUT="$VENDOR_DIR/../../static/vendor"    # served, version-stamped assets
PREACT_OUT="$BROWSER_OUT/preact"                 # runtime ESM modules served to the browser
PREACT_TYPES="$VENDOR_DIR/preact"                # .d.ts type stubs (compile-time only)

# Read an installed dependency's version from its package.json. Used to stamp the
# vendored asset filenames so each file's name records its upstream release.
pkgver() { node -p "require('$VENDOR_DIR/node_modules/$1/package.json').version"; }

# Reconcile package-lock.json with package.json first (a no-op producing no diff
# when they're already in sync; it adds the missing entries after a dependency is
# added/bumped). Then do a clean, lock-pinned install. `npm ci` alone aborts on
# an out-of-sync lock.
npm install --package-lock-only --ignore-scripts
npm ci --ignore-scripts

PREACT_VER="$(pkgver preact)"
LEAFLET_VER="$(pkgver leaflet)"
QUILL_VER="$(pkgver quill)"

WORK_DIR="$(mktemp -d)"
trap 'rm -rf "$WORK_DIR"' EXIT

mkdir -p "$PREACT_OUT" "$BROWSER_OUT/images" \
         "$PREACT_TYPES/src" "$PREACT_TYPES/hooks/src" "$PREACT_TYPES/jsx-runtime/src"

# --- Preact: prebuilt ESM modules + type stubs -----------------------------
#
# Preact ships self-contained ESM (dist/*.module.js) plus its own .d.ts, so no
# bundling is needed — copy them verbatim. The runtime modules go to
# web/static/vendor/preact/ (served, version-stamped, loaded via the import map);
# the .d.ts go to web/ts/vendor/preact/ (compile-time only, resolved via the
# tsconfig `paths` entries, so they are NOT version-stamped). hooks.module.js and
# jsxRuntime.module.js import the bare specifier "preact", so the import map — not
# a relative path inside them — resolves it; the version-stamped filename only
# needs updating in the import map.

rm -f "$PREACT_OUT"/{preact,hooks,jsx-runtime}-*.module.js

PREACT_SRC="$VENDOR_DIR/node_modules/preact"
cp "$PREACT_SRC/dist/preact.module.js"                 "$PREACT_OUT/preact-$PREACT_VER.module.js"
cp "$PREACT_SRC/hooks/dist/hooks.module.js"            "$PREACT_OUT/hooks-$PREACT_VER.module.js"
cp "$PREACT_SRC/jsx-runtime/dist/jsxRuntime.module.js" "$PREACT_OUT/jsx-runtime-$PREACT_VER.module.js"

cp "$PREACT_SRC/src/index.d.ts"             "$PREACT_TYPES/src/index.d.ts"
cp "$PREACT_SRC/src/jsx.d.ts"               "$PREACT_TYPES/src/jsx.d.ts"
cp "$PREACT_SRC/src/dom.d.ts"               "$PREACT_TYPES/src/dom.d.ts"
cp "$PREACT_SRC/hooks/src/index.d.ts"       "$PREACT_TYPES/hooks/src/index.d.ts"
cp "$PREACT_SRC/jsx-runtime/src/index.d.ts" "$PREACT_TYPES/jsx-runtime/src/index.d.ts"

# Preact's .d.ts use extensionless relative imports; this repo's tsconfig uses
# Node16 module resolution, which requires explicit .js extensions. Add them.
cat > "$WORK_DIR/normalize-preact-dts.mjs" <<'EOF'
import { readFileSync, writeFileSync } from "node:fs";
const [typesDir] = process.argv.slice(2);
const edits = [
  [`${typesDir}/src/index.d.ts`, [["./jsx", "./jsx.js"], ["./dom", "./dom.js"]]],
  [`${typesDir}/jsx-runtime/src/index.d.ts`, [["../../src/jsx", "../../src/jsx.js"]]],
];
for (const [file, subs] of edits) {
  let s = readFileSync(file, "utf8");
  for (const [from, to] of subs) s = s.split(`from '${from}'`).join(`from '${to}'`);
  writeFileSync(file, s);
}
EOF
node "$WORK_DIR/normalize-preact-dts.mjs" "$PREACT_TYPES"

# --- Leaflet: browser build + CSS + marker/layer images --------------------
#
# leaflet.css references its images by the relative path images/… , so the image
# directory keeps its plain name (unversioned) sitting next to the version-stamped
# CSS; the relative reference resolves regardless of the CSS filename.
rm -f "$BROWSER_OUT"/leaflet-*.js "$BROWSER_OUT"/leaflet-*.css
LEAFLET_SRC="$VENDOR_DIR/node_modules/leaflet/dist"
cp "$LEAFLET_SRC/leaflet.js"   "$BROWSER_OUT/leaflet-$LEAFLET_VER.js"
cp "$LEAFLET_SRC/leaflet.css"  "$BROWSER_OUT/leaflet-$LEAFLET_VER.css"
cp "$LEAFLET_SRC"/images/*.png "$BROWSER_OUT/images/"

# --- Quill: browser build + snow theme CSS ---------------------------------
rm -f "$BROWSER_OUT"/quill-*.js "$BROWSER_OUT"/quill-*.css
QUILL_SRC="$VENDOR_DIR/node_modules/quill/dist"
cp "$QUILL_SRC/quill.js"       "$BROWSER_OUT/quill-$QUILL_VER.js"
cp "$QUILL_SRC/quill.snow.css" "$BROWSER_OUT/quill-$QUILL_VER.snow.css"

# --- Reminder: keep the hand-written references in sync ---------------------
cat <<EOF

Wrote version-stamped vendored assets under $BROWSER_OUT/:
  preact/preact-$PREACT_VER.module.js
  preact/hooks-$PREACT_VER.module.js
  preact/jsx-runtime-$PREACT_VER.module.js
  leaflet-$LEAFLET_VER.js
  leaflet-$LEAFLET_VER.css
  quill-$QUILL_VER.js
  quill-$QUILL_VER.snow.css

Reminder: these filenames are referenced by hand — update on a version bump:
  web/static/index.html (import map)
    preact             -> ./vendor/preact/preact-$PREACT_VER.module.js
    preact/hooks       -> ./vendor/preact/hooks-$PREACT_VER.module.js
    preact/jsx-runtime -> ./vendor/preact/jsx-runtime-$PREACT_VER.module.js
  web/ts/components/MapPicker.tsx
    vendor/leaflet.css -> vendor/leaflet-$LEAFLET_VER.css
    vendor/leaflet.js  -> vendor/leaflet-$LEAFLET_VER.js
  web/ts/components/RichEditor.tsx
    vendor/quill.snow.css -> vendor/quill-$QUILL_VER.snow.css
    vendor/quill.js       -> vendor/quill-$QUILL_VER.js
EOF

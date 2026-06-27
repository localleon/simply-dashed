#!/usr/bin/env sh
set -eu

root_dir="$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)"
vendor_dir="$root_dir/internal/server/static/vendor"

mkdir -p "$vendor_dir"

curl -fsSL https://cdnjs.cloudflare.com/ajax/libs/normalize/8.0.1/normalize.min.css -o "$vendor_dir/normalize-8.0.1.min.css"
curl -fsSL https://cdnjs.cloudflare.com/ajax/libs/milligram/1.4.1/milligram.min.css -o "$vendor_dir/milligram-1.4.1.min.css"
curl -fsSL https://unpkg.com/htmx.org@2.0.10/dist/htmx.min.js -o "$vendor_dir/htmx-2.0.10.min.js"

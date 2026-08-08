#!/bin/sh
set -eu

# The development compose file keeps node_modules in a named volume. Sync it
# when package-lock.json changes so an old volume cannot hide newly installed
# dependencies from the image.
lock_hash="$(sha256sum package-lock.json | awk '{print $1}')"
hash_file="node_modules/.can-i-do-it-lock-hash"

if [ ! -f "$hash_file" ] || [ "$(cat "$hash_file")" != "$lock_hash" ]; then
  npm ci
  printf '%s\n' "$lock_hash" > "$hash_file"
fi

exec "$@"
